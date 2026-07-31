package resource_test

import (
	"context"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"

	"github.com/aleksandarknezevic/chainform/internal/chain"
	"github.com/aleksandarknezevic/chainform/internal/config"
	"github.com/aleksandarknezevic/chainform/internal/resource"
)

// Richer attribute types: dynamic and fixed-size arrays, bytes32, bytes, and an
// enum (uint8 on the wire), all driven by the vault ABI fixture.

const (
	vaultAddr    = "0x0000000000000000000000000000000000000020"
	vaultABIPath = "../../testdata/vault.abi.json"
	keeperA      = "0x1111111111111111111111111111111111111111"
	keeperB      = "0x2222222222222222222222222222222222222222"
	keeperC      = "0x4444444444444444444444444444444444444444"
	guardianA    = "0x3333333333333333333333333333333333333333"
	rootHex      = "0xa1b2c3d4e5f60718293a4b5c6d7e8f90a1b2c3d4e5f60718293a4b5c6d7e8f90"
)

func vaultConfig(spec, expect map[string]any) config.ResourceConfig {
	spec["abi"] = vaultABIPath
	return config.ResourceConfig{
		Type:    "contract",
		Name:    "vault",
		Address: vaultAddr,
		Spec:    spec,
		Expect:  expect,
	}
}

// vaultReader returns the on-chain state in exactly the Go types a live
// eth_call decodes to: typed slices, a byte array for bytes32, a byte slice for
// dynamic bytes, and a sized integer for the enum.
func vaultReader() *chain.MockReader {
	addr := common.HexToAddress(vaultAddr)
	var root [32]byte
	copy(root[:], common.FromHex(rootHex))
	return chain.NewMockReader().
		Set(addr, "keepers", []common.Address{common.HexToAddress(keeperA), common.HexToAddress(keeperB)}).
		Set(addr, "guardians", []common.Address{common.HexToAddress(guardianA)}).
		Set(addr, "merkleRoot", root).
		Set(addr, "tierCaps", [3]*big.Int{big.NewInt(1000), big.NewInt(5000), big.NewInt(10000)}).
		Set(addr, "mode", uint8(2)).
		Set(addr, "extraData", []byte{0xde, 0xad, 0xbe, 0xef})
}

func TestContractRichTypesDrift(t *testing.T) {
	res, err := resource.Build(vaultConfig(map[string]any{
		"keepers":    []any{keeperA, keeperB, keeperC}, // one keeper added
		"merkleRoot": rootHex,                          // unchanged
		"tierCaps":   []any{1000, 5000, 10000},         // unchanged
		"mode":       1,                                // Halted -> Active
		"extraData":  "0xdeadbeef",                     // unchanged
	}, nil))
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	cur, err := res.Refresh(context.Background(), vaultReader())
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	ops, err := res.Plan(cur)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	// Only the two drifted attributes produce operations, in sorted order.
	if len(ops) != 2 {
		t.Fatalf("got %d operations, want 2: %+v", len(ops), ops)
	}
	if ops[0].Method != "setKeepers" || ops[1].Method != "setMode" {
		t.Fatalf("methods = %q, %q; want setKeepers, setMode", ops[0].Method, ops[1].Method)
	}
	if got, ok := ops[0].Args[0].([]common.Address); !ok || len(got) != 3 {
		t.Errorf("setKeepers arg = %T(%v), want []common.Address of 3", ops[0].Args[0], ops[0].Args[0])
	}
	if got, ok := ops[1].Args[0].(uint8); !ok || got != 1 {
		t.Errorf("setMode arg = %T(%v), want uint8(1)", ops[1].Args[0], ops[1].Args[0])
	}
	for i, op := range ops {
		if _, err := chain.Pack(op.Method, op.Inputs, op.Args...); err != nil {
			t.Errorf("op[%d] (%s) Pack: %v", i, op.Method, err)
		}
	}
	if !strings.Contains(ops[0].Reason, keeperC) {
		t.Errorf("setKeepers reason = %q, want the new keeper listed", ops[0].Reason)
	}
}

