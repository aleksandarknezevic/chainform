package resource

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"github.com/aleksandarknezevic/chainform/internal/chain"
	"github.com/aleksandarknezevic/chainform/internal/config"
)

func init() {
	Register("protocol", newProtocol)
}

// protocolResource is the reference resource implementation. It manages two
// attributes — feeBps (uint256) and paused (bool) — and exists primarily to
// demonstrate the Resource contract end to end. Real resources will typically
// be driven by a contract ABI rather than hand-written getters/setters.
type protocolResource struct {
	name    string
	address common.Address
	desired protocolSpec
}

// protocolSpec uses pointers so that an attribute is only managed when it is
// explicitly declared in configuration. Unset attributes are left untouched.
type protocolSpec struct {
	feeBps *uint64
	paused *bool
}

func newProtocol(cfg config.ResourceConfig) (Resource, error) {
	if !common.IsHexAddress(cfg.Address) {
		return nil, fmt.Errorf("protocol %q: invalid address %q", cfg.Name, cfg.Address)
	}
	if len(cfg.Expect) > 0 {
		return nil, fmt.Errorf("protocol %q: expect blocks are only supported by the ABI-driven \"contract\" resource", cfg.Name)
	}
	spec, err := parseProtocolSpec(cfg.Spec)
	if err != nil {
		return nil, fmt.Errorf("protocol %q: %w", cfg.Name, err)
	}
	return &protocolResource{
		name:    cfg.Name,
		address: common.HexToAddress(cfg.Address),
		desired: spec,
	}, nil
}

func parseProtocolSpec(m map[string]any) (protocolSpec, error) {
	var s protocolSpec
	for k, v := range m {
		switch k {
		case "feeBps":
			n, err := toUint64(v)
			if err != nil {
				return s, fmt.Errorf("feeBps: %w", err)
			}
			s.feeBps = &n
		case "paused":
			b, ok := v.(bool)
			if !ok {
				return s, fmt.Errorf("paused: expected bool, got %T", v)
			}
			s.paused = &b
		default:
			return s, fmt.Errorf("unknown attribute %q", k)
		}
	}
	return s, nil
}

func (p *protocolResource) Type() string            { return "protocol" }
func (p *protocolResource) Name() string            { return p.name }
func (p *protocolResource) Address() common.Address { return p.address }

func (p *protocolResource) Refresh(ctx context.Context, r chain.Reader) (State, error) {
	state := State{}
	if p.desired.feeBps != nil {
		out, err := r.Read(ctx, chain.ViewCall{To: p.address, Method: "feeBps", Outputs: []string{"uint256"}})
		if err != nil {
			return nil, err
		}
		n, ok := out[0].(*big.Int)
		if !ok {
			return nil, fmt.Errorf("feeBps: unexpected return type %T", out[0])
		}
		state["feeBps"] = n.Uint64()
	}
	if p.desired.paused != nil {
		out, err := r.Read(ctx, chain.ViewCall{To: p.address, Method: "paused", Outputs: []string{"bool"}})
		if err != nil {
			return nil, err
		}
		b, ok := out[0].(bool)
		if !ok {
			return nil, fmt.Errorf("paused: unexpected return type %T", out[0])
		}
		state["paused"] = b
	}
	return state, nil
}

func (p *protocolResource) Plan(current State) ([]Operation, error) {
	var ops []Operation

	if p.desired.feeBps != nil {
		cur, _ := current["feeBps"].(uint64)
		if cur != *p.desired.feeBps {
			ops = append(ops, Operation{
				Resource: p.name,
				To:       p.address,
				Method:   "setFeeBps",
				Inputs:   []string{"uint256"},
				Args:     []any{new(big.Int).SetUint64(*p.desired.feeBps)},
				Value:    big.NewInt(0),
				Reason:   fmt.Sprintf("feeBps: %d -> %d", cur, *p.desired.feeBps),
			})
		}
	}

	if p.desired.paused != nil {
		cur, _ := current["paused"].(bool)
		if cur != *p.desired.paused {
			method := "unpause"
			if *p.desired.paused {
				method = "pause"
			}
			ops = append(ops, Operation{
				Resource: p.name,
				To:       p.address,
				Method:   method,
				Inputs:   []string{},
				Value:    big.NewInt(0),
				Reason:   fmt.Sprintf("paused: %v -> %v", cur, *p.desired.paused),
			})
		}
	}

	return ops, nil
}

// toUint64 coerces a decoded numeric value into a uint64. The config loader
// decodes plain integers as int; other numeric types are accepted defensively.
func toUint64(v any) (uint64, error) {
	switch n := v.(type) {
	case int:
		if n < 0 {
			return 0, fmt.Errorf("must be non-negative, got %d", n)
		}
		return uint64(n), nil
	case int64:
		if n < 0 {
			return 0, fmt.Errorf("must be non-negative, got %d", n)
		}
		return uint64(n), nil
	case uint64:
		return n, nil
	case float64:
		if n < 0 || n != float64(uint64(n)) {
			return 0, fmt.Errorf("must be a non-negative integer, got %v", n)
		}
		return uint64(n), nil
	default:
		return 0, fmt.Errorf("expected integer, got %T", v)
	}
}
