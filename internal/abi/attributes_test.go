package abi_test

import (
	"testing"

	"github.com/aleksandarknezevic/chainform/internal/abi"
)

func TestAttributes_Protocol(t *testing.T) {
	parsed, err := abi.Load("../../testdata/protocol.abi.json")
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	attrs := abi.Attributes(parsed)

	// feeBps, owner and paused each have a matching setX setter of the right
	// type. name() is a getter with no setter, so it is not a managed
	// attribute. Results are sorted by name.
	want := []abi.Attribute{
		{Name: "feeBps", Getter: "feeBps", Setter: "setFeeBps", Type: "uint256"},
		{Name: "owner", Getter: "owner", Setter: "setOwner", Type: "address"},
		{Name: "paused", Getter: "paused", Setter: "setPaused", Type: "bool"},
	}
	if len(attrs) != len(want) {
		t.Fatalf("got %d attributes, want %d: %+v", len(attrs), len(want), attrs)
	}
	for i, w := range want {
		if attrs[i] != w {
			t.Errorf("attribute[%d] = %+v, want %+v", i, attrs[i], w)
		}
	}
}

// WETH exposes getters (name, symbol, decimals, totalSupply) but no setX
// setters, so it yields no managed attributes.
func TestAttributes_WETH_None(t *testing.T) {
	parsed, err := abi.Load("../../testdata/weth.abi.json")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if attrs := abi.Attributes(parsed); len(attrs) != 0 {
		t.Fatalf("got %d attributes, want 0: %+v", len(attrs), attrs)
	}
}

// The Chainlink aggregator proxy exposes many getters but its only setters
// (transferOwnership, proposeAggregator, ...) do not follow the setX
// convention, so it yields no managed attributes — it is inspect-only.
func TestAttributes_Aggregator_None(t *testing.T) {
	parsed, err := abi.Load("../../testdata/aggregator.abi.json")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if attrs := abi.Attributes(parsed); len(attrs) != 0 {
		t.Fatalf("got %d attributes, want 0: %+v", len(attrs), attrs)
	}
	// But it does expose zero-arg, single-output getters for `show`.
	if got := len(abi.Getters(parsed)); got != 8 {
		t.Fatalf("got %d getters, want 8", got)
	}
}

func TestSetterName(t *testing.T) {
	cases := map[string]string{
		"feeBps": "setFeeBps",
		"owner":  "setOwner",
		"x":      "setX",
		"":       "",
	}
	for in, want := range cases {
		if got := abi.SetterName(in); got != want {
			t.Errorf("SetterName(%q) = %q, want %q", in, got, want)
		}
	}
}
