package chain

import (
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

// SupportedType reports whether ChainForm can encode and decode values of the
// given Solidity ABI type string, e.g. "uint256", "bytes32", "address[]".
//
// Supported: bool, string, address, intN/uintN, bytes, bytesN, and slices or
// fixed-size arrays of any supported element type (including nested arrays).
// Not supported: tuples (structs), fixed-point numbers, and function types.
//
// Tuples deserve a note. A tuple's type string, e.g. "(uint256,bool)", cannot
// be turned back into an ABI type: go-ethereum's parser needs the component
// list, and given only the string it silently parses "(uint256,bool)" as
// "uint256". Everything in ChainForm describes calls by type string, so tuples
// are rejected up front rather than mis-decoded.
func SupportedType(t string) bool {
	t = strings.TrimSpace(t)
	if t == "" {
		return false
	}
	if strings.ContainsAny(t, "(),") || strings.HasPrefix(t, "tuple") {
		return false
	}
	ty, err := abi.NewType(t, "", nil)
	if err != nil {
		return false
	}
	return supported(ty)
}

// supported walks a parsed ABI type, descending into array element types.
func supported(t abi.Type) bool {
	switch t.T {
	case abi.BoolTy, abi.StringTy, abi.AddressTy, abi.IntTy, abi.UintTy,
		abi.BytesTy, abi.FixedBytesTy:
		return true
	case abi.SliceTy, abi.ArrayTy:
		return t.Elem != nil && supported(*t.Elem)
	default:
		return false
	}
}
