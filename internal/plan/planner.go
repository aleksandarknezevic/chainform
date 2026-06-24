package plan

import (
	"context"
	"fmt"

	"github.com/aleksandarknezevic/chainform/internal/chain"
	"github.com/aleksandarknezevic/chainform/internal/config"
	"github.com/aleksandarknezevic/chainform/internal/resource"
)

// Planner reconciles a configuration against actual on-chain state read
// through a chain.Reader. It is the analogue of a controller's reconcile loop:
// build resource -> refresh actual state -> diff -> collect operations.
type Planner struct {
	cfg    *config.Config
	reader chain.Reader
}

// NewPlanner returns a Planner for the given configuration and reader.
func NewPlanner(cfg *config.Config, reader chain.Reader) *Planner {
	return &Planner{cfg: cfg, reader: reader}
}

// Run executes one reconciliation pass and returns the resulting Plan.
func (p *Planner) Run(ctx context.Context) (*Plan, error) {
	out := &Plan{Chain: p.cfg.Chain}

	for _, rc := range p.cfg.Resources {
		res, err := resource.Build(rc)
		if err != nil {
			return nil, err
		}

		current, err := res.Refresh(ctx, p.reader)
		if err != nil {
			return nil, fmt.Errorf("refresh %s/%s: %w", res.Type(), res.Name(), err)
		}

		ops, err := res.Plan(current)
		if err != nil {
			return nil, fmt.Errorf("plan %s/%s: %w", res.Type(), res.Name(), err)
		}

		for i := range ops {
			data, err := chain.Pack(ops[i].Method, ops[i].Inputs, ops[i].Args...)
			if err != nil {
				return nil, fmt.Errorf("encode %s.%s: %w", ops[i].Resource, ops[i].Method, err)
			}
			ops[i].Calldata = data
			out.Operations = append(out.Operations, ops[i])
		}

		// Read-only assertions (expect blocks) are reported, never executed.
		if a, ok := res.(resource.Asserter); ok {
			assertions, err := a.Assert(current)
			if err != nil {
				return nil, fmt.Errorf("assert %s/%s: %w", res.Type(), res.Name(), err)
			}
			out.Assertions = append(out.Assertions, assertions...)
		}
	}

	return out, nil
}
