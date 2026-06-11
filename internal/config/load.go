package config

import (
	"fmt"
	"os"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclparse"
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
	Type    string   `hcl:"type,label"`
	Name    string   `hcl:"name,label"`
	Address string   `hcl:"address"`
	Spec    hcl.Body `hcl:",remain"` // type-specific attributes
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
		spec, err := decodeSpec(r.Spec, ctx)
		if err != nil {
			return nil, fmt.Errorf("resource %q %q: %w", r.Type, r.Name, err)
		}
		cfg.Resources = append(cfg.Resources, ResourceConfig{
			Type:    r.Type,
			Name:    r.Name,
			Address: r.Address,
			Spec:    spec,
		})
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// decodeSpec evaluates the remaining attributes of a resource block into a
// generic attribute map for the resource provider to interpret.
func decodeSpec(body hcl.Body, ctx *hcl.EvalContext) (map[string]any, error) {
	attrs, diags := body.JustAttributes()
	if diags.HasErrors() {
		return nil, fmt.Errorf("%s", diags.Error())
	}
	spec := make(map[string]any, len(attrs))
	for name, attr := range attrs {
		val, diags := attr.Expr.Value(ctx)
		if diags.HasErrors() {
			return nil, fmt.Errorf("%s: %s", name, diags.Error())
		}
		gv, err := ctyToGo(val)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		spec[name] = gv
	}
	return spec, nil
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
