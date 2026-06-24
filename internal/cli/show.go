package cli

import (
	"context"
	"fmt"
	"io"
	"sort"

	"github.com/spf13/cobra"

	"github.com/aleksandarknezevic/chainform/internal/chain"
	"github.com/aleksandarknezevic/chainform/internal/config"
	"github.com/aleksandarknezevic/chainform/internal/resource"
)

func newShowCmd() *cobra.Command {
	var file string
	var mock bool
	cmd := &cobra.Command{
		Use:   "show",
		Short: "Print actual on-chain state for the configured resources, without diffing",
		Long: "Read and print the current on-chain state of each configured resource.\n" +
			"Unlike `plan`, show performs no diff and proposes no operations — it is a\n" +
			"quick way to inspect a contract. For ABI-driven `contract` resources it\n" +
			"prints every readable getter derived from the ABI.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.Load(file)
			if err != nil {
				return err
			}
			reader, closeReader, err := openReader(cmd.Context(), cfg, mock)
			if err != nil {
				return err
			}
			defer closeReader()

			w := cmd.OutOrStdout()
			for i, rc := range cfg.Resources {
				res, err := resource.Build(rc)
				if err != nil {
					return err
				}
				rows, err := observe(cmd.Context(), res, reader)
				if err != nil {
					return fmt.Errorf("show %s/%s: %w", res.Type(), res.Name(), err)
				}
				if i > 0 {
					fmt.Fprintln(w)
				}
				printResource(w, res, rows)
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&file, "file", "f", defaultConfigFile, "path to configuration file")
	cmd.Flags().BoolVar(&mock, "mock", false, "use the offline demo reader instead of a live RPC endpoint")
	return cmd
}

// row is a single name/value line of inspected state.
type row struct {
	name  string
	value string
}

// observe reads a resource's current state. It prefers the richer Inspector
// view (every readable getter) and falls back to the managed state reported by
// Refresh for resources that do not implement it.
func observe(ctx context.Context, res resource.Resource, r chain.Reader) ([]row, error) {
	if ins, ok := res.(resource.Inspector); ok {
		obs, err := ins.Inspect(ctx, r)
		if err != nil {
			return nil, err
		}
		rows := make([]row, len(obs))
		for i, o := range obs {
			rows[i] = row{name: o.Name, value: resource.FormatValue(o.Value)}
		}
		return rows, nil
	}

	state, err := res.Refresh(ctx, r)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(state))
	for k := range state {
		names = append(names, k)
	}
	sort.Strings(names)
	rows := make([]row, len(names))
	for i, n := range names {
		rows[i] = row{name: n, value: resource.FormatValue(state[n])}
	}
	return rows, nil
}

func printResource(w io.Writer, res resource.Resource, rows []row) {
	fmt.Fprintf(w, "%s.%s @ %s\n", res.Type(), res.Name(), res.Address().Hex())
	if len(rows) == 0 {
		fmt.Fprintln(w, "  (no readable state)")
		return
	}
	width := 0
	for _, r := range rows {
		if len(r.name) > width {
			width = len(r.name)
		}
	}
	for _, r := range rows {
		fmt.Fprintf(w, "  %-*s = %s\n", width, r.name, r.value)
	}
}
