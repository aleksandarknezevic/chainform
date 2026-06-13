package abi_test

import (
	"testing"

	"github.com/chainform/chainform/internal/abi"
)

func TestSetters_WETH(t *testing.T) {
	parsed, err := abi.Load("../../testdata/weth.abi.json")
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	setters := abi.Setters(parsed)

	// WETH's only single-argument state-mutating function is withdraw(uint256).
	// approve/transfer take two args and transferFrom takes three, so they are
	// not setter candidates; deposit takes none.
	if len(setters) != 1 {
		t.Fatalf("got %d setters, want 1: %+v", len(setters), setters)
	}
	if setters[0].Name != "withdraw" {
		t.Errorf("setter name = %q, want withdraw", setters[0].Name)
	}
	if setters[0].InputType != "uint256" {
		t.Errorf("setter input type = %q, want uint256", setters[0].InputType)
	}
}

func TestSetters_Protocol(t *testing.T) {
	parsed, err := abi.Load("../../testdata/protocol.abi.json")
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	setters := abi.Setters(parsed)

	want := map[string]string{
		"setFeeBps": "uint256",
		"setPaused": "bool",
		"setOwner":  "address",
	}
	if len(setters) != len(want) {
		t.Fatalf("got %d setters, want %d: %+v", len(setters), len(want), setters)
	}
	for _, s := range setters {
		typ, ok := want[s.Name]
		if !ok {
			t.Errorf("unexpected setter %q", s.Name)
			continue
		}
		if s.InputType != typ {
			t.Errorf("setter %q input type = %q, want %q", s.Name, s.InputType, typ)
		}
	}
}
