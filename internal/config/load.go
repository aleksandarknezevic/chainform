package config

import (
	"fmt"
	"os"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

// hclRoot mirrors the HCL document structure for decoding. The public Config
// type is intentionally decoupled from this so the rest of the codebase never
// depends on HCL.
type hclRoot struct {
	Version   string        `hcl:"version,optional"`
	Chain     hclChain      `hcl:"chain,block"`
	Resources []hclResource `hcl:"resource,block"`
}

type hclChain struct {
	Name    string `hcl:"name,optional"`
	ChainID uint64 `hcl:"chain_id"`
	RPC     string `hcl:"rpc,optional"`
}

type hclResource struct {
	Type string   `hcl:"type,label"`
	Name string   `hcl:"name,label"`
	Body hcl.Body `hcl:",remain"` // address, type-specific attributes, expect block
}

// Load reads, parses, and validates a configuration file.
func Load(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}
	return Parse(raw, path)
}

// Parse decodes and validates an HCL configuration document. filename is used
// only for diagnostics. The env("VAR") function is available to resolve
// environment variables (e.g. for RPC endpoints) at load time.
func Parse(raw []byte, filename string) (*Config, error) {
	parser := hclparse.NewParser()
	file, diags := parser.ParseHCL(raw, filename)
	if diags.HasErrors() {
		return nil, fmt.Errorf("parse %s: %s", filename, diags.Error())
	}

	ctx := evalContext()

	var root hclRoot
	if diags := gohcl.DecodeBody(file.Body, ctx, &root); diags.HasErrors() {
		return nil, fmt.Errorf("decode %s: %s", filename, diags.Error())
	}

	cfg := &Config{
		Version: root.Version,
		Chain: Chain{
			Name:    root.Chain.Name,
			ChainID: root.Chain.ChainID,
			RPC:     root.Chain.RPC,
		},
	}
	for _, r := range root.Resources {
		address, spec, expect, err := decodeResourceBody(r.Body, ctx)
		if err != nil {
			return nil, fmt.Errorf("resource %q %q: %w", r.Type, r.Name, err)
		}
		cfg.Resources = append(cfg.Resources, ResourceConfig{
			Type:    r.Type,
			Name:    r.Name,
			Address: address,
			Spec:    spec,
			Expect:  expect,
		})
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// decodeResourceBody splits a resource block body into its address, its
// type-specific spec attributes, and any read-only `expect` block.
//
// It reads attributes directly from the native-HCL body rather than via
// Body.JustAttributes, because a resource body may also contain an `expect`
// block and JustAttributes rejects any body that contains a block — even one
// that has already been consumed.
func decodeResourceBody(body hcl.Body, ctx *hcl.EvalContext) (address string, spec, expect map[string]any, err error) {
	hb, ok := body.(*hclsyntax.Body)
	if !ok {
		return "", nil, nil, fmt.Errorf("unsupported configuration syntax (expected native HCL)")
	}

	spec = make(map[string]any, len(hb.Attributes))
	for name, attr := range hb.Attributes {
		gv, err := evalAttr(name, attr, ctx)
		if err != nil {
			return "", nil, nil, err
		}
		if name == "address" {
			s, ok := gv.(string)
			if !ok {
				return "", nil, nil, fmt.Errorf("address must be a string, got %T", gv)
			}
			address = s
			continue
		}
		spec[name] = gv
	}

	for _, blk := range hb.Blocks {
		if blk.Type != "expect" {
			return "", nil, nil, fmt.Errorf("unexpected %q block", blk.Type)
		}
		if expect != nil {
			return "", nil, nil, fmt.Errorf("at most one expect block is allowed")
		}
		expect, err = decodeAttrs(blk.Body, ctx)
		if err != nil {
			return "", nil, nil, fmt.Errorf("expect: %w", err)
		}
	}

	return address, spec, expect, nil
}

// decodeAttrs evaluates all attributes of a (sub-)block body into a generic
// attribute map. The body must contain attributes only, no nested blocks.
func decodeAttrs(body hcl.Body, ctx *hcl.EvalContext) (map[string]any, error) {
	hb, ok := body.(*hclsyntax.Body)
	if !ok {
		return nil, fmt.Errorf("unsupported configuration syntax (expected native HCL)")
	}
	if len(hb.Blocks) > 0 {
		return nil, fmt.Errorf("unexpected %q block", hb.Blocks[0].Type)
	}
	out := make(map[string]any, len(hb.Attributes))
	for name, attr := range hb.Attributes {
		gv, err := evalAttr(name, attr, ctx)
		if err != nil {
			return nil, err
		}
		out[name] = gv
	}
	return out, nil
}

// evalAttr evaluates a single attribute expression into a plain Go value.
func evalAttr(name string, attr *hclsyntax.Attribute, ctx *hcl.EvalContext) (any, error) {
	val, diags := attr.Expr.Value(ctx)
	if diags.HasErrors() {
		return nil, fmt.Errorf("%s: %s", name, diags.Error())
	}
	gv, err := ctyToGo(val)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	return gv, nil
}

// ctyToGo converts a cty scalar value into a plain Go value. Integers are
// returned as int so resource providers receive the same types they would
// from any other decoder.
func ctyToGo(v cty.Value) (any, error) {
	if v.IsNull() {
		return nil, nil
	}
	switch v.Type() {
	case cty.Bool:
		return v.True(), nil
	case cty.String:
		return v.AsString(), nil
	case cty.Number:
		bf := v.AsBigFloat()
		if bf.IsInt() {
			i, _ := bf.Int64()
			return int(i), nil
		}
		f, _ := bf.Float64()
		return f, nil
	default:
		return nil, fmt.Errorf("unsupported value type %s", v.Type().FriendlyName())
	}
}

// evalContext provides the functions available inside a configuration. env
// resolves an environment variable to a string (empty if unset).
func evalContext() *hcl.EvalContext {
	return &hcl.EvalContext{
		Functions: map[string]function.Function{
			"env": function.New(&function.Spec{
				Params: []function.Parameter{{Name: "name", Type: cty.String}},
				Type:   function.StaticReturnType(cty.String),
				Impl: func(args []cty.Value, _ cty.Type) (cty.Value, error) {
					return cty.StringVal(os.Getenv(args[0].AsString())), nil
				},
			}),
		},
	}
}
