package resource

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math/big"
	"reflect"
	"strconv"
	"strings"

	ethabi "github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

// The value layer bridges three representations of an attribute value:
//
//   - HCL-decoded desired values (bool, string, int, []any) from the config
//     loader,
//   - chain-decoded current values (bool, string, sized ints, *big.Int,
//     common.Address, []byte, [N]byte, typed slices and arrays) returned by a
//     chain.Reader, and
//   - the exact Go type the ABI encoder expects when packing a setter call.
//
// canonical() folds the first two into a single comparable form so drift can
// be detected regardless of which decoder produced the value; setterArg()
// produces the third for building operations. Together they let a generic,
// ABI-driven resource handle attributes without hand-written type code.

// canonical normalizes a value (from either HCL or the chain) into a single
// comparable form for the given ABI type:
//
//	bool         -> bool
//	string       -> string
//	address      -> common.Address
//	int/uint     -> *big.Int
//	bytes/bytesN -> []byte
//	T[] / T[N]   -> []any of canonical T
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
	case ethabi.BytesTy:
		return toBytes(v, -1)
	case ethabi.FixedBytesTy:
		return toBytes(v, t.Size)
	case ethabi.SliceTy, ethabi.ArrayTy:
		return canonicalList(t, v)
	default:
		return nil, fmt.Errorf("unsupported attribute type %q", t.String())
	}
}

// canonicalList normalizes a slice or fixed-size array element by element.
// Both HCL lists ([]any) and chain-decoded typed slices/arrays (e.g.
// []common.Address, [3]*big.Int) are accepted; the result is always a []any of
// canonical element values.
func canonicalList(t ethabi.Type, v any) (any, error) {
	if t.Elem == nil {
		return nil, fmt.Errorf("unsupported attribute type %q", t.String())
	}
	rv := reflect.ValueOf(v)
	if !rv.IsValid() || (rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array) {
		return nil, fmt.Errorf("expected a list of %s, got %T", t.Elem.String(), v)
	}
	if t.T == ethabi.ArrayTy && rv.Len() != t.Size {
		return nil, fmt.Errorf("%s requires exactly %d element(s), got %d", t.String(), t.Size, rv.Len())
	}
	out := make([]any, rv.Len())
	for i := range out {
		cv, err := canonical(*t.Elem, rv.Index(i).Interface())
		if err != nil {
			return nil, fmt.Errorf("[%d]: %w", i, err)
		}
		out[i] = cv
	}
	return out, nil
}

// toBytes normalizes a bytes value into a []byte. Accepted inputs are a hex
// string (with or without the 0x prefix, as written in configuration), a
// []byte, or a fixed-size byte array as decoded from the chain. A size >= 0
// enforces the exact length required by bytesN.
func toBytes(v any, size int) ([]byte, error) {
	var b []byte
	switch x := v.(type) {
	case []byte:
		b = append([]byte(nil), x...)
	case string:
		s := strings.TrimPrefix(strings.TrimPrefix(x, "0x"), "0X")
		decoded, err := hex.DecodeString(s)
		if err != nil {
			return nil, fmt.Errorf("invalid hex bytes %q: %w", x, err)
		}
		b = decoded
	default:
		rv := reflect.ValueOf(v)
		if !rv.IsValid() ||
			(rv.Kind() != reflect.Array && rv.Kind() != reflect.Slice) ||
			rv.Type().Elem().Kind() != reflect.Uint8 {
			return nil, fmt.Errorf("expected bytes (hex string or byte array), got %T", v)
		}
		b = byteSlice(rv)
	}
	if size >= 0 && len(b) != size {
		return nil, fmt.Errorf("bytes%d requires exactly %d byte(s), got %d", size, size, len(b))
	}
	return b, nil
}

// setterArg converts a canonical value into the exact Go type the ABI encoder
// expects for the given type, matching go-ethereum's type checking: integers
// of width 8/16/32/64 use native sized types and all other widths *big.Int;
// bytesN uses [N]byte; lists use a typed slice or array.
func setterArg(t ethabi.Type, canonicalVal any) (any, error) {
	switch t.T {
	case ethabi.BoolTy, ethabi.StringTy, ethabi.AddressTy:
		return canonicalVal, nil
	case ethabi.BytesTy:
		b, ok := canonicalVal.([]byte)
		if !ok {
			return nil, fmt.Errorf("expected bytes, got %T", canonicalVal)
		}
		return b, nil
	case ethabi.FixedBytesTy:
		b, ok := canonicalVal.([]byte)
		if !ok {
			return nil, fmt.Errorf("expected bytes, got %T", canonicalVal)
		}
		if len(b) != t.Size {
			return nil, fmt.Errorf("bytes%d requires exactly %d byte(s), got %d", t.Size, t.Size, len(b))
		}
		// bytesN packs from [N]byte, whose length is part of its Go type.
		arr := reflect.New(t.GetType()).Elem()
		reflect.Copy(arr.Slice(0, t.Size), reflect.ValueOf(b))
		return arr.Interface(), nil
	case ethabi.SliceTy, ethabi.ArrayTy:
		return setterList(t, canonicalVal)
	case ethabi.IntTy, ethabi.UintTy:
		n, ok := canonicalVal.(*big.Int)
		if !ok {
			return nil, fmt.Errorf("expected integer, got %T", canonicalVal)
		}
		if t.T == ethabi.UintTy {
			switch t.Size {
			case 8:
				u, err := asUintN(n, 8)
				if err != nil {
					return nil, err
				}
				return uint8(u), nil
			case 16:
				u, err := asUintN(n, 16)
				if err != nil {
					return nil, err
				}
				return uint16(u), nil
			case 32:
				u, err := asUintN(n, 32)
				if err != nil {
					return nil, err
				}
				return uint32(u), nil
			case 64:
				u, err := asUintN(n, 64)
				if err != nil {
					return nil, err
				}
				return u, nil
			default:
				if n.Sign() < 0 {
					return nil, fmt.Errorf("%s out of range: %s", t.String(), n.String())
				}
				return n, nil
			}
		}
		switch t.Size {
		case 8:
			i, err := asIntN(n, 8)
			if err != nil {
				return nil, err
			}
			return int8(i), nil
		case 16:
			i, err := asIntN(n, 16)
			if err != nil {
				return nil, err
			}
			return int16(i), nil
		case 32:
			i, err := asIntN(n, 32)
			if err != nil {
				return nil, err
			}
			return int32(i), nil
		case 64:
			i, err := asIntN(n, 64)
			if err != nil {
				return nil, err
			}
			return i, nil
		default:
			return n, nil
		}
	default:
		return nil, fmt.Errorf("unsupported attribute type %q", t.String())
	}
}

