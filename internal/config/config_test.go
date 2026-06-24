package config

import (
	"os"
	"testing"
)

func TestParseValid(t *testing.T) {
	t.Setenv("TEST_RPC", "https://rpc.example/key")
	raw := []byte(`
version = "1"

chain {
  name     = "ethereum"
  chain_id = 1
  rpc      = env("TEST_RPC")
}

resource "protocol" "main" {
  address = "0x0000000000000000000000000000000000000001"
  feeBps  = 30
  paused  = false
}
`)
	cfg, err := Parse(raw, "test.hcl")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if cfg.Chain.ChainID != 1 {
		t.Errorf("chain_id = %d, want 1", cfg.Chain.ChainID)
	}
	if cfg.Chain.RPC != "https://rpc.example/key" {
		t.Errorf("env() not resolved: %q", cfg.Chain.RPC)
	}
	if len(cfg.Resources) != 1 {
		t.Fatalf("resources = %d, want 1", len(cfg.Resources))
	}
	r := cfg.Resources[0]
	if r.Type != "protocol" || r.Name != "main" {
		t.Errorf("labels = %q/%q, want protocol/main", r.Type, r.Name)
	}
	if r.Address != "0x0000000000000000000000000000000000000001" {
		t.Errorf("address = %q", r.Address)
	}
	// address must not leak into the spec map.
	if _, ok := r.Spec["address"]; ok {
		t.Error("address should not appear in spec")
	}
	if r.Spec["feeBps"] != 30 {
		t.Errorf("feeBps = %v (%T), want 30 (int)", r.Spec["feeBps"], r.Spec["feeBps"])
	}
	if r.Spec["paused"] != false {
		t.Errorf("paused = %v, want false", r.Spec["paused"])
	}
}

func TestParseExpectBlock(t *testing.T) {
	raw := []byte(`
chain {
  chain_id = 1
}

resource "contract" "feed" {
  address = "0x0000000000000000000000000000000000000001"
  abi     = "feed.abi.json"

  expect {
    decimals    = 8
    description = "ETH / USD"
  }
}
`)
	cfg, err := Parse(raw, "test.hcl")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	r := cfg.Resources[0]
	if r.Spec["abi"] != "feed.abi.json" {
		t.Errorf("abi = %v, want feed.abi.json", r.Spec["abi"])
	}
	// expect attributes go into Expect, not Spec.
	if _, ok := r.Spec["decimals"]; ok {
		t.Error("expect attribute leaked into spec")
	}
	if r.Expect["decimals"] != 8 {
		t.Errorf("expect.decimals = %v, want 8", r.Expect["decimals"])
	}
	if r.Expect["description"] != "ETH / USD" {
		t.Errorf("expect.description = %v", r.Expect["description"])
	}
}

func TestParseJSONValid(t *testing.T) {
	raw := []byte(`
{
  "version": "1",
  "chain": {
    "name": "ethereum",
    "chain_id": 1,
    "rpc": "https://rpc.example/key"
  },
  "resources": [
    {
      "type": "contract",
      "name": "feed",
      "address": "0x0000000000000000000000000000000000000001",
      "spec": {
        "abi": "testdata/aggregator.abi.json",
        "foo": 42
      },
      "expect": {
        "decimals": 8,
        "description": "ETH / USD"
      }
    }
  ]
}
`)

	cfg, err := Parse(raw, "test.json")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if cfg.Chain.ChainID != 1 {
		t.Errorf("chain_id = %d, want 1", cfg.Chain.ChainID)
	}
	if len(cfg.Resources) != 1 {
		t.Fatalf("resources = %d, want 1", len(cfg.Resources))
	}
	r := cfg.Resources[0]
	if r.Type != "contract" || r.Name != "feed" {
		t.Errorf("labels = %q/%q, want contract/feed", r.Type, r.Name)
	}
	if r.Spec["abi"] != "testdata/aggregator.abi.json" {
		t.Errorf("spec.abi = %v", r.Spec["abi"])
	}
	if r.Spec["foo"] != 42 {
		t.Errorf("spec.foo = %v (%T), want 42 (int)", r.Spec["foo"], r.Spec["foo"])
	}
	if r.Expect["decimals"] != 8 {
		t.Errorf("expect.decimals = %v (%T), want 8 (int)", r.Expect["decimals"], r.Expect["decimals"])
	}
}

func TestParseJSONFlatResourceAttrs(t *testing.T) {
	raw := []byte(`
{
  "chain": { "chain_id": 1 },
  "resources": [
    {
      "type": "protocol",
      "name": "main",
      "address": "0x0000000000000000000000000000000000000001",
      "feeBps": 30,
      "paused": false
    }
  ]
}
`)

	cfg, err := Parse(raw, "test.json")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	r := cfg.Resources[0]
	if r.Spec["feeBps"] != 30 {
		t.Errorf("feeBps = %v (%T), want 30 (int)", r.Spec["feeBps"], r.Spec["feeBps"])
	}
	if r.Spec["paused"] != false {
		t.Errorf("paused = %v, want false", r.Spec["paused"])
	}
}

func TestParseSyntaxError(t *testing.T) {
	if _, err := Parse([]byte(`chain { = }`), "bad.hcl"); err == nil {
		t.Error("expected parse error")
	}
}

func TestValidateErrors(t *testing.T) {
	cases := map[string]Config{
		"missing chainId": {
			Resources: []ResourceConfig{{Type: "protocol", Name: "a", Address: "0x1"}},
		},
		"no resources": {
			Chain: Chain{ChainID: 1},
		},
		"duplicate name": {
			Chain: Chain{ChainID: 1},
			Resources: []ResourceConfig{
				{Type: "protocol", Name: "a", Address: "0x1"},
				{Type: "protocol", Name: "a", Address: "0x2"},
			},
		},
	}
	for name, cfg := range cases {
		t.Run(name, func(t *testing.T) {
			if err := cfg.Validate(); err == nil {
				t.Error("expected validation error, got nil")
			}
		})
	}
}

func TestLoadMissingFile(t *testing.T) {
	if _, err := Load(os.DevNull + "/nope.hcl"); err == nil {
		t.Error("expected error loading missing file")
	}
}
