package cli

// ExitError signals a non-zero process exit without treating the run as a
// failure. Used when `plan` detects drift but has already printed the plan.
type ExitError struct {
	Code int
}

func (e *ExitError) Error() string { return "" }
