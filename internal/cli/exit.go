package cli

// Process exit codes. They are part of the CLI contract: CI can gate on drift
// without parsing output, and still tell drift apart from a broken run.
const (
	// ExitNoDrift is returned when actual state matches desired state.
	ExitNoDrift = 0
	// ExitDrift is returned by `plan` when drift is detected: a managed
	// attribute differs or an `expect` assertion failed. The plan is printed.
	ExitDrift = 1
	// ExitFailure is returned when a command could not run to completion -
	// unreadable config, invalid resource, RPC error. No plan is produced.
	ExitFailure = 2
)

// ExitError signals a non-zero process exit without treating the run as a
// failure. Used when `plan` detects drift but has already printed the plan;
// anything else that fails returns an ordinary error, which the entrypoint
// reports as ExitFailure.
type ExitError struct {
	Code int
}

func (e *ExitError) Error() string { return "" }
