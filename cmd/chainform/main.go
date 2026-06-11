// Command chainform is the ChainForm CLI: declare desired on-chain protocol
// state, detect drift, and export reviewable operations.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/chainform/chainform/internal/cli"
)

// version is overridden at build time via -ldflags "-X main.version=...".
var version = "0.1.0-dev"

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if err := cli.NewRootCmd(version).ExecuteContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
