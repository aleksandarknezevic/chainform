// Command chainform is the ChainForm CLI: declare desired on-chain protocol
// state, detect drift, and export reviewable operations.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"

	"github.com/aleksandarknezevic/chainform/internal/cli"
)

// version is overridden at build time via -ldflags "-X main.version=...".
var version = "0.1.0-dev"

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if err := cli.NewRootCmd(version).ExecuteContext(ctx); err != nil {
		var exitErr *cli.ExitError
		if errors.As(err, &exitErr) {
			// Drift: the plan was printed, the run itself succeeded.
			os.Exit(exitErr.Code)
		}
		// A command that could not complete exits with a code of its own, so CI
		// never mistakes a broken run for detected drift.
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(cli.ExitFailure)
	}
}
