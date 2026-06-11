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
}

// Empty reports whether there is no drift (no operations required).
func (p *Plan) Empty() bool { return len(p.Operations) == 0 }

// Render writes a human-readable, Terraform-style summary of the plan.
func (p *Plan) Render(w io.Writer) {
	if p.Empty() {
		fmt.Fprintln(w, "No drift. Actual on-chain state matches desired state.")
		return
	}

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
