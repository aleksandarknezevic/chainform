package chain

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

// MockReader returns canned values keyed by "address.method". It is intended
// for unit tests that need deterministic, fully-controlled on-chain state.
type MockReader struct {
	values map[string][]any
}

// NewMockReader returns an empty MockReader.
func NewMockReader() *MockReader {
	return &MockReader{values: map[string][]any{}}
}

// Set registers the values a given (address, method) view call should return.
func (m *MockReader) Set(addr common.Address, method string, values ...any) *MockReader {
	m.values[mockKey(addr, method)] = values
	return m
}

// Read implements Reader.
func (m *MockReader) Read(_ context.Context, call ViewCall) ([]any, error) {
	v, ok := m.values[mockKey(call.To, call.Method)]
	if !ok {
		return nil, fmt.Errorf("mock: no value configured for %s.%s", call.To.Hex(), call.Method)
	}
	return v, nil
}

func mockKey(addr common.Address, method string) string {
	return strings.ToLower(addr.Hex()) + "." + method
}

// DemoReader returns fixed values so that ChainForm commands produce meaningful
// output without a live RPC endpoint. It serves the two shipped examples: a
// mutable "protocol" contract whose state is intentionally drifted (feeBps =
// 50, paused = true) so `plan` has work to do, and a read-only Chainlink-style
// price feed so `show` has state to print. For demos and documentation only.
type DemoReader struct{}

// Read implements Reader.
func (DemoReader) Read(_ context.Context, call ViewCall) ([]any, error) {
	switch call.Method {
	// protocol contract (mutable; drifted from desired state)
	case "feeBps":
		return []any{big.NewInt(50)}, nil
	case "paused":
		return []any{true}, nil
	case "name":
		return []any{"Demo Protocol"}, nil

	// price feed (read-only; inspected via `show`)
	case "decimals":
		return []any{uint8(8)}, nil
	case "description":
		return []any{"ETH / USD"}, nil
	case "version":
		return []any{big.NewInt(4)}, nil
	case "latestAnswer", "getAnswer":
		return []any{big.NewInt(372912340000)}, nil // 3729.1234 at 8 decimals
	case "latestRound":
		return []any{big.NewInt(299)}, nil
	case "latestTimestamp":
		return []any{big.NewInt(1749816000)}, nil
	case "aggregator":
		return []any{common.HexToAddress("0x719E22E3D4b690E5d96cCb40619180B5427F14AE")}, nil
	case "owner":
		return []any{common.HexToAddress("0x21f73D42Eb58Ba49dDB685dc29D3bF5c0f0373CA")}, nil

	default:
		return nil, fmt.Errorf("demo reader has no value for method %q", call.Method)
	}
}
