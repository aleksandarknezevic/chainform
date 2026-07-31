package abi_test

import (
	"testing"

	"github.com/aleksandarknezevic/chainform/internal/abi"
)

const vaultABI = "../../testdata/vault.abi.json"

// The vault fixture exposes arrays, bytes32, bytes and an enum alongside a
// struct-valued getter/setter pair. The struct pair must be dropped: its type
// string cannot be turned back into an ABI type, so reading it would decode
// the wrong value.
func TestAttributes_RichTypesAndSkippedStruct(t *testing.T) {
	parsed, err := abi.Load(vaultABI)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	want := map[string]string{
		"extraData":  "bytes",
		"keepers":    "address[]",
		"merkleRoot": "bytes32",
		"mode":       "uint8",
		"tierCaps":   "uint256[3]",
	}

	attrs := abi.Attributes(parsed)
	if len(attrs) != len(want) {
		t.Fatalf("got %d attributes, want %d: %+v", len(attrs), len(want), attrs)
	}
	for _, a := range attrs {
		typ, ok := want[a.Name]
		if !ok {
			t.Errorf("unexpected managed attribute %q", a.Name)
			continue
		}
		if a.Type != typ {
			t.Errorf("attribute %q type = %q, want %q", a.Name, a.Type, typ)
		}
		if a.Setter != abi.SetterName(a.Name) {
			t.Errorf("attribute %q setter = %q, want %q", a.Name, a.Setter, abi.SetterName(a.Name))
		}
	}
}

func TestGettersSkipStructReturns(t *testing.T) {
	parsed, err := abi.Load(vaultABI)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	for _, g := range abi.Getters(parsed) {
		if g.Name == "riskParams" {
			t.Fatalf("struct-returning getter %q should be skipped", g.Name)
		}
	}
	// guardians is getter-only: readable, and assertable via `expect`.
	var found bool
	for _, g := range abi.Getters(parsed) {
		if g.Name == "guardians" && g.OutputType == "address[]" {
			found = true
		}
	}
	if !found {
		t.Error("guardians() address[] missing from getters")
	}
}

func TestSettersSkipStructArguments(t *testing.T) {
	parsed, err := abi.Load(vaultABI)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	for _, s := range abi.Setters(parsed) {
		if s.Name == "setRiskParams" {
			t.Fatalf("struct-taking setter %q should be skipped", s.Name)
		}
	}
}
