package resource_test

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"

	"github.com/aleksandarknezevic/chainform/internal/chain"
	"github.com/aleksandarknezevic/chainform/internal/config"
	"github.com/aleksandarknezevic/chainform/internal/resource"
)

const (
	contractAddr = "0x0000000000000000000000000000000000000010"
	abiPath      = "../../testdata/protocol.abi.json"
	curOwner     = "0x1111111111111111111111111111111111111111"
	wantOwner    = "0x2222222222222222222222222222222222222222"
)

func contractConfig(spec map[string]any) config.ResourceConfig {
	spec["abi"] = abiPath
	return config.ResourceConfig{
		Type:    "contract",
		Name:    "proto",
		Address: contractAddr,
		Spec:    spec,
	}
}

// driftedReader reports state that differs from the desired config below so
// the resource must produce one operation per managed attribute.
func driftedReader() *chain.MockReader {
	addr := common.HexToAddress(contractAddr)
	return chain.NewMockReader().
		Set(addr, "feeBps", big.NewInt(50)).
		Set(addr, "paused", true).
		Set(addr, "owner", common.HexToAddress(curOwner))
}

func TestContractDetectsDrift(t *testing.T) {
	res, err := resource.Build(contractConfig(map[string]any{
		"feeBps": 30,
		"paused": false,
		"owner":  wantOwner,
	}))
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	cur, err := res.Refresh(context.Background(), driftedReader())
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}

	ops, err := res.Plan(cur)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(ops) != 3 {
		t.Fatalf("got %d operations, want 3: %+v", len(ops), ops)
	}

	// Operations are emitted in sorted attribute order: feeBps, owner, paused.
	want := []struct {
		method string
		input  string
	}{
		{"setFeeBps", "uint256"},
		{"setOwner", "address"},
		{"unpause", ""},
	}
	for i, w := range want {
		if ops[i].Method != w.method {
			t.Errorf("op[%d].Method = %q, want %q", i, ops[i].Method, w.method)
		}
		if w.input == "" {
			if len(ops[i].Inputs) != 0 {
				t.Errorf("op[%d].Inputs = %v, want none", i, ops[i].Inputs)
			}
		} else if len(ops[i].Inputs) != 1 || ops[i].Inputs[0] != w.input {
			t.Errorf("op[%d].Inputs = %v, want [%s]", i, ops[i].Inputs, w.input)
		}
		// Every operation must encode cleanly with the produced argument types;
		// this catches any mismatch between setterArg and the ABI encoder.
		if _, err := chain.Pack(ops[i].Method, ops[i].Inputs, ops[i].Args...); err != nil {
			t.Errorf("op[%d] Pack: %v", i, err)
		}
	}
}

func TestContractPausedUsesPause(t *testing.T) {
	res, err := resource.Build(contractConfig(map[string]any{"paused": true}))
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	cur, err := res.Refresh(context.Background(), chain.NewMockReader().
		Set(common.HexToAddress(contractAddr), "paused", false))
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	ops, err := res.Plan(cur)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(ops) != 1 {
		t.Fatalf("got %d operations, want 1", len(ops))
	}
	if ops[0].Method != "pause" {
		t.Errorf("method = %q, want pause", ops[0].Method)
	}
}

func TestContractNoDrift(t *testing.T) {
	res, err := resource.Build(contractConfig(map[string]any{
		"feeBps": 50,
		"paused": true,
		"owner":  curOwner,
	}))
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	cur, err := res.Refresh(context.Background(), driftedReader())
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	ops, err := res.Plan(cur)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(ops) != 0 {
		t.Fatalf("expected no drift, got %d operations: %+v", len(ops), ops)
	}
}

// Only declared attributes are read and managed; omitted ones are untouched.
func TestContractManagesOnlyDeclared(t *testing.T) {
	res, err := resource.Build(contractConfig(map[string]any{"feeBps": 30}))
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	cur, err := res.Refresh(context.Background(), driftedReader())
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if _, ok := cur["paused"]; ok {
		t.Error("paused was read despite not being declared")
	}
	ops, err := res.Plan(cur)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(ops) != 1 || ops[0].Method != "setFeeBps" {
		t.Fatalf("want single setFeeBps op, got %+v", ops)
	}
}

func TestContractUnknownAttribute(t *testing.T) {
	// name() has a getter but no setter, so it cannot be managed.
	_, err := resource.Build(contractConfig(map[string]any{"name": "Protocol"}))
	if err == nil {
		t.Fatal("expected error for unsettable attribute, got nil")
	}
}

const aggregatorABI = "../../testdata/aggregator.abi.json"

