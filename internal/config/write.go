package config

import (
	"io"
	"sort"

	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
)

// ResourceDoc is the data needed to render a single-resource configuration
// document, e.g. the output of `chainform import`. Managed and Expect hold the
// attribute values already converted to cty values; Managed becomes top-level
// (getter+setter) attributes and Expect becomes a read-only `expect` block.
type ResourceDoc struct {
	ChainName string
	ChainID   uint64
	RPCEnvVar string // environment variable name for the rpc endpoint, e.g. "RPC_URL"

	Type    string
	Name    string
	Address string
	ABIPath string

	Managed map[string]cty.Value
	Expect  map[string]cty.Value
}

// WriteResource renders doc as a ChainForm HCL configuration document. The
// output is valid input to Parse: a managed attribute carries its current
// on-chain value, so an immediate `plan` against the same state reports no
// drift.
func WriteResource(w io.Writer, doc ResourceDoc) error {
	f := hclwrite.NewEmptyFile()
	body := f.Body()

	body.SetAttributeValue("version", cty.StringVal("1"))
	body.AppendNewline()

	chainBody := body.AppendNewBlock("chain", nil).Body()
	if doc.ChainName != "" {
		chainBody.SetAttributeValue("name", cty.StringVal(doc.ChainName))
	}
	chainBody.SetAttributeValue("chain_id", cty.NumberUIntVal(doc.ChainID))
	envVar := doc.RPCEnvVar
	if envVar == "" {
		envVar = "RPC_URL"
	}
	chainBody.SetAttributeRaw("rpc", hclwrite.TokensForFunctionCall(
		"env", hclwrite.TokensForValue(cty.StringVal(envVar)),
	))
	body.AppendNewline()

	resBody := body.AppendNewBlock("resource", []string{doc.Type, doc.Name}).Body()
	resBody.SetAttributeValue("address", cty.StringVal(doc.Address))
	resBody.SetAttributeValue("abi", cty.StringVal(doc.ABIPath))
	if len(doc.Managed) > 0 {
		resBody.AppendNewline()
		setSorted(resBody, doc.Managed)
	}
	if len(doc.Expect) > 0 {
		resBody.AppendNewline()
		expectBody := resBody.AppendNewBlock("expect", nil).Body()
		setSorted(expectBody, doc.Expect)
	}

	_, err := f.WriteTo(w)
	return err
}

// setSorted writes attributes in a stable (sorted) order for deterministic
// output.
func setSorted(body *hclwrite.Body, attrs map[string]cty.Value) {
	names := make([]string, 0, len(attrs))
	for k := range attrs {
		names = append(names, k)
	}
	sort.Strings(names)
	for _, k := range names {
		body.SetAttributeValue(k, attrs[k])
	}
}
