package plan_test

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"

	"github.com/chainform/chainform/internal/chain"
	"github.com/chainform/chainform/internal/config"
	"github.com/chainform/chainform/internal/plan"

	_ "github.com/chainform/chainform/internal/resource" // register built-in resources
)

const testAddr = "0x0000000000000000000000000000000000000001"

func newConfig() *config.Config {
	return &config.Config{
		Version: "1",
		Chain:   config.Chain{Name: "ethereum", ChainID: 1},
		Resources: []config.ResourceConfig{{
			Type:    "protocol",
			Name:    "main",
			Address: testAddr,
			Spec:    map[string]any{"feeBps": 30, "paused": false},
		}},
	}
}

// The DemoReader reports feeBps=50, paused=true. With desired feeBps=30,
// paused=false the planner should produce setFeeBps(30) and unpause().
func TestRunDetectsDrift(t *testing.T) {
	p, err := plan.NewPlanner(newConfig(), chain.DemoReader{}).Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got := len(p.Operations); got != 2 {
		t.Fatalf("expected 2 operations, got %d", got)
	}
	if m := p.Operations[0].Method; m != "setFeeBps" {
		t.Errorf("op[0] method = %q, want setFeeBps", m)
	}
	if m := p.Operations[1].Method; m != "unpause" {
		t.Errorf("op[1] method = %q, want unpause", m)
	}
	if len(p.Operations[0].Calldata) == 0 {
		t.Error("op[0] calldata not encoded")
	}
}

// When desired state matches actual state, the plan must be empty.
func TestRunNoDrift(t *testing.T) {
	mock := chain.NewMockReader()
	addr := common.HexToAddress(testAddr)
	mock.Set(addr, "feeBps", big.NewInt(30))
	mock.Set(addr, "paused", false)

	p, err := plan.NewPlanner(newConfig(), mock).Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !p.Empty() {
		t.Fatalf("expected no drift, got %d operations", len(p.Operations))
	}
}