// setterList builds the typed slice or array the ABI encoder expects
// (e.g. []common.Address for address[], [3]*big.Int for uint256[3]) from a
// canonical []any.
func setterList(t ethabi.Type, canonicalVal any) (any, error) {
	if t.Elem == nil {
		return nil, fmt.Errorf("unsupported attribute type %q", t.String())
	}
	items, ok := canonicalVal.([]any)
	if !ok {
		return nil, fmt.Errorf("expected a list of %s, got %T", t.Elem.String(), canonicalVal)
	}
	if t.T == ethabi.ArrayTy && len(items) != t.Size {
		return nil, fmt.Errorf("%s requires exactly %d element(s), got %d", t.String(), t.Size, len(items))
	}

	out := reflect.New(t.GetType()).Elem()
	if t.T == ethabi.SliceTy {
		out.Set(reflect.MakeSlice(t.GetType(), len(items), len(items)))
	}
	elemType := out.Type().Elem()
	for i, item := range items {
		ev, err := setterArg(*t.Elem, item)
		if err != nil {
			return nil, fmt.Errorf("[%d]: %w", i, err)
		}
		rv := reflect.ValueOf(ev)
		if !rv.IsValid() || !rv.Type().AssignableTo(elemType) {
			return nil, fmt.Errorf("[%d]: cannot use %T as %s", i, ev, t.Elem.String())
		}
		out.Index(i).Set(rv)
	}
	return out.Interface(), nil
}

func asUintN(n *big.Int, bits uint) (uint64, error) {
	if n.Sign() < 0 || n.BitLen() > int(bits) {
		return 0, fmt.Errorf("uint%d out of range: %s", bits, n.String())
	}
	return n.Uint64(), nil
}

func asIntN(n *big.Int, bits uint) (int64, error) {
	max := new(big.Int).Lsh(big.NewInt(1), bits-1)
	max.Sub(max, big.NewInt(1))
	min := new(big.Int).Neg(new(big.Int).Lsh(big.NewInt(1), bits-1))
	if n.Cmp(min) < 0 || n.Cmp(max) > 0 {
		return 0, fmt.Errorf("int%d out of range: %s", bits, n.String())
	}
	return n.Int64(), nil
}

// valueEqual reports whether two canonical values are equal: integers by
// numeric value, bytes byte-wise, lists element by element.
func valueEqual(a, b any) bool {
	switch av := a.(type) {
	case *big.Int:
		bv, ok := b.(*big.Int)
		return ok && av.Cmp(bv) == 0
	case []byte:
		bv, ok := b.([]byte)
		return ok && bytes.Equal(av, bv)
	case []any:
		bv, ok := b.([]any)
		if !ok || len(av) != len(bv) {
			return false
		}
		for i := range av {
			if !valueEqual(av[i], bv[i]) {
				return false
			}
		}
		return true
	}
	return reflect.DeepEqual(a, b)
}

// display renders a canonical value for a human-readable drift reason.
func display(v any) string {
	return formatValue(v, false)
}

// formatValue renders a decoded or canonical value as text. quoteStrings is
// set for state listings (`show`), where quoting makes empty or padded strings
// visible, and unset for inline drift reasons.
func formatValue(v any, quoteStrings bool) string {
	switch x := v.(type) {
	case nil:
		return "<none>"
	case *big.Int:
		return x.String()
	case common.Address:
		return x.Hex()
	case []byte:
		return hexString(x)
	case string:
		if quoteStrings {
			return strconv.Quote(x)
		}
		return x
	}

	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Slice, reflect.Array:
		if rv.Type().Elem().Kind() == reflect.Uint8 {
			return hexString(byteSlice(rv))
		}
		parts := make([]string, rv.Len())
		for i := range parts {
			parts[i] = formatValue(rv.Index(i).Interface(), quoteStrings)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	}
	return fmt.Sprintf("%v", v)
}

// hexString renders raw bytes as 0x-prefixed hex.
func hexString(b []byte) string {
	return "0x" + hex.EncodeToString(b)
}

// byteSlice copies a reflected byte slice or byte array into a []byte.
func byteSlice(rv reflect.Value) []byte {
	out := make([]byte, rv.Len())
	for i := range out {
		out[i] = byte(rv.Index(i).Uint())
	}
	return out
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
