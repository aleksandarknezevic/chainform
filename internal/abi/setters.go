package abi

import (
	"sort"

	ethabi "github.com/ethereum/go-ethereum/accounts/abi"
)

// Setter describes a single state-mutating function that takes exactly one
// argument — i.e. a writable on-chain attribute.
type Setter struct {
	Name      string // function name, e.g. "setFeeBps"
	InputType string // ABI type of the single argument, e.g. "uint256"
}

// Setters returns all setter candidates from a parsed ABI.
// A setter is a non-constant (state-mutating) function with exactly 1 input.
func Setters(parsed *ethabi.ABI) []Setter {
	var out []Setter
	for _, method := range parsed.Methods {
		if method.IsConstant() {
			continue
		}
		if len(method.Inputs) != 1 {
			continue
		}
		out = append(out, Setter{
			Name:      method.Name,
			InputType: method.Inputs[0].Type.String(),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out
}
