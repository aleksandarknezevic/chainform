package resource

import (
	"context"
	"fmt"
	"math/big"
	"sort"

	ethabi "github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"

	"github.com/chainform/chainform/internal/abi"
	"github.com/chainform/chainform/internal/chain"
	"github.com/chainform/chainform/internal/config"
)

func init() {
	Register("contract", newContract)
}

// contractResource is a generic, ABI-driven resource. Rather than hand-writing
// getters and setters for each contract (as protocolResource does), it loads a
// contract ABI and derives the managed attributes from it: each declared spec
// attribute X is read via the getter X() and reconciled via the setter setX().
//
// This is what makes arbitrary contracts manageable without writing Go: point
// the resource at an ABI, declare the attributes you care about, and ChainForm
// figures out how to read and converge them.
type contractResource struct {
	name    string
	address common.Address
	attrs   map[string]managedAttr // attribute name -> derived getter/setter
	desired map[string]any         // attribute name -> canonical desired value
	expects map[string]expectAttr  // read-only assertion -> getter + expected
	getters []abi.Getter           // all readable getters, for `show` (sorted)
	toggles map[string]abi.TogglePair // bool getter -> pause/unpause-style pair
}

// managedAttr couples an ABI-derived attribute with its parsed type, kept so
// the resource does not re-parse the ABI type on every refresh and plan.
type managedAttr struct {
	abi.Attribute
	typ ethabi.Type
}

// expectAttr is a read-only assertion: a getter and the value it is expected
// to return. It has no setter, so drift is reported but never converged.
type expectAttr struct {
	getter abi.Getter
	typ    ethabi.Type
	want   any // canonical expected value
}

const abiAttr = "abi"

func newContract(cfg config.ResourceConfig) (Resource, error) {
	if !common.IsHexAddress(cfg.Address) {
		return nil, fmt.Errorf("contract %q: invalid address %q", cfg.Name, cfg.Address)
	}

	abiPath, ok := cfg.Spec[abiAttr].(string)
	if !ok || abiPath == "" {
		return nil, fmt.Errorf("contract %q: %q attribute (path to the ABI JSON file) is required", cfg.Name, abiAttr)
	}
	parsed, err := abi.Load(abiPath)
	if err != nil {
		return nil, fmt.Errorf("contract %q: %w", cfg.Name, err)
	}

	derived := make(map[string]abi.Attribute)
	for _, a := range abi.Attributes(parsed) {
		derived[a.Name] = a
	}
	getters := abi.Getters(parsed)
	gettersByName := make(map[string]abi.Getter, len(getters))
	for _, g := range getters {
		gettersByName[g.Name] = g
	}

	r := &contractResource{
		name:    cfg.Name,
		address: common.HexToAddress(cfg.Address),
		attrs:   make(map[string]managedAttr),
		desired: make(map[string]any),
		expects: make(map[string]expectAttr),
		getters: getters,
		toggles: abi.BoolTogglePairs(parsed),
	}

	for k, v := range cfg.Spec {
		if k == abiAttr {
			continue
		}
		a, ok := derived[k]
		if !ok {
			return nil, fmt.Errorf("contract %q: attribute %q has no %s()/%s(...) getter+setter pair in the ABI",
				cfg.Name, k, k, abi.SetterName(k))
		}
		typ, err := ethabi.NewType(a.Type, "", nil)
		if err != nil {
			return nil, fmt.Errorf("contract %q: attribute %q: %w", cfg.Name, k, err)
		}
		cv, err := canonical(typ, v)
		if err != nil {
			return nil, fmt.Errorf("contract %q: attribute %q: %w", cfg.Name, k, err)
		}
		r.attrs[k] = managedAttr{Attribute: a, typ: typ}
		r.desired[k] = cv
	}

	for k, v := range cfg.Expect {
		if _, dup := r.attrs[k]; dup {
			return nil, fmt.Errorf("contract %q: attribute %q is both managed and expected; declare it in one place", cfg.Name, k)
		}
		g, ok := gettersByName[k]
		if !ok {
			return nil, fmt.Errorf("contract %q: expect %q has no getter %s() in the ABI", cfg.Name, k, k)
		}
		typ, err := ethabi.NewType(g.OutputType, "", nil)
		if err != nil {
			return nil, fmt.Errorf("contract %q: expect %q: %w", cfg.Name, k, err)
		}
		cv, err := canonical(typ, v)
		if err != nil {
			return nil, fmt.Errorf("contract %q: expect %q: %w", cfg.Name, k, err)
		}
		r.expects[k] = expectAttr{getter: g, typ: typ, want: cv}
	}

	// A contract with no managed attributes is valid: it is read-only and
	// produces no operations, but its state can still be inspected via `show`
	// and checked against any `expect` assertions.
	return r, nil
}

func (c *contractResource) Type() string            { return "contract" }
func (c *contractResource) Name() string            { return c.name }
func (c *contractResource) Address() common.Address { return c.address }

