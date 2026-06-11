package chain

import (
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/crypto"
)

// Selector returns the 4-byte function selector for a method and its input
// ABI types, e.g. Selector("setFeeBps", []string{"uint256"}).
func Selector(method string, inputs []string) []byte {
	sig := fmt.Sprintf("%s(%s)", method, strings.Join(inputs, ","))
	return crypto.Keccak256([]byte(sig))[:4]
}

// Pack builds calldata: the 4-byte selector followed by the ABI-encoded
// arguments. Argument Go types must match the ABI types (e.g. *big.Int for
// uint256, common.Address for address, bool for bool).
func Pack(method string, inputs []string, args ...any) ([]byte, error) {
	args = nonNil(args)
	encoded, err := packArgs(inputs, args...)
	if err != nil {
		return nil, fmt.Errorf("pack %s: %w", method, err)
	}
	return append(Selector(method, inputs), encoded...), nil
}

// Unpack decodes return data given the output ABI types.
func Unpack(outputs []string, data []byte) ([]any, error) {
	args, err := arguments(outputs)
	if err != nil {
		return nil, err
	}
	return args.Unpack(data)
}

func packArgs(inputs []string, args ...any) ([]byte, error) {
	if len(inputs) == 0 {
		return nil, nil
	}
	a, err := arguments(inputs)
	if err != nil {
		return nil, err
	}
	return a.Pack(args...)
}

func arguments(types []string) (abi.Arguments, error) {
	var out abi.Arguments
	for i, t := range types {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		ty, err := abi.NewType(t, "", nil)
		if err != nil {
			return nil, fmt.Errorf("abi type %d (%q): %w", i, t, err)
		}
		out = append(out, abi.Argument{Type: ty})
	}
	return out, nil
}

func nonNil(args []any) []any {
	if args == nil {
		return []any{}
	}
	return args
}
