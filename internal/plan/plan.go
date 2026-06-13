// Package plan implements the reconciliation step: for every configured
// resource it reads actual state, diffs it against desired state, and
// collects the resulting operations into a Plan that can be rendered for
// review or handed to an exporter.
package plan

import (
	"fmt"
	"io"

	"github.com/chainform/chainform/internal/config"
	"github.com/chainform/chainform/internal/resource"
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