// A read-only contract (no setX setters) builds with no managed attributes and
// produces an empty plan, but its getters can still be inspected.
func TestContractReadOnly(t *testing.T) {
	res, err := resource.Build(config.ResourceConfig{
		Type:    "contract",
		Name:    "feed",
		Address: contractAddr,
		Spec:    map[string]any{"abi": aggregatorABI},
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	cur, err := res.Refresh(context.Background(), chain.NewMockReader())
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	ops, err := res.Plan(cur)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(ops) != 0 {
		t.Fatalf("read-only contract should produce no operations, got %d", len(ops))
	}
}

func TestContractInspect(t *testing.T) {
	res, err := resource.Build(config.ResourceConfig{
		Type:    "contract",
		Name:    "feed",
		Address: contractAddr,
		Spec:    map[string]any{"abi": aggregatorABI},
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	ins, ok := res.(resource.Inspector)
	if !ok {
		t.Fatal("contract resource does not implement Inspector")
	}

	addr := common.HexToAddress(contractAddr)
	reader := chain.NewMockReader().
		Set(addr, "decimals", uint8(8)).
		Set(addr, "description", "ETH / USD").
		Set(addr, "version", big.NewInt(4)).
		Set(addr, "latestAnswer", big.NewInt(372912340000)).
		Set(addr, "latestRound", big.NewInt(299)).
		Set(addr, "latestTimestamp", big.NewInt(1749816000)).
		Set(addr, "aggregator", common.HexToAddress(curOwner)).
		Set(addr, "owner", common.HexToAddress(wantOwner))

	obs, err := ins.Inspect(context.Background(), reader)
	if err != nil {
		t.Fatalf("Inspect: %v", err)
	}

	// Only zero-arg, single-output getters are reported (getAnswer takes an
	// argument; latestRoundData returns five values), and they are sorted.
	wantNames := []string{
		"aggregator", "decimals", "description", "latestAnswer",
		"latestRound", "latestTimestamp", "owner", "version",
	}
	if len(obs) != len(wantNames) {
		t.Fatalf("got %d observations, want %d: %+v", len(obs), len(wantNames), obs)
	}
	for i, name := range wantNames {
		if obs[i].Name != name {
			t.Errorf("observation[%d].Name = %q, want %q", i, obs[i].Name, name)
		}
	}

	byName := map[string]resource.Observation{}
	for _, o := range obs {
		byName[o.Name] = o
	}
	if got := resource.FormatValue(byName["description"].Value); got != `"ETH / USD"` {
		t.Errorf("description = %s, want %q", got, "ETH / USD")
	}
	if got := resource.FormatValue(byName["latestAnswer"].Value); got != "372912340000" {
		t.Errorf("latestAnswer = %s, want 372912340000", got)
	}
}

func TestContractAssert(t *testing.T) {
	res, err := resource.Build(config.ResourceConfig{
		Type:    "contract",
		Name:    "feed",
		Address: contractAddr,
		Spec:    map[string]any{"abi": aggregatorABI},
		Expect: map[string]any{
			"decimals":    8,             // matches on-chain
			"description": "WRONG / USD", // mismatches on-chain "ETH / USD"
		},
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	addr := common.HexToAddress(contractAddr)
	reader := chain.NewMockReader().
		Set(addr, "decimals", uint8(8)).
		Set(addr, "description", "ETH / USD")

	cur, err := res.Refresh(context.Background(), reader)
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}

	// A read-only contract never produces operations.
	ops, err := res.Plan(cur)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(ops) != 0 {
		t.Fatalf("expected no operations, got %d", len(ops))
	}

	as, ok := res.(resource.Asserter)
	if !ok {
		t.Fatal("contract does not implement Asserter")
	}
	assertions, err := as.Assert(cur)
	if err != nil {
		t.Fatalf("Assert: %v", err)
	}
	if len(assertions) != 2 {
		t.Fatalf("got %d assertions, want 2: %+v", len(assertions), assertions)
	}

	byAttr := map[string]resource.Assertion{}
	for _, a := range assertions {
		byAttr[a.Attr] = a
	}
	if !byAttr["decimals"].Satisfied() {
		t.Error("decimals assertion should be satisfied (8 == 8)")
	}
	if byAttr["description"].Satisfied() {
		t.Error("description assertion should fail (\"ETH / USD\" != \"WRONG / USD\")")
	}
}

// expect against an attribute with no getter in the ABI is an error.
func TestContractExpectUnknownGetter(t *testing.T) {
	_, err := resource.Build(config.ResourceConfig{
		Type:    "contract",
		Name:    "feed",
		Address: contractAddr,
		Spec:    map[string]any{"abi": aggregatorABI},
		Expect:  map[string]any{"nonexistent": 1},
	})
	if err == nil {
		t.Fatal("expected error for expect on a getter that does not exist")
	}
}

// The protocol resource does not support expect blocks.
func TestProtocolRejectsExpect(t *testing.T) {
	_, err := resource.Build(config.ResourceConfig{
		Type:    "protocol",
		Name:    "main",
		Address: contractAddr,
		Spec:    map[string]any{"feeBps": 30},
		Expect:  map[string]any{"feeBps": 30},
	})
	if err == nil {
		t.Fatal("expected error: protocol should reject expect blocks")
	}
}

func TestContractMissingABI(t *testing.T) {
	_, err := resource.Build(config.ResourceConfig{
		Type:    "contract",
		Name:    "proto",
		Address: contractAddr,
		Spec:    map[string]any{"feeBps": 30},
	})
	if err == nil {
		t.Fatal("expected error when abi attribute is missing, got nil")
	}
}
