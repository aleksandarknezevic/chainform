package plan_test

import (
	"bytes"
	"encoding/json"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"

	"github.com/aleksandarknezevic/chainform/internal/config"
	"github.com/aleksandarknezevic/chainform/internal/plan"
	"github.com/aleksandarknezevic/chainform/internal/resource"
)

// JSON has no byte type, so byte values are emitted as 0x hex; lists are
// emitted element by element in both operation args and assertions.
func TestRenderJSONRichTypes(t *testing.T) {
	addr := common.HexToAddress("0x00000000000000000000000000000000000000AA")
	keeper := common.HexToAddress("0x1111111111111111111111111111111111111111")
	var root [32]byte
	root[0], root[31] = 0xa1, 0xff

	p := &plan.Plan{
		Chain: config.Chain{Name: "ethereum", ChainID: 1},
		Operations: []resource.Operation{
			{
				Resource: "vault", To: addr, Method: "setKeepers",
				Inputs: []string{"address[]"}, Args: []any{[]common.Address{keeper}},
				Value: big.NewInt(0), Calldata: []byte{0x01},
			},
			{
				Resource: "vault", To: addr, Method: "setMerkleRoot",
				Inputs: []string{"bytes32"}, Args: []any{root},
				Value: big.NewInt(0), Calldata: []byte{0x02},
			},
			{
				Resource: "vault", To: addr, Method: "setExtraData",
				Inputs: []string{"bytes"}, Args: []any{[]byte{0xde, 0xad}},
				Value: big.NewInt(0), Calldata: []byte{0x03},
			},
		},
		Assertions: []resource.Assertion{{
			Resource: "vault", Attr: "guardians", Type: "address[]",
			Expected: []any{keeper},
			Actual:   []any{keeper, addr},
		}},
	}

	var buf bytes.Buffer
	if err := p.RenderJSON(&buf); err != nil {
		t.Fatalf("RenderJSON: %v", err)
	}

	var doc struct {
		Operations []struct {
			Method string `json:"method"`
			Args   []any  `json:"args"`
		} `json:"operations"`
		Assertions []struct {
			Expected  []string `json:"expected"`
			Actual    []string `json:"actual"`
			Satisfied bool     `json:"satisfied"`
		} `json:"assertions"`
	}
	if err := json.Unmarshal(buf.Bytes(), &doc); err != nil {
		t.Fatalf("unmarshal: %v\n%s", err, buf.String())
	}

	list, ok := doc.Operations[0].Args[0].([]any)
	if !ok || len(list) != 1 || list[0] != keeper.Hex() {
		t.Errorf("setKeepers args = %#v, want [[%s]]", doc.Operations[0].Args, keeper.Hex())
	}
	wantRoot := "0xa1" + "00000000000000000000000000000000000000000000000000000000000000"[:60] + "ff"
	if got := doc.Operations[1].Args[0]; got != wantRoot {
		t.Errorf("setMerkleRoot arg = %v, want %s", got, wantRoot)
	}
	if got := doc.Operations[2].Args[0]; got != "0xdead" {
		t.Errorf("setExtraData arg = %v, want 0xdead", got)
	}

	a := doc.Assertions[0]
	if len(a.Expected) != 1 || a.Expected[0] != keeper.Hex() {
		t.Errorf("assertion expected = %v", a.Expected)
	}
	if len(a.Actual) != 2 || a.Satisfied {
		t.Errorf("assertion actual = %v, satisfied = %v; want 2 elements and unsatisfied", a.Actual, a.Satisfied)
	}
}

// The human renderer must print bytes and lists readably, not as Go values.
func TestRenderRichTypeArgs(t *testing.T) {
	var root [32]byte
	root[0] = 0xa1
	p := &plan.Plan{
		Chain: config.Chain{Name: "ethereum", ChainID: 1},
		Operations: []resource.Operation{{
			Resource: "vault",
			To:       common.HexToAddress("0x00000000000000000000000000000000000000AA"),
			Method:   "setMerkleRoot",
			Inputs:   []string{"bytes32"},
			Args:     []any{root},
			Value:    big.NewInt(0),
		}},
	}

	var buf bytes.Buffer
	p.Render(&buf)
	if !bytes.Contains(buf.Bytes(), []byte("setMerkleRoot(0xa1")) {
		t.Errorf("rendered plan does not show hex bytes:\n%s", buf.String())
	}
}
