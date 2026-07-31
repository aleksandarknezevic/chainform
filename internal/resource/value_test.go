package resource

import (
	"math/big"
	"strings"
	"testing"

	ethabi "github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"

	"github.com/aleksandarknezevic/chainform/internal/chain"
)

func mustType(t *testing.T, typ string) ethabi.Type {
	t.Helper()
	at, err := ethabi.NewType(typ, "", nil)
	if err != nil {
		t.Fatalf("NewType(%q): %v", typ, err)
	}
	return at
}

func TestSetterArgUint8Range(t *testing.T) {
	typ := mustType(t, "uint8")

	arg, err := setterArg(typ, big.NewInt(255))
	if err != nil {
		t.Fatalf("setterArg(uint8, 255): %v", err)
	}
	if got, ok := arg.(uint8); !ok || got != 255 {
		t.Fatalf("arg = %T(%v), want uint8(255)", arg, arg)
	}

	if _, err := setterArg(typ, big.NewInt(256)); err == nil || !strings.Contains(err.Error(), "out of range") {
		t.Fatalf("expected out-of-range error for 256, got %v", err)
	}
	if _, err := setterArg(typ, big.NewInt(-1)); err == nil || !strings.Contains(err.Error(), "out of range") {
		t.Fatalf("expected out-of-range error for -1, got %v", err)
	}
}

func TestSetterArgInt8Range(t *testing.T) {
	typ := mustType(t, "int8")

	arg, err := setterArg(typ, big.NewInt(127))
	if err != nil {
		t.Fatalf("setterArg(int8, 127): %v", err)
	}
	if got, ok := arg.(int8); !ok || got != 127 {
		t.Fatalf("arg = %T(%v), want int8(127)", arg, arg)
	}

	arg, err = setterArg(typ, big.NewInt(-128))
	if err != nil {
		t.Fatalf("setterArg(int8, -128): %v", err)
	}
	if got, ok := arg.(int8); !ok || got != -128 {
		t.Fatalf("arg = %T(%v), want int8(-128)", arg, arg)
	}

	if _, err := setterArg(typ, big.NewInt(128)); err == nil || !strings.Contains(err.Error(), "out of range") {
		t.Fatalf("expected out-of-range error for 128, got %v", err)
	}
	if _, err := setterArg(typ, big.NewInt(-129)); err == nil || !strings.Contains(err.Error(), "out of range") {
		t.Fatalf("expected out-of-range error for -129, got %v", err)
	}
}

// A bytes32 written as hex in config and the [32]byte decoded from the chain
// must fold to the same canonical value, or every plan would report drift.
func TestCanonicalBytes32FromConfigAndChain(t *testing.T) {
	typ := mustType(t, "bytes32")
	hexVal := "0xa1b2c3d4e5f60718293a4b5c6d7e8f90a1b2c3d4e5f60718293a4b5c6d7e8f90"

	var onChain [32]byte
	copy(onChain[:], common.FromHex(hexVal))

	fromConfig, err := canonical(typ, hexVal)
	if err != nil {
		t.Fatalf("canonical(bytes32, hex string): %v", err)
	}
	fromChain, err := canonical(typ, onChain)
	if err != nil {
		t.Fatalf("canonical(bytes32, [32]byte): %v", err)
	}
	if !valueEqual(fromConfig, fromChain) {
		t.Fatalf("canonical values differ: %s vs %s", display(fromConfig), display(fromChain))
	}
	if got := display(fromConfig); got != hexVal {
		t.Errorf("display = %q, want %q", got, hexVal)
	}

	// bytesN is length-checked; an accidentally truncated root must not encode.
	if _, err := canonical(typ, "0xa1b2"); err == nil || !strings.Contains(err.Error(), "exactly 32 byte") {
		t.Errorf("short bytes32: err = %v, want length error", err)
	}
	if _, err := canonical(typ, "0xzz"); err == nil || !strings.Contains(err.Error(), "invalid hex") {
		t.Errorf("invalid hex: err = %v, want hex error", err)
	}
}

func TestCanonicalDynamicBytes(t *testing.T) {
	typ := mustType(t, "bytes")

	fromConfig, err := canonical(typ, "0xdeadbeef")
	if err != nil {
		t.Fatalf("canonical(bytes, hex string): %v", err)
	}
	fromChain, err := canonical(typ, []byte{0xde, 0xad, 0xbe, 0xef})
	if err != nil {
		t.Fatalf("canonical(bytes, []byte): %v", err)
	}
	if !valueEqual(fromConfig, fromChain) {
		t.Fatalf("canonical values differ: %s vs %s", display(fromConfig), display(fromChain))
	}
	// The 0x prefix is optional, and dynamic bytes have no length constraint.
	if _, err := canonical(typ, "deadbeefcafe"); err != nil {
		t.Errorf("canonical(bytes, unprefixed hex): %v", err)
	}
}

