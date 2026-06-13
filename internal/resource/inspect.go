package resource

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"github.com/chainform/chainform/internal/chain"
)

// Observation is a single readable on-chain value reported by `chainform show`.
type Observation struct {
	Name  string // getter / attribute name, e.g. "decimals"
	Type  string // ABI type of the value, e.g. "uint8"
	Value any    // decoded value, as returned by the chain.Reader
}

// Inspector is an optional Resource capability: report the full observable
// state of a contract — every readable getter — independent of which
// attributes are managed. Read-only contracts (e.g. a price feed) implement it
// so `chainform show` can print on-chain state without computing a diff.
//
// Resources that do not implement Inspector are still inspectable through their
// managed state via Refresh; the show command falls back to that.
type Inspector interface {
	Inspect(ctx context.Context, r chain.Reader) ([]Observation, error)
}

// FormatValue renders a decoded on-chain value for human-readable display:
// integers in base 10, addresses as checksummed hex, strings quoted.
func FormatValue(v any) string {
	switch x := v.(type) {
	case *big.Int:
		return x.String()
	case common.Address:
		return x.Hex()
	case string:
		return fmt.Sprintf("%q", x)
	default:
		return fmt.Sprintf("%v", x)
	}
}
