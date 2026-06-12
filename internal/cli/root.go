// Package cli wires the ChainForm subcommands together using cobra.
package cli

import (
	"context"
	"errors"

	"github.com/spf13/cobra"

	"github.com/chainform/chainform/internal/chain"
	"github.com/chainform/chainform/internal/config"
)

// NewRootCmd builds the root `chainform` command tree.
func NewRootCmd(version string) *cobra.Command {
	root := &cobra.Command{
		Use:   "chainform",
		Short: "Infrastructure as Code for blockchain protocols",
		Long: "ChainForm manages on-chain protocol state the way Terraform manages cloud\n" +
			"infrastructure: declare desired state in configuration, read actual state\n" +
			"from the chain, detect drift, and generate reviewable operations.",
		SilenceUsage:  true,
		SilenceErrors: false,
	}

	root.AddCommand(
		newValidateCmd(),
		newPlanCmd(),
		newExportCmd(),
		newVersionCmd(version),
	)
	return root
}

// openReader returns a Reader for the configured chain. With --mock it returns
// the offline DemoReader so commands run end-to-end without an RPC endpoint.
func openReader(ctx context.Context, cfg *config.Config, mock bool) (chain.Reader, func(), error) {
	if mock {
		return chain.DemoReader{}, func() {}, nil
	}
	if cfg.Chain.RPC == "" {
		return nil, nil, errors.New(`chain.rpc is empty: set it (e.g. rpc = env("RPC_URL")) or pass --mock`)
	}
	client, err := chain.Dial(ctx, cfg.Chain.RPC)
	if err != nil {
		return nil, nil, err
	}
	return client, client.Close, nil
}
