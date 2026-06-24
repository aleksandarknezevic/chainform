// Package plan implements the reconciliation step: for every configured
// resource it reads actual state, diffs it against desired state, and
// collects the resulting operations into a Plan that can be rendered for
// review or handed to an exporter.
package plan

import (
	"encoding/json"
	"fmt"
	"io"
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"github.com/aleksandarknezevic/chainform/internal/config"
	"github.com/aleksandarknezevic/chainform/internal/resource"
)

// Plan is the ordered set of operations required to converge actual on-chain
// state to the desired state, together with the chain they target.
type Plan struct {
	Chain      config.Chain
	Operations []resource.Operation
	// Assertions are read-only invariant checks (from `expect` blocks). They
	// never become operations; failing ones are reported as warnings.
	Assertions []resource.Assertion
}

// Empty reports whether there is no drift (no operations required).
func (p *Plan) Empty() bool { return len(p.Operations) == 0 }

// HasDrift reports whether any managed attribute drifted or any read-only
// expectation failed.
func (p *Plan) HasDrift() bool {
	if !p.Empty() {
		return true
	}
	for _, a := range p.Assertions {
		if !a.Satisfied() {
			return true
		}
	}
	return false
}

// failedAssertions returns the assertions whose on-chain value does not match
// the expected value.
func (p *Plan) failedAssertions() []resource.Assertion {
	var out []resource.Assertion
	for _, a := range p.Assertions {
		if !a.Satisfied() {
			out = append(out, a)
		}
	}
	return out
}

// Render writes a human-readable, Terraform-style summary of the plan.
func (p *Plan) Render(w io.Writer) {
	failed := p.failedAssertions()

	if p.Empty() && len(failed) == 0 {
		fmt.Fprintln(w, "No drift. Actual on-chain state matches desired state.")
		return
	}

	if !p.Empty() {
		fmt.Fprintf(w, "Plan: %d operation(s) on %s (chainId %d)\n\n",
			len(p.Operations), nameOr(p.Chain.Name, "evm"), p.Chain.ChainID)

		for i, op := range p.Operations {
			fmt.Fprintf(w, "  %d. %s.%s(%s)\n", i+1, op.Resource, op.Method, formatArgs(op.Args))
			fmt.Fprintf(w, "       to:       %s\n", op.To.Hex())
			if op.Reason != "" {
				fmt.Fprintf(w, "       drift:    %s\n", op.Reason)
			}
			fmt.Fprintf(w, "       calldata: 0x%x\n", op.Calldata)
			if i < len(p.Operations)-1 {
				fmt.Fprintln(w)
			}
		}
	}

	if len(failed) > 0 {
		if !p.Empty() {
			fmt.Fprintln(w)
		}
		fmt.Fprintf(w, "Read-only drift: %d expectation(s) not met — no setter, cannot be changed:\n\n", len(failed))
		for _, a := range failed {
			fmt.Fprintf(w, "  ! %s.%s (%s): on-chain %s, expected %s\n",
				a.Resource, a.Attr, a.Type,
				resource.FormatValue(a.Actual), resource.FormatValue(a.Expected))
		}
	}
}

// RenderJSON writes a machine-readable JSON representation of the plan.
func (p *Plan) RenderJSON(w io.Writer) error {
	failed := p.failedAssertions()
	out := jsonPlan{
		Chain: jsonChain{
			Name:    p.Chain.Name,
			ChainID: p.Chain.ChainID,
			RPC:     p.Chain.RPC,
		},
		Operations: make([]jsonOperation, len(p.Operations)),
		Assertions: make([]jsonAssertion, len(p.Assertions)),
		Summary: jsonSummary{
			OperationCount:       len(p.Operations),
			AssertionCount:       len(p.Assertions),
			FailedAssertionCount: len(failed),
			Empty:                p.Empty(),
		},
	}

	for i, op := range p.Operations {
		out.Operations[i] = jsonOperation{
			Resource: op.Resource,
			To:       op.To.Hex(),
			Method:   op.Method,
			Inputs:   op.Inputs,
			Args:     op.Args,
			ValueWei: bigIntStringOrZero(op.Value),
			Reason:   op.Reason,
			Calldata: fmt.Sprintf("0x%x", op.Calldata),
		}
	}

	for i, a := range p.Assertions {
		out.Assertions[i] = jsonAssertion{
			Resource:  a.Resource,
			Attr:      a.Attr,
			Type:      a.Type,
			Expected:  jsonValue(a.Expected),
			Actual:    jsonValue(a.Actual),
			Satisfied: a.Satisfied(),
		}
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

type jsonPlan struct {
	Chain      jsonChain       `json:"chain"`
	Operations []jsonOperation `json:"operations"`
	Assertions []jsonAssertion `json:"assertions"`
	Summary    jsonSummary     `json:"summary"`
}

type jsonChain struct {
	Name    string `json:"name"`
	ChainID uint64 `json:"chainId"`
	RPC     string `json:"rpc"`
}

type jsonOperation struct {
	Resource string   `json:"resource"`
	To       string   `json:"to"`
	Method   string   `json:"method"`
	Inputs   []string `json:"inputs"`
	Args     []any    `json:"args"`
	ValueWei string   `json:"valueWei"`
	Reason   string   `json:"reason,omitempty"`
	Calldata string   `json:"calldata"`
}

type jsonAssertion struct {
	Resource  string `json:"resource"`
	Attr      string `json:"attr"`
	Type      string `json:"type"`
	Expected  any    `json:"expected"`
	Actual    any    `json:"actual"`
	Satisfied bool   `json:"satisfied"`
}

type jsonSummary struct {
	OperationCount       int  `json:"operationCount"`
	AssertionCount       int  `json:"assertionCount"`
	FailedAssertionCount int  `json:"failedAssertionCount"`
	Empty                bool `json:"empty"`
}

func jsonValue(v any) any {
	switch x := v.(type) {
	case *big.Int:
		return x.String()
	case common.Address:
		return x.Hex()
	default:
		return x
	}
}

func bigIntStringOrZero(v *big.Int) string {
	if v == nil {
		return "0"
	}
	return v.String()
}

func formatArgs(args []any) string {
	parts := make([]string, len(args))
	for i, a := range args {
		parts[i] = fmt.Sprintf("%v", a)
	}
	return join(parts, ", ")
}

func join(parts []string, sep string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += sep
		}
		out += p
	}
	return out
}

func nameOr(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}
