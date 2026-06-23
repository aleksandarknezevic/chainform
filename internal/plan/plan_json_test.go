package plan_test

import (
	"bytes"
	"encoding/json"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"

	"github.com/chainform/chainform/internal/config"
	"github.com/chainform/chainform/internal/plan"
	"github.com/chainform/chainform/internal/resource"
)

func TestRenderJSONIncludesSummaryAndHexFields(t *testing.T) {
	addr := common.HexToAddress("0x00000000000000000000000000000000000000AA")
	p := &plan.Plan{
		Chain: config.Chain{Name: "ethereum", ChainID: 1},
		Operations: []resource.Operation{{
			Resource: "main",
			To:       addr,
			Method:   "setFeeBps",
			Inputs:   []string{"uint256"},
			Args:     []any{uint64(30)},
			Reason:   "feeBps: 50 -> 30",
			Calldata: []byte{0x72, 0xc2, 0x7b, 0x62},
		}},
		Assertions: []resource.Assertion{
			{Resource: "main", Attr: "name", Type: "string", Expected: "Demo", Actual: "Demo"},
			{Resource: "main", Attr: "owner", Type: "address", Expected: addr, Actual: common.HexToAddress("0x00000000000000000000000000000000000000BB")},
		},
	}

	var buf bytes.Buffer
	if err := p.RenderJSON(&buf); err != nil {
		t.Fatalf("RenderJSON: %v", err)
	}

	var got struct {
		Operations []struct {
			To       string `json:"to"`
			ValueWei string `json:"valueWei"`
			Calldata string `json:"calldata"`
		} `json:"operations"`
		Assertions []struct {
			Expected  any  `json:"expected"`
			Actual    any  `json:"actual"`
			Satisfied bool `json:"satisfied"`
		} `json:"assertions"`
		Summary struct {
			OperationCount       int  `json:"operationCount"`
			AssertionCount       int  `json:"assertionCount"`
			FailedAssertionCount int  `json:"failedAssertionCount"`
			Empty                bool `json:"empty"`
		} `json:"summary"`
	}
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal json output: %v\n%s", err, buf.String())
	}

	if len(got.Operations) != 1 {
		t.Fatalf("operations len = %d, want 1", len(got.Operations))
	}
	if got.Operations[0].To != addr.Hex() {
		t.Errorf("operations[0].to = %q, want %q", got.Operations[0].To, addr.Hex())
	}
	if got.Operations[0].ValueWei != "0" {
		t.Errorf("operations[0].valueWei = %q, want 0", got.Operations[0].ValueWei)
	}
	if got.Operations[0].Calldata != "0x72c27b62" {
		t.Errorf("operations[0].calldata = %q, want 0x72c27b62", got.Operations[0].Calldata)
	}

	if len(got.Assertions) != 2 {
		t.Fatalf("assertions len = %d, want 2", len(got.Assertions))
	}
	if expected, ok := got.Assertions[1].Expected.(string); !ok || expected != addr.Hex() {
		t.Errorf("assertions[1].expected = %#v, want %q", got.Assertions[1].Expected, addr.Hex())
	}
	if got.Assertions[0].Satisfied != true || got.Assertions[1].Satisfied != false {
		t.Errorf("unexpected assertion satisfaction values: %+v", got.Assertions)
	}

	if got.Summary.OperationCount != 1 {
		t.Errorf("summary.operationCount = %d, want 1", got.Summary.OperationCount)
	}
	if got.Summary.AssertionCount != 2 {
		t.Errorf("summary.assertionCount = %d, want 2", got.Summary.AssertionCount)
	}
	if got.Summary.FailedAssertionCount != 1 {
		t.Errorf("summary.failedAssertionCount = %d, want 1", got.Summary.FailedAssertionCount)
	}
	if got.Summary.Empty {
		t.Error("summary.empty = true, want false")
	}
}

func TestRenderJSONStringifiesBigIntsInAssertions(t *testing.T) {
	p := &plan.Plan{
		Chain: config.Chain{Name: "ethereum", ChainID: 1},
		Assertions: []resource.Assertion{{
			Resource: "main",
			Attr:     "feeBps",
			Type:     "uint256",
			Expected: big.NewInt(30),
			Actual:   big.NewInt(50),
		}},
	}

	var buf bytes.Buffer
	if err := p.RenderJSON(&buf); err != nil {
		t.Fatalf("RenderJSON: %v", err)
	}

	var got struct {
		Assertions []struct {
			Expected string `json:"expected"`
			Actual   string `json:"actual"`
		} `json:"assertions"`
	}
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal json output: %v\n%s", err, buf.String())
	}

	if got.Assertions[0].Expected != "30" {
		t.Errorf("expected = %q, want 30", got.Assertions[0].Expected)
	}
	if got.Assertions[0].Actual != "50" {
		t.Errorf("actual = %q, want 50", got.Assertions[0].Actual)
	}
}
