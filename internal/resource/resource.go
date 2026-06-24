// Package resource defines the core reconciliation contract: a Resource knows
// how to read its own actual state from the chain and how to compute the
// minimal set of operations required to converge that state toward the
// desired state declared in configuration.
//
// New resource types (accessControl, proxy, ...) are added by implementing
// Resource and registering a Factory in an init() function. Nothing else in
// the codebase needs to change; the planner discovers resources through the
// registry.
package resource

import (
	"context"
	"fmt"
	"math/big"
	"sort"

	"github.com/ethereum/go-ethereum/common"

	"github.com/aleksandarknezevic/chainform/internal/chain"
	"github.com/aleksandarknezevic/chainform/internal/config"
)

// State is a bag of observed or desired attributes for a single resource.
type State map[string]any

// Operation is a single contract call required to move actual state toward
// desired state. The planner fills Calldata after a resource returns its
// operations, so resources only need to describe the call intent.
type Operation struct {
	Resource string         // local resource name (from config)
	To       common.Address // contract to call
	Method   string         // function name, e.g. "setFeeBps"
	Inputs   []string       // ABI input types, e.g. ["uint256"]
	Args     []any          // argument values matching Inputs
	Value    *big.Int       // wei to send (typically 0)
	Reason   string         // human-readable drift description
	Calldata []byte         // ABI-encoded calldata; populated by the planner
}

// Resource is a managed on-chain entity.
type Resource interface {
	// Type returns the resource type identifier (matches config "type").
	Type() string
	// Name returns the unique local name (matches config "name").
	Name() string
	// Address returns the managed contract address.
	Address() common.Address
	// Refresh reads the current on-chain state via the given Reader.
	Refresh(ctx context.Context, r chain.Reader) (State, error)
	// Plan compares the desired state (held by the resource) against the
	// supplied current state and returns the operations needed to converge.
	// It must return no operations when there is no drift.
	Plan(current State) ([]Operation, error)
}

// Factory builds a Resource from its configuration block.
type Factory func(cfg config.ResourceConfig) (Resource, error)

var registry = map[string]Factory{}

// Register makes a resource type available to the planner. Call it from an
// init() function in the file that implements the type.
func Register(typ string, f Factory) {
	if _, dup := registry[typ]; dup {
		panic(fmt.Sprintf("resource type %q registered twice", typ))
	}
	registry[typ] = f
}

// Build instantiates a Resource from config using its registered factory.
func Build(cfg config.ResourceConfig) (Resource, error) {
	f, ok := registry[cfg.Type]
	if !ok {
		return nil, fmt.Errorf("unknown resource type %q (known: %v)", cfg.Type, Types())
	}
	return f(cfg)
}

// Types returns the registered resource types, sorted.
func Types() []string {
	out := make([]string, 0, len(registry))
	for t := range registry {
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}