func TestContractRichTypesNoDrift(t *testing.T) {
	res, err := resource.Build(vaultConfig(map[string]any{
		"keepers":    []any{keeperA, keeperB},
		"merkleRoot": rootHex,
		"tierCaps":   []any{1000, 5000, 10000},
		"mode":       2,
		"extraData":  "deadbeef", // the 0x prefix is optional
	}, nil))
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	cur, err := res.Refresh(context.Background(), vaultReader())
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	ops, err := res.Plan(cur)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(ops) != 0 {
		t.Fatalf("got %d operations, want none: %+v", len(ops), ops)
	}
}

func TestContractExpectArray(t *testing.T) {
	res, err := resource.Build(vaultConfig(map[string]any{}, map[string]any{
		"guardians": []any{guardianA, keeperC}, // one guardian too many
	}))
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	cur, err := res.Refresh(context.Background(), vaultReader())
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}

	// A getter-only array can be asserted but never reconciled.
	ops, err := res.Plan(cur)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(ops) != 0 {
		t.Fatalf("read-only assertion produced %d operations", len(ops))
	}

	asserter, ok := res.(resource.Asserter)
	if !ok {
		t.Fatal("contract resource does not implement Asserter")
	}
	assertions, err := asserter.Assert(cur)
	if err != nil {
		t.Fatalf("Assert: %v", err)
	}
	if len(assertions) != 1 || assertions[0].Satisfied() {
		t.Fatalf("assertions = %+v, want one unsatisfied", assertions)
	}
	if assertions[0].Type != "address[]" {
		t.Errorf("assertion type = %q, want address[]", assertions[0].Type)
	}
}

// A struct-valued getter/setter pair is not manageable, and Inspect must not
// try to read it: its type string cannot be decoded.
func TestContractRejectsStructAttribute(t *testing.T) {
	_, err := resource.Build(vaultConfig(map[string]any{"riskParams": 1}, nil))
	if err == nil || !strings.Contains(err.Error(), "riskParams") {
		t.Fatalf("Build with struct attribute: err = %v, want rejection", err)
	}
}

func TestContractInspectSkipsStructGetter(t *testing.T) {
	res, err := resource.Build(vaultConfig(map[string]any{}, nil))
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	ins, ok := res.(resource.Inspector)
	if !ok {
		t.Fatal("contract resource does not implement Inspector")
	}
	obs, err := ins.Inspect(context.Background(), vaultReader())
	if err != nil {
		t.Fatalf("Inspect: %v", err)
	}
	for _, o := range obs {
		if o.Name == "riskParams" {
			t.Fatal("Inspect read the struct-returning getter")
		}
	}
	// The rich-typed getters are all inspectable and render readably.
	want := map[string]string{
		"extraData":  "0xdeadbeef",
		"merkleRoot": rootHex,
		"tierCaps":   "[1000, 5000, 10000]",
		"keepers":    "[" + keeperA + ", " + keeperB + "]",
	}
	for _, o := range obs {
		if w, ok := want[o.Name]; ok {
			if got := resource.FormatValue(o.Value); got != w {
				t.Errorf("%s = %s, want %s", o.Name, got, w)
			}
			delete(want, o.Name)
		}
	}
	if len(want) != 0 {
		t.Errorf("getters missing from Inspect: %v", want)
	}
}

func TestContractRejectsMalformedRichValues(t *testing.T) {
	cases := []struct {
		name string
		spec map[string]any
		want string
	}{
		{name: "short bytes32", spec: map[string]any{"merkleRoot": "0xa1b2"}, want: "exactly 32 byte"},
		{name: "wrong fixed array length", spec: map[string]any{"tierCaps": []any{1, 2}}, want: "exactly 3 element"},
		{name: "scalar for array", spec: map[string]any{"keepers": keeperA}, want: "expected a list of address"},
		{name: "bad address element", spec: map[string]any{"keepers": []any{"nope"}}, want: "invalid address"},
		{name: "enum out of range", spec: map[string]any{"mode": 300}, want: "out of range"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := resource.Build(vaultConfig(tc.spec, nil))
			if err == nil {
				// uint8 range is enforced when the setter argument is built.
				cur, rerr := res.Refresh(context.Background(), vaultReader())
				if rerr != nil {
					t.Fatalf("Refresh: %v", rerr)
				}
				_, err = res.Plan(cur)
			}
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v, want mention of %q", err, tc.want)
			}
		})
	}
}
