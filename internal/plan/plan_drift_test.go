package plan_test

import (
	"testing"

	"github.com/chainform/chainform/internal/plan"
	"github.com/chainform/chainform/internal/resource"
)

func TestHasDrift_Operations(t *testing.T) {
	p := &plan.Plan{
		Operations: []resource.Operation{{Method: "setFeeBps"}},
	}
	if !p.HasDrift() {
		t.Fatal("expected drift when operations present")
	}
}

func TestHasDrift_FailedAssertion(t *testing.T) {
	p := &plan.Plan{
		Assertions: []resource.Assertion{{
			Attr: "decimals", Expected: 8, Actual: 6,
		}},
	}
	if !p.HasDrift() {
		t.Fatal("expected drift when assertion failed")
	}
}

func TestHasDrift_None(t *testing.T) {
	p := &plan.Plan{
		Assertions: []resource.Assertion{{
			Attr: "decimals", Expected: 8, Actual: 8,
		}},
	}
	if p.HasDrift() {
		t.Fatal("expected no drift")
	}
}
