package abi

import (
	"sort"

	ethabi "github.com/ethereum/go-ethereum/accounts/abi"

	"github.com/aleksandarknezevic/chainform/internal/chain"
)

// Getter describes a single zero-argument view/pure function that returns
// exactly one value — i.e. a readable on-chain attribute.
type Getter struct {
	Name       string // function name, e.g. "decimals"
	OutputType string // ABI type of the single return value, e.g. "uint8"
}

// Getters returns all getter candidates from a parsed ABI.
// A getter is a view/pure function with 0 inputs and exactly 1 output whose
// type ChainForm can decode (chain.SupportedType) — a getter returning a
// struct is skipped rather than read and mis-decoded.
func Getters(parsed *ethabi.ABI) []Getter {
	var out []Getter
	for _, method := range parsed.Methods {
		if !method.IsConstant() {
			continue
		}
		if len(method.Inputs) != 0 {
			continue
		}
		if len(method.Outputs) != 1 {
			continue
		}
		if !chain.SupportedType(method.Outputs[0].Type.String()) {
			continue
		}
		out = append(out, Getter{
			Name:       method.Name,
			OutputType: method.Outputs[0].Type.String(),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out
}
