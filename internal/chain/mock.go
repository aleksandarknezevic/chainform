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

// DemoReader returns fixed, intentionally-drifted values so that `chainform
// plan` produces meaningful output without a live RPC endpoint. The values
// mirror the "actual state" in the project README (feeBps = 50, paused =
// true). For demos and documentation only.
type DemoReader struct{}

// Read implements Reader.
func (DemoReader) Read(_ context.Context, call ViewCall) ([]any, error) {
	switch call.Method {
	case "feeBps":
		return []any{big.NewInt(50)}, nil
	case "paused":
		return []any{true}, nil
	default:
		return nil, fmt.Errorf("demo reader has no value for method %q", call.Method)
	}
}
