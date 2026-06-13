package resource

import (
	"fmt"
	"math/big"
	"reflect"

	ethabi "github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

// The value layer bridges three representations of an attribute value:
//
//   - HCL-decoded desired values (bool, string, int) from the config loader,
//   - chain-decoded current values (bool, string, sized ints, *big.Int,
//     common.Address) returned by a chain.Reader, and
//   - the exact Go type the ABI encoder expects when packing a setter call.
//
// canonical() folds the first two into a single comparable form so drift can
// be detected regardless of which decoder produced the value; setterArg()
// produces the third for building operations. Together they let a generic,
// ABI-driven resource handle attributes without hand-written type code.

// canonical normalizes a value (from either HCL or the chain) into a single
// comparable form for the given ABI type:
//
//	bool    -> bool
//	string  -> string
//	address -> common.Address
//	int/uint-> *big.Int
//
// It is used for both the desired and current value so the two are always
// compared in the same representation.
func canonical(t ethabi.Type, v any) (any, error) {
	switch t.T {
	case ethabi.BoolTy:
		b, ok := v.(bool)
		if !ok {
			return nil, fmt.Errorf("expected bool, got %T", v)
		}
		return b, nil
	case ethabi.StringTy:
		s, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("expected string, got %T", v)
		}
		return s, nil
	case ethabi.AddressTy:
		switch a := v.(type) {
		case common.Address:
			return a, nil
		case string:
			if !common.IsHexAddress(a) {
				return nil, fmt.Errorf("invalid address %q", a)
			}
			return common.HexToAddress(a), nil
		default:
			return nil, fmt.Errorf("expected address, got %T", v)
		}
	case ethabi.IntTy, ethabi.UintTy:
		n, err := toBig(v)
		if err != nil {
			return nil, err
		}
		if t.T == ethabi.UintTy && n.Sign() < 0 {
			return nil, fmt.Errorf("%s must be non-negative, got %s", t.String(), n)
		}
		return n, nil
	default:
		return nil, fmt.Errorf("unsupported attribute type %q", t.String())
	}
}

// setterArg converts a canonical value into the exact Go type the ABI encoder
// expects for the given type, matching go-ethereum's type checking: integers
// of width 8/16/32/64 use native sized types, all other widths use *big.Int.
func setterArg(t ethabi.Type, canonicalVal any) (any, error) {
	switch t.T {
	case ethabi.BoolTy, ethabi.StringTy, ethabi.AddressTy:
		return canonicalVal, nil
	case ethabi.IntTy, ethabi.UintTy:
		n, ok := canonicalVal.(*big.Int)
		if !ok {
			return nil, fmt.Errorf("expected integer, got %T", canonicalVal)
		}
		if t.T == ethabi.UintTy {
			switch t.Size {
			case 8:
				return uint8(n.Uint64()), nil
			case 16:
				return uint16(n.Uint64()), nil
			case 32:
				return uint32(n.Uint64()), nil
			case 64:
				return n.Uint64(), nil
			default:
				return n, nil
			}
		}
		switch t.Size {
		case 8:
			return int8(n.Int64()), nil
		case 16:
			return int16(n.Int64()), nil
		case 32:
			return int32(n.Int64()), nil
		case 64:
			return n.Int64(), nil
		default:
			return n, nil
		}
	default:
		return nil, fmt.Errorf("unsupported attribute type %q", t.String())
	}
}

// valueEqual reports whether two canonical values are equal.
func valueEqual(a, b any) bool {
	an, aok := a.(*big.Int)
	bn, bok := b.(*big.Int)
	if aok && bok {
		return an.Cmp(bn) == 0
	}
	return reflect.DeepEqual(a, b)
}

// display renders a canonical value for a human-readable drift reason.
func display(v any) string {
	switch x := v.(type) {
	case *big.Int:
		return x.String()
	case common.Address:
		return x.Hex()
	default:
		return fmt.Sprintf("%v", x)
	}
}

// toBig converts any integer-kinded Go value (or *big.Int) into a *big.Int.
func toBig(v any) (*big.Int, error) {
	if n, ok := v.(*big.Int); ok {
		return new(big.Int).Set(n), nil
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return big.NewInt(rv.Int()), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return new(big.Int).SetUint64(rv.Uint()), nil
	default:
		return nil, fmt.Errorf("expected integer, got %T", v)
	}
}
