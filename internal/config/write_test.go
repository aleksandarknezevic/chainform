package config

import (
	"bytes"
	"testing"

	"github.com/zclconf/go-cty/cty"
)

// WriteResource output must be valid input to Parse, and the round trip must
// preserve the managed attributes and expect block. This also guards the
// import → plan "no drift" contract: a managed value written out is read back
// unchanged.
func TestWriteResourceRoundTrip(t *testing.T) {
	doc := ResourceDoc{
		ChainName: "ethereum",
		ChainID:   1,
		Type:      "contract",
		Name:      "vault",
		Address:   "0x0000000000000000000000000000000000000001",
		ABIPath:   "vault.abi.json",
		Managed: map[string]cty.Value{
			"feeBps": cty.NumberIntVal(30),
			"paused": cty.False,
		},
		Expect: map[string]cty.Value{
			"owner":    cty.StringVal("0x2222222222222222222222222222222222222222"),
			"decimals": cty.NumberIntVal(8),
		},
	}

	var buf bytes.Buffer
	if err := WriteResource(&buf, doc); err != nil {
		t.Fatalf("WriteResource: %v", err)
	}

	cfg, err := Parse(buf.Bytes(), "imported.hcl")
	if err != nil {
		t.Fatalf("Parse generated config: %v\n---\n%s", err, buf.String())
	}

	if cfg.Chain.ChainID != 1 || cfg.Chain.Name != "ethereum" {
		t.Errorf("chain = %+v", cfg.Chain)
	}
	if len(cfg.Resources) != 1 {
		t.Fatalf("resources = %d, want 1", len(cfg.Resources))
	}
	r := cfg.Resources[0]
	if r.Type != "contract" || r.Name != "vault" {
		t.Errorf("labels = %q/%q", r.Type, r.Name)
	}
	if r.Address != doc.Address {
		t.Errorf("address = %q", r.Address)
	}
	if r.Spec["abi"] != "vault.abi.json" {
		t.Errorf("abi = %v", r.Spec["abi"])
	}
	if r.Spec["feeBps"] != 30 {
		t.Errorf("feeBps = %v (%T), want 30", r.Spec["feeBps"], r.Spec["feeBps"])
	}
	if r.Spec["paused"] != false {
		t.Errorf("paused = %v, want false", r.Spec["paused"])
	}
	if r.Expect["decimals"] != 8 {
		t.Errorf("expect.decimals = %v, want 8", r.Expect["decimals"])
	}
	if r.Expect["owner"] != "0x2222222222222222222222222222222222222222" {
		t.Errorf("expect.owner = %v", r.Expect["owner"])
	}
}

// A read-only contract (no managed attributes) still produces a valid config
// with just an expect block.
func TestWriteResourceExpectOnly(t *testing.T) {
	doc := ResourceDoc{
		ChainID: 11155111,
		Type:    "contract",
		Name:    "feed",
		Address: "0x694AA1769357215DE4FAC081bf1f309aDC325306",
		ABIPath: "aggregator.abi.json",
		Expect: map[string]cty.Value{
			"decimals": cty.NumberIntVal(8),
		},
	}
	var buf bytes.Buffer
	if err := WriteResource(&buf, doc); err != nil {
		t.Fatalf("WriteResource: %v", err)
	}
	cfg, err := Parse(buf.Bytes(), "feed.hcl")
	if err != nil {
		t.Fatalf("Parse: %v\n---\n%s", err, buf.String())
	}
	r := cfg.Resources[0]
	if len(r.Expect) != 1 || r.Expect["decimals"] != 8 {
		t.Errorf("expect = %v", r.Expect)
	}
}
