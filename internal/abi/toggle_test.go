package abi_test

import (
	"testing"

	"github.com/aleksandarknezevic/chainform/internal/abi"
)

func TestBoolTogglePairs_Protocol(t *testing.T) {
	parsed, err := abi.Load("../../testdata/protocol.abi.json")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	pairs := abi.BoolTogglePairs(parsed)
	pair, ok := pairs["paused"]
	if !ok {
		t.Fatal("expected paused toggle pair")
	}
	if pair.On != "pause" || pair.Off != "unpause" {
		t.Errorf("pair = %+v, want pause/unpause", pair)
	}
}

func TestBoolTogglePairs_NoneWithoutMutators(t *testing.T) {
	parsed, err := abi.Load("../../testdata/weth.abi.json")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(abi.BoolTogglePairs(parsed)) != 0 {
		t.Fatal("expected no toggle pairs for WETH")
	}
}
