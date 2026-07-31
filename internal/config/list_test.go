package config

import (
	"bytes"
	"testing"

	"github.com/zclconf/go-cty/cty"
)

// List attributes must reach the provider as []any of plain Go values,
// regardless of the surface syntax they were written in.
func TestParseListAttributes(t *testing.T) {
	raw := []byte(`
version = "1"

chain {
  chain_id = 1
}

resource "contract" "vault" {
  address = "0x0000000000000000000000000000000000000001"
  abi     = "vault.abi.json"

  keepers  = ["0x1111111111111111111111111111111111111111", "0x2222222222222222222222222222222222222222"]
  tierCaps = [1000, 5000, 10000]
  nested   = [[1, 2], [3]]

  expect {
    guardians = ["0x3333333333333333333333333333333333333333"]
  }
}
`)

	cfg, err := Parse(raw, "vault.hcl")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	spec := cfg.Resources[0].Spec

	keepers, ok := spec["keepers"].([]any)
	if !ok || len(keepers) != 2 {
		t.Fatalf("keepers = %#v, want []any of 2", spec["keepers"])
	}
	if keepers[0] != "0x1111111111111111111111111111111111111111" {
		t.Errorf("keepers[0] = %v", keepers[0])
	}

	caps, ok := spec["tierCaps"].([]any)
	if !ok || len(caps) != 3 || caps[2] != 10000 {
		t.Fatalf("tierCaps = %#v, want []any{1000, 5000, 10000}", spec["tierCaps"])
	}

	nested, ok := spec["nested"].([]any)
	if !ok || len(nested) != 2 {
		t.Fatalf("nested = %#v, want []any of 2", spec["nested"])
	}
	if inner, ok := nested[0].([]any); !ok || len(inner) != 2 || inner[1] != 2 {
		t.Errorf("nested[0] = %#v, want []any{1, 2}", nested[0])
	}

	expect, ok := cfg.Resources[0].Expect["guardians"].([]any)
	if !ok || len(expect) != 1 {
		t.Fatalf("expect.guardians = %#v, want []any of 1", cfg.Resources[0].Expect["guardians"])
	}
}

func TestParseJSONListAttributes(t *testing.T) {
	raw := []byte(`{
  "version": "1",
  "chain": { "chain_id": 1 },
  "resources": [
    {
      "type": "contract",
      "name": "vault",
      "address": "0x0000000000000000000000000000000000000001",
      "spec": {
        "abi": "vault.abi.json",
        "tierCaps": [1000, 5000, 10000]
      },
      "expect": { "guardians": ["0x3333333333333333333333333333333333333333"] }
    }
  ]
}`)

	cfg, err := Parse(raw, "vault.json")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	caps, ok := cfg.Resources[0].Spec["tierCaps"].([]any)
	if !ok || len(caps) != 3 || caps[0] != 1000 {
		t.Fatalf("tierCaps = %#v, want []any{1000, 5000, 10000}", cfg.Resources[0].Spec["tierCaps"])
	}
	if _, ok := cfg.Resources[0].Expect["guardians"].([]any); !ok {
		t.Fatalf("expect.guardians = %#v, want []any", cfg.Resources[0].Expect["guardians"])
	}
}

// Solidity structs are not supported anywhere in the pipeline, so an object
// value must be reported at load time rather than silently mis-encoded.
func TestParseRejectsObjectValues(t *testing.T) {
	hcl := []byte(`
version = "1"

chain {
  chain_id = 1
}

resource "contract" "vault" {
  address    = "0x0000000000000000000000000000000000000001"
  abi        = "vault.abi.json"
  riskParams = { maxLoss = 1, ltvBps = 5000 }
}
`)
	if _, err := Parse(hcl, "vault.hcl"); err == nil {
		t.Fatal("expected an error for an object attribute value")
	}

	json := []byte(`{
  "version": "1",
  "chain": { "chain_id": 1 },
  "resources": [
    { "type": "contract", "name": "v", "address": "0x0000000000000000000000000000000000000001",
      "spec": { "riskParams": { "maxLoss": 1 } } }
  ]
}`)
	if _, err := Parse(json, "vault.json"); err == nil {
		t.Fatal("expected an error for an object attribute value in JSON")
	}
}

// import writes list and bytes values; they must parse back to the same values
// so `import` still round-trips to a no-drift plan for richer types.
func TestWriteResourceRoundTripRichTypes(t *testing.T) {
	doc := ResourceDoc{
		ChainID: 1,
		Type:    "contract",
		Name:    "vault",
		Address: "0x0000000000000000000000000000000000000001",
		ABIPath: "vault.abi.json",
		Managed: map[string]cty.Value{
			"keepers": cty.TupleVal([]cty.Value{
				cty.StringVal("0x1111111111111111111111111111111111111111"),
				cty.StringVal("0x2222222222222222222222222222222222222222"),
			}),
			"tierCaps":   cty.TupleVal([]cty.Value{cty.NumberIntVal(1000), cty.NumberIntVal(5000)}),
			"merkleRoot": cty.StringVal("0xa1b2"),
			"empty":      cty.EmptyTupleVal,
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

	spec := cfg.Resources[0].Spec
	keepers, ok := spec["keepers"].([]any)
	if !ok || len(keepers) != 2 || keepers[1] != "0x2222222222222222222222222222222222222222" {
		t.Errorf("keepers = %#v", spec["keepers"])
	}
	if caps, ok := spec["tierCaps"].([]any); !ok || len(caps) != 2 || caps[0] != 1000 {
		t.Errorf("tierCaps = %#v", spec["tierCaps"])
	}
	if spec["merkleRoot"] != "0xa1b2" {
		t.Errorf("merkleRoot = %#v", spec["merkleRoot"])
	}
	if empty, ok := spec["empty"].([]any); !ok || len(empty) != 0 {
		t.Errorf("empty = %#v, want an empty list", spec["empty"])
	}
}
