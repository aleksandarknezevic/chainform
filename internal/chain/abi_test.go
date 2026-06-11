package chain

import (
	"math/big"
	"testing"
)

func TestSelector(t *testing.T) {
	got := Selector("setFeeBps", []string{"uint256"})
	if len(got) != 4 {
		t.Fatalf("selector length = %d, want 4", len(got))
	}
	// Selectors must be deterministic and distinguish different signatures.
	if string(got) != string(Selector("setFeeBps", []string{"uint256"})) {
		t.Error("selector is not deterministic")
	}
	if string(got) == string(Selector("setFeeBps", []string{"uint8"})) {
		t.Error("selector should depend on argument types")
	}
}

func TestPackUnpackRoundTrip(t *testing.T) {
	data, err := Pack("setFeeBps", []string{"uint256"}, big.NewInt(30))
	if err != nil {
		t.Fatalf("Pack: %v", err)
	}
	if len(data) != 4+32 {
		t.Fatalf("calldata length = %d, want 36", len(data))
	}
	out, err := Unpack([]string{"uint256"}, data[4:])
	if err != nil {
		t.Fatalf("Unpack: %v", err)
	}
	if got := out[0].(*big.Int); got.Int64() != 30 {
		t.Fatalf("round-trip value = %v, want 30", got)
	}
}

func TestPackNoArgs(t *testing.T) {
	data, err := Pack("unpause", nil)
	if err != nil {
		t.Fatalf("Pack: %v", err)
	}
	if len(data) != 4 {
		t.Fatalf("calldata length = %d, want 4 (selector only)", len(data))
	}
}
