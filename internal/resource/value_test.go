package resource

import (
	"math/big"
	"strings"
	"testing"

	ethabi "github.com/ethereum/go-ethereum/accounts/abi"
)

func mustType(t *testing.T, typ string) ethabi.Type {
	t.Helper()
	at, err := ethabi.NewType(typ, "", nil)
	if err != nil {
		t.Fatalf("NewType(%q): %v", typ, err)
	}
	return at
}

func TestSetterArgUint8Range(t *testing.T) {
	typ := mustType(t, "uint8")

	arg, err := setterArg(typ, big.NewInt(255))
	if err != nil {
		t.Fatalf("setterArg(uint8, 255): %v", err)
	}
	if got, ok := arg.(uint8); !ok || got != 255 {
		t.Fatalf("arg = %T(%v), want uint8(255)", arg, arg)
	}

	if _, err := setterArg(typ, big.NewInt(256)); err == nil || !strings.Contains(err.Error(), "out of range") {
		t.Fatalf("expected out-of-range error for 256, got %v", err)
	}
	if _, err := setterArg(typ, big.NewInt(-1)); err == nil || !strings.Contains(err.Error(), "out of range") {
		t.Fatalf("expected out-of-range error for -1, got %v", err)
	}
}

func TestSetterArgInt8Range(t *testing.T) {
	typ := mustType(t, "int8")

	arg, err := setterArg(typ, big.NewInt(127))
	if err != nil {
		t.Fatalf("setterArg(int8, 127): %v", err)
	}
	if got, ok := arg.(int8); !ok || got != 127 {
		t.Fatalf("arg = %T(%v), want int8(127)", arg, arg)
	}

	arg, err = setterArg(typ, big.NewInt(-128))
	if err != nil {
		t.Fatalf("setterArg(int8, -128): %v", err)
	}
	if got, ok := arg.(int8); !ok || got != -128 {
		t.Fatalf("arg = %T(%v), want int8(-128)", arg, arg)
	}

	if _, err := setterArg(typ, big.NewInt(128)); err == nil || !strings.Contains(err.Error(), "out of range") {
		t.Fatalf("expected out-of-range error for 128, got %v", err)
	}
	if _, err := setterArg(typ, big.NewInt(-129)); err == nil || !strings.Contains(err.Error(), "out of range") {
		t.Fatalf("expected out-of-range error for -129, got %v", err)
	}
}