// Lists arrive as []any from HCL and as typed slices from the chain; both must
// canonicalize element by element into the same comparable form.
func TestCanonicalListFromConfigAndChain(t *testing.T) {
	typ := mustType(t, "address[]")

	fromConfig, err := canonical(typ, []any{
		"0x1111111111111111111111111111111111111111",
		"0x2222222222222222222222222222222222222222",
	})
	if err != nil {
		t.Fatalf("canonical(address[], []any): %v", err)
	}
	fromChain, err := canonical(typ, []common.Address{
		common.HexToAddress("0x1111111111111111111111111111111111111111"),
		common.HexToAddress("0x2222222222222222222222222222222222222222"),
	})
	if err != nil {
		t.Fatalf("canonical(address[], []common.Address): %v", err)
	}
	if !valueEqual(fromConfig, fromChain) {
		t.Fatalf("canonical values differ: %s vs %s", display(fromConfig), display(fromChain))
	}

	// Order and length are part of the value: both count as drift.
	reordered, err := canonical(typ, []any{
		"0x2222222222222222222222222222222222222222",
		"0x1111111111111111111111111111111111111111",
	})
	if err != nil {
		t.Fatalf("canonical(address[], reordered): %v", err)
	}
	if valueEqual(fromConfig, reordered) {
		t.Error("reordered list compared equal")
	}
	shorter, err := canonical(typ, []any{"0x1111111111111111111111111111111111111111"})
	if err != nil {
		t.Fatalf("canonical(address[], shorter): %v", err)
	}
	if valueEqual(fromConfig, shorter) {
		t.Error("shorter list compared equal")
	}

	if _, err := canonical(typ, "0x1111111111111111111111111111111111111111"); err == nil ||
		!strings.Contains(err.Error(), "expected a list of address") {
		t.Errorf("scalar for list type: err = %v, want list error", err)
	}
	if _, err := canonical(typ, []any{"not-an-address"}); err == nil || !strings.Contains(err.Error(), "[0]") {
		t.Errorf("bad element: err = %v, want indexed error", err)
	}
}

func TestCanonicalFixedArrayLength(t *testing.T) {
	typ := mustType(t, "uint256[3]")

	if _, err := canonical(typ, []any{1, 2, 3}); err != nil {
		t.Fatalf("canonical(uint256[3], 3 elements): %v", err)
	}
	if _, err := canonical(typ, []any{1, 2}); err == nil || !strings.Contains(err.Error(), "exactly 3 element") {
		t.Errorf("wrong length: err = %v, want length error", err)
	}
}

// setterArg must produce exactly the Go types the ABI encoder accepts; packing
// the result is the real assertion.
func TestSetterArgRichTypesPack(t *testing.T) {
	cases := []struct {
		typ     string
		value   any
		wantGo  string
		method  string
		wantLen int // calldata length in bytes, selector included
	}{
		{typ: "address[]", value: []any{"0x1111111111111111111111111111111111111111"},
			wantGo: "[]common.Address", method: "setKeepers", wantLen: 4 + 32*3},
		{typ: "uint256[3]", value: []any{1, 2, 3},
			wantGo: "[3]*big.Int", method: "setTierCaps", wantLen: 4 + 32*3},
		{typ: "bytes32", value: "0xa1b2c3d4e5f60718293a4b5c6d7e8f90a1b2c3d4e5f60718293a4b5c6d7e8f90",
			wantGo: "[32]uint8", method: "setMerkleRoot", wantLen: 4 + 32},
		{typ: "bytes", value: "0xdeadbeef",
			wantGo: "[]uint8", method: "setExtraData", wantLen: 4 + 32*3},
		{typ: "uint8", value: 2, wantGo: "uint8", method: "setMode", wantLen: 4 + 32},
	}

	for _, tc := range cases {
		typ := mustType(t, tc.typ)
		cv, err := canonical(typ, tc.value)
		if err != nil {
			t.Fatalf("canonical(%s): %v", tc.typ, err)
		}
		arg, err := setterArg(typ, cv)
		if err != nil {
			t.Fatalf("setterArg(%s): %v", tc.typ, err)
		}
		if got := goTypeName(arg); got != tc.wantGo {
			t.Errorf("%s: setter arg type = %s, want %s", tc.typ, got, tc.wantGo)
		}
		data, err := chain.Pack(tc.method, []string{tc.typ}, arg)
		if err != nil {
			t.Fatalf("Pack(%s): %v", tc.typ, err)
		}
		if len(data) != tc.wantLen {
			t.Errorf("%s: calldata length = %d, want %d", tc.typ, len(data), tc.wantLen)
		}
	}
}

func TestFormatValueRichTypes(t *testing.T) {
	cases := []struct {
		in   any
		want string
	}{
		{in: []byte{0xde, 0xad}, want: "0xdead"},
		{in: [4]byte{0xca, 0xfe, 0x00, 0x01}, want: "0xcafe0001"},
		{in: []common.Address{common.HexToAddress("0x1111111111111111111111111111111111111111")},
			want: "[0x1111111111111111111111111111111111111111]"},
		{in: []any{big.NewInt(1), big.NewInt(2)}, want: "[1, 2]"},
		{in: [3]*big.Int{big.NewInt(1), big.NewInt(5), big.NewInt(9)}, want: "[1, 5, 9]"},
		{in: "ETH / USD", want: `"ETH / USD"`},
		{in: nil, want: "<none>"},
	}
	for _, tc := range cases {
		if got := FormatValue(tc.in); got != tc.want {
			t.Errorf("FormatValue(%v) = %q, want %q", tc.in, got, tc.want)
		}
	}
	// Drift reasons render inline, without quoting strings.
	if got := display("ETH / USD"); got != "ETH / USD" {
		t.Errorf("display(string) = %q, want unquoted", got)
	}
}

func goTypeName(v any) string {
	switch v.(type) {
	case []common.Address:
		return "[]common.Address"
	case [3]*big.Int:
		return "[3]*big.Int"
	case [32]uint8:
		return "[32]uint8"
	case []uint8:
		return "[]uint8"
	case uint8:
		return "uint8"
	default:
		return "unknown"
	}
}
