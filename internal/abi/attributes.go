package abi

import (
	"sort"
	"strings"

	ethabi "github.com/ethereum/go-ethereum/accounts/abi"
)

// Attribute is a managed contract attribute: a zero-argument getter paired
// with a single-argument setter that follows the conventional "setX" naming.
//
// The pairing is by convention: an attribute named "feeBps" is read by the
// getter feeBps() and written by the setter setFeeBps(...). The getter's
// output type and the setter's input type must match; pairs that disagree on
// type are not considered manageable attributes.
type Attribute struct {
	Name   string // attribute name, equal to the getter, e.g. "feeBps"
	Getter string // getter function name, e.g. "feeBps"
	Setter string // setter function name, e.g. "setFeeBps"
	Type   string // ABI type shared by the getter output and setter input
}

// Attributes derives the set of managed attributes from a parsed ABI by
// pairing each getter X() with a setter setX(T) of matching type. Getters
// without a corresponding setter (read-only values) and setters without a
// corresponding getter are omitted. The result is sorted by name.
func Attributes(parsed *ethabi.ABI) []Attribute {
	setters := make(map[string]Setter)
	for _, s := range Setters(parsed) {
		setters[s.Name] = s
	}

	var out []Attribute
	for _, g := range Getters(parsed) {
		s, ok := setters[SetterName(g.Name)]
		if !ok {
			continue
		}
		if s.InputType != g.OutputType {
			continue
		}
		out = append(out, Attribute{
			Name:   g.Name,
			Getter: g.Name,
			Setter: s.Name,
			Type:   g.OutputType,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out
}

// SetterName returns the conventional setter name for a getter, e.g.
// "feeBps" -> "setFeeBps". It returns "" for an empty getter name.
func SetterName(getter string) string {
	if getter == "" {
		return ""
	}
	return "set" + strings.ToUpper(getter[:1]) + getter[1:]
}
