package resource

// Assertion is a read-only invariant: an attribute with a getter but no setter
// whose on-chain value is compared against an expected value declared in an
// `expect` block. It can drift, but because there is no setter it can never be
// converged — so it is reported, never turned into an Operation. This keeps the
// planning invariants intact: planning sends nothing, and an assertion is not
// an operation.
type Assertion struct {
	Resource string // local resource name
	Attr     string // attribute / getter name, e.g. "decimals"
	Type     string // ABI type, e.g. "uint8"
	Expected any    // canonical expected value
	Actual   any    // canonical on-chain value
}

// Satisfied reports whether the on-chain value matches the expectation.
func (a Assertion) Satisfied() bool {
	return valueEqual(a.Actual, a.Expected)
}

// Asserter is an optional Resource capability: evaluate read-only expectations
// against the current on-chain state and report findings. Findings are surfaced
// by `plan` as warnings but are never executed.
type Asserter interface {
	// Assert compares the resource's declared expectations against current
	// state (already read by Refresh) and returns one Assertion per expectation.
	Assert(current State) ([]Assertion, error)
}