func (c *contractResource) Refresh(ctx context.Context, r chain.Reader) (State, error) {
	state := State{}
	for _, name := range c.attrNames() {
		a := c.attrs[name]
		out, err := r.Read(ctx, chain.ViewCall{
			To:      c.address,
			Method:  a.Getter,
			Outputs: []string{a.Type},
		})
		if err != nil {
			return nil, err
		}
		if len(out) != 1 {
			return nil, fmt.Errorf("%s: getter %s returned %d values, want 1", name, a.Getter, len(out))
		}
		cv, err := canonical(a.typ, out[0])
		if err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		state[name] = cv
	}
	for _, name := range c.expectNames() {
		e := c.expects[name]
		out, err := r.Read(ctx, chain.ViewCall{
			To:      c.address,
			Method:  e.getter.Name,
			Outputs: []string{e.getter.OutputType},
		})
		if err != nil {
			return nil, err
		}
		if len(out) != 1 {
			return nil, fmt.Errorf("%s: getter %s returned %d values, want 1", name, e.getter.Name, len(out))
		}
		cv, err := canonical(e.typ, out[0])
		if err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		state[name] = cv
	}
	return state, nil
}

// Assert evaluates the read-only `expect` assertions against current state.
func (c *contractResource) Assert(current State) ([]Assertion, error) {
	out := make([]Assertion, 0, len(c.expects))
	for _, name := range c.expectNames() {
		e := c.expects[name]
		out = append(out, Assertion{
			Resource: c.name,
			Attr:     name,
			Type:     e.getter.OutputType,
			Expected: e.want,
			Actual:   current[name],
		})
	}
	return out, nil
}

func (c *contractResource) Plan(current State) ([]Operation, error) {
	var ops []Operation
	for _, name := range c.attrNames() {
		a := c.attrs[name]
		want := c.desired[name]
		if valueEqual(current[name], want) {
			continue
		}
		if op, ok, err := c.planBoolToggle(name, current[name], want); err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		} else if ok {
			ops = append(ops, op)
			continue
		}
		arg, err := setterArg(a.typ, want)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		ops = append(ops, Operation{
			Resource: c.name,
			To:       c.address,
			Method:   a.Setter,
			Inputs:   []string{a.Type},
			Args:     []any{arg},
			Value:    big.NewInt(0),
			Reason:   fmt.Sprintf("%s: %s -> %s", name, display(current[name]), display(want)),
		})
	}
	return ops, nil
}

// planBoolToggle emits a zero-arg pause/unpause-style operation when the ABI
// exposes a toggle pair for this bool attribute. Returns ok=false to fall back
// to the conventional setX(bool) setter.
func (c *contractResource) planBoolToggle(name string, current, want any) (Operation, bool, error) {
	pair, ok := c.toggles[name]
	if !ok {
		return Operation{}, false, nil
	}
	cur, ok := current.(bool)
	if !ok {
		return Operation{}, false, fmt.Errorf("expected bool current value, got %T", current)
	}
	desired, ok := want.(bool)
	if !ok {
		return Operation{}, false, fmt.Errorf("expected bool desired value, got %T", want)
	}
	method := pair.Off
	if desired {
		method = pair.On
	}
	return Operation{
		Resource: c.name,
		To:       c.address,
		Method:   method,
		Inputs:   []string{},
		Args:     nil,
		Value:    big.NewInt(0),
		Reason:   fmt.Sprintf("%s: %v -> %v", name, cur, desired),
	}, true, nil
}

// Inspect reads every getter derived from the ABI and reports its value,
// regardless of which attributes are managed. It implements Inspector so
// `chainform show` can print on-chain state for read-only contracts.
func (c *contractResource) Inspect(ctx context.Context, r chain.Reader) ([]Observation, error) {
	obs := make([]Observation, 0, len(c.getters))
	for _, g := range c.getters {
		out, err := r.Read(ctx, chain.ViewCall{
			To:      c.address,
			Method:  g.Name,
			Outputs: []string{g.OutputType},
		})
		if err != nil {
			return nil, fmt.Errorf("%s: %w", g.Name, err)
		}
		if len(out) != 1 {
			return nil, fmt.Errorf("%s: getter returned %d values, want 1", g.Name, len(out))
		}
		obs = append(obs, Observation{Name: g.Name, Type: g.OutputType, Value: out[0]})
	}
	return obs, nil
}

// attrNames returns the managed attribute names in a stable order so that
// refresh reads and planned operations are deterministic.
func (c *contractResource) attrNames() []string {
	names := make([]string, 0, len(c.attrs))
	for k := range c.attrs {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}

// expectNames returns the read-only assertion names in a stable order.
func (c *contractResource) expectNames() []string {
	names := make([]string, 0, len(c.expects))
	for k := range c.expects {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}
