package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

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

// jsonRoot mirrors the JSON configuration document structure.
type jsonRoot struct {
	Version   string         `json:"version"`
	Chain     jsonChain      `json:"chain"`
	Resources []jsonResource `json:"resources"`
}

type jsonChain struct {
	Name    string `json:"name"`
	ChainID uint64 `json:"chain_id"`
	RPC     string `json:"rpc"`
}

type jsonResource struct {
	Type    string         `json:"type"`
	Name    string         `json:"name"`
	Address string         `json:"address"`
	Spec    map[string]any `json:"spec"`
	Expect  map[string]any `json:"expect"`
}

// Load reads, parses, and validates a configuration file.
func Load(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}
	return Parse(raw, path)
}

// Parse decodes and validates a ChainForm configuration document as either
// HCL or JSON. filename is used only for diagnostics. For HCL, the env("VAR")
// function is available to resolve environment variables (e.g. for RPC
// endpoints) at load time.
func Parse(raw []byte, filename string) (*Config, error) {
	if looksLikeJSON(raw) {
		return parseJSON(raw, filename)
	}
	return parseHCL(raw, filename)
}

func parseHCL(raw []byte, filename string) (*Config, error) {
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

func parseJSON(raw []byte, filename string) (*Config, error) {
	var top map[string]any
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(&top); err != nil {
		return nil, fmt.Errorf("parse %s: %w", filename, err)
	}
	if err := dec.Decode(&struct{}{}); err == nil {
		return nil, fmt.Errorf("parse %s: unexpected trailing JSON content", filename)
	} else if err != io.EOF {
		return nil, fmt.Errorf("parse %s: %w", filename, err)
	}

	root, err := decodeJSONRoot(top)
	if err != nil {
		return nil, fmt.Errorf("decode %s: %w", filename, err)
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
		cfg.Resources = append(cfg.Resources, ResourceConfig{
			Type:    r.Type,
			Name:    r.Name,
			Address: r.Address,
			Spec:    r.Spec,
			Expect:  r.Expect,
		})
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func looksLikeJSON(raw []byte) bool {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" {
		return false
	}
	return strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[")
}

func decodeJSONRoot(top map[string]any) (jsonRoot, error) {
	var out jsonRoot
	if top == nil {
		return out, fmt.Errorf("expected JSON object at top level")
	}

	if v, ok := top["version"]; ok {
		s, ok := v.(string)
		if !ok {
			return out, fmt.Errorf("version: expected string, got %T", v)
		}
		out.Version = s
	}

	chainVal, ok := top["chain"]
	if !ok {
		return out, fmt.Errorf("chain: required")
	}
	chainMap, ok := chainVal.(map[string]any)
	if !ok {
		return out, fmt.Errorf("chain: expected object, got %T", chainVal)
	}

	if v, ok := chainMap["name"]; ok {
		s, ok := v.(string)
		if !ok {
			return out, fmt.Errorf("chain.name: expected string, got %T", v)
		}
		out.Chain.Name = s
	}
	if v, ok := chainMap["rpc"]; ok {
		s, ok := v.(string)
		if !ok {
			return out, fmt.Errorf("chain.rpc: expected string, got %T", v)
		}
		out.Chain.RPC = s
	}
	if v, ok := chainMap["chain_id"]; ok {
		n, err := toJSONUint64(v)
		if err != nil {
			return out, fmt.Errorf("chain.chain_id: %w", err)
		}
		out.Chain.ChainID = n
	}

	resourcesVal, ok := top["resources"]
	if !ok {
		return out, fmt.Errorf("resources: required")
	}
	resourcesList, ok := resourcesVal.([]any)
	if !ok {
		return out, fmt.Errorf("resources: expected array, got %T", resourcesVal)
	}

	out.Resources = make([]jsonResource, 0, len(resourcesList))
	for i, item := range resourcesList {
		obj, ok := item.(map[string]any)
		if !ok {
			return out, fmt.Errorf("resources[%d]: expected object, got %T", i, item)
		}
		res, err := decodeJSONResource(obj)
		if err != nil {
			return out, fmt.Errorf("resources[%d]: %w", i, err)
		}
		out.Resources = append(out.Resources, res)
	}

	return out, nil
}

func decodeJSONResource(obj map[string]any) (jsonResource, error) {
	var out jsonResource
	out.Spec = map[string]any{}

	for k, v := range obj {
		switch k {
		case "type":
			s, ok := v.(string)
			if !ok {
				return out, fmt.Errorf("type: expected string, got %T", v)
			}
			out.Type = s
		case "name":
			s, ok := v.(string)
			if !ok {
				return out, fmt.Errorf("name: expected string, got %T", v)
			}
			out.Name = s
		case "address":
			s, ok := v.(string)
			if !ok {
				return out, fmt.Errorf("address: expected string, got %T", v)
			}
			out.Address = s
		case "spec":
			specMap, ok := v.(map[string]any)
			if !ok {
				return out, fmt.Errorf("spec: expected object, got %T", v)
			}
			spec, err := normalizeJSONMap(specMap)
			if err != nil {
				return out, fmt.Errorf("spec: %w", err)
			}
			for sk, sv := range spec {
				out.Spec[sk] = sv
			}
		case "expect":
			expectMap, ok := v.(map[string]any)
			if !ok {
				return out, fmt.Errorf("expect: expected object, got %T", v)
			}
			expect, err := normalizeJSONMap(expectMap)
			if err != nil {
				return out, fmt.Errorf("expect: %w", err)
			}
			out.Expect = expect
		default:
			cv, err := normalizeJSONValue(v)
			if err != nil {
				return out, fmt.Errorf("%s: %w", k, err)
			}
			// JSON also accepts flat resource attributes, same as HCL top-level attrs.
			out.Spec[k] = cv
		}
	}

	if len(out.Spec) == 0 {
		out.Spec = nil
	}
	return out, nil
}

func normalizeJSONMap(m map[string]any) (map[string]any, error) {
	out := make(map[string]any, len(m))
	for k, v := range m {
		cv, err := normalizeJSONValue(v)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", k, err)
		}
		out[k] = cv
	}
	return out, nil
}

func normalizeJSONValue(v any) (any, error) {
	switch x := v.(type) {
	case nil, bool, string:
		return x, nil
	case json.Number:
		if i, err := x.Int64(); err == nil {
			return int(i), nil
		}
		f, err := x.Float64()
		if err != nil {
			return nil, fmt.Errorf("invalid number %q", x.String())
		}
		return f, nil
	case float64:
		if x == float64(int64(x)) {
			return int(x), nil
		}
		return x, nil
	default:
		return nil, fmt.Errorf("unsupported value type %T", v)
	}
}

func toJSONUint64(v any) (uint64, error) {
	switch x := v.(type) {
	case json.Number:
		i, err := x.Int64()
		if err != nil {
			return 0, fmt.Errorf("expected integer, got %q", x.String())
		}
		if i < 0 {
			return 0, fmt.Errorf("must be non-negative, got %d", i)
		}
		return uint64(i), nil
	case float64:
		if x < 0 || x != float64(uint64(x)) {
			return 0, fmt.Errorf("expected non-negative integer, got %v", x)
		}
		return uint64(x), nil
	default:
		return 0, fmt.Errorf("expected number, got %T", v)
	}
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
