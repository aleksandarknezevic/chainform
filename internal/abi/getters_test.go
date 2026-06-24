package abi_test

import (
	"testing"

	"github.com/aleksandarknezevic/chainform/internal/abi"
)

func TestGetters_WETH(t *testing.T) {
	parsed, err := abi.Load("../../testdata/weth.abi.json")
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	getters := abi.Getters(parsed)

	// WETH has 4 zero-arg view functions with 1 output:
	// decimals, name, symbol, totalSupply
	want := []struct {
		name       string
		outputType string
	}{
		{"decimals", "uint8"},
		{"name", "string"},
		{"symbol", "string"},
		{"totalSupply", "uint256"},
	}

	if len(getters) != len(want) {
		t.Fatalf("got %d getters, want %d: %+v", len(getters), len(want), getters)
	}

	for i, w := range want {
		if getters[i].Name != w.name {
			t.Errorf("getter[%d].Name = %q, want %q", i, getters[i].Name, w.name)
		}
		if getters[i].OutputType != w.outputType {
			t.Errorf("getter[%d].OutputType = %q, want %q", i, getters[i].OutputType, w.outputType)
		}
	}
}
