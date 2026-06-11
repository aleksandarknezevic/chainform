// Package chain provides read access to EVM contract state and the ABI
// helpers needed to encode operations. It deliberately knows nothing about
// configuration or resources: it exposes a small Reader interface that the
// rest of ChainForm depends on, plus concrete implementations (a live
// JSON-RPC client, a programmable mock, and a fixed demo reader).
package chain

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
)

// ViewCall describes a read-only (eth_call) contract invocation. Inputs and
// Outputs are Solidity ABI type lists, e.g. ["uint256"] or ["address","bool"].
type ViewCall struct {
	To      common.Address
	Method  string
	Inputs  []string
	Args    []any
	Outputs []string
}

// Reader reads on-chain state via read-only calls. Resources depend on this
// interface, never on a concrete client, which keeps them testable offline.
type Reader interface {
	Read(ctx context.Context, call ViewCall) ([]any, error)
}
