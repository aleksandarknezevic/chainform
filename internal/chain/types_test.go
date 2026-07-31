package chain

import (
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestSupportedType(t *testing.T) {
	supported := []string{
		"bool", "string", "address", "uint8", "uint256", "int256",
		"bytes", "bytes4", "bytes32",
		"address[]", "uint256[]", "bool[]", "string[]", "bytes32[]",
		"uint256[3]", "address[][]",
	}
	for _, typ := range supported {
		if !SupportedType(typ) {
			t.Errorf("SupportedType(%q) = false, want true", typ)
		}
	}

	// Tuples cannot round-trip through a type string, and the remaining kinds
	// have no value representation in ChainForm.
	unsupported := []string{"", "tuple", "(uint256,bool)", "(uint256,bool)[]", "fixed128x18", "function", "nonsense"}
	for _, typ := range unsupported {
		if SupportedType(typ) {
			t.Errorf("SupportedType(%q) = true, want false", typ)
		}
	}
}

// A tuple type string parses as its first component in go-ethereum, so packing
// it would silently encode the wrong thing. It must be rejected instead.
func TestPackRejectsTupleType(t *testing.T) {
	_, err := Pack("setRiskParams", []string{"(uint256,bool)"}, big.NewInt(1))
	if err == nil || !strings.Contains(err.Error(), "unsupported type") {
		t.Fatalf("Pack with tuple input: err = %v, want unsupported type", err)
	}
	if _, err := Unpack([]string{"(uint256,bool)"}, make([]byte, 64)); err == nil {
		t.Fatal("Unpack with tuple output: expected error")
	}
}

func TestPackUnpackRichTypes(t *testing.T) {
	keepers := []common.Address{
		common.HexToAddress("0x1111111111111111111111111111111111111111"),
		common.HexToAddress("0x2222222222222222222222222222222222222222"),
	}
	data, err := Pack("setKeepers", []string{"address[]"}, keepers)
	if err != nil {
		t.Fatalf("Pack(address[]): %v", err)
	}
	out, err := Unpack([]string{"address[]"}, data[4:])
	if err != nil {
		t.Fatalf("Unpack(address[]): %v", err)
	}
	got, ok := out[0].([]common.Address)
	if !ok || len(got) != 2 || got[1] != keepers[1] {
		t.Fatalf("round-trip address[] = %#v, want %#v", out[0], keepers)
	}

	var root [32]byte
	copy(root[:], common.FromHex("0xa1b2c3d4"))
	data, err = Pack("setMerkleRoot", []string{"bytes32"}, root)
	if err != nil {
		t.Fatalf("Pack(bytes32): %v", err)
	}
	out, err = Unpack([]string{"bytes32"}, data[4:])
	if err != nil {
		t.Fatalf("Unpack(bytes32): %v", err)
	}
	if out[0].([32]byte) != root {
		t.Fatalf("round-trip bytes32 = %x, want %x", out[0], root)
	}

	caps := [3]*big.Int{big.NewInt(1), big.NewInt(2), big.NewInt(3)}
	data, err = Pack("setTierCaps", []string{"uint256[3]"}, caps)
	if err != nil {
		t.Fatalf("Pack(uint256[3]): %v", err)
	}
	out, err = Unpack([]string{"uint256[3]"}, data[4:])
	if err != nil {
		t.Fatalf("Unpack(uint256[3]): %v", err)
	}
	if got := out[0].([3]*big.Int); got[2].Int64() != 3 {
		t.Fatalf("round-trip uint256[3] = %v, want [1 2 3]", got)
	}
}
