package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/chainform/chainform/internal/config"
	"github.com/chainform/chainform/internal/export"
	"github.com/chainform/chainform/internal/plan"
)

const defaultConfigFile = "chainform.hcl"

func newValidateCmd() *cobra.Command {
	var file string
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate a configuration file without contacting the chain",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.Load(file)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "OK: %d resource(s) on %s (chainId %d)\n",
				len(cfg.Resources), cfg.Chain.Name, cfg.Chain.ChainID)
			return nil
		},
	}
	cmd.Flags().StringVarP(&file, "file", "f", defaultConfigFile, "path to configuration file")
	return cmd
}

func newPlanCmd() *cobra.Command {
	var file string
	var mock bool
	cmd := &cobra.Command{
		Use:   "plan",
		Short: "Show the operations required to converge on-chain state to desired state",
		RunE: func(cmd *cobra.Command, _ []string) error {
			p, err := buildPlan(cmd, file, mock)
			if err != nil {
				return err
			}
			p.Render(cmd.OutOrStdout())
			return nil
		},
	}
	cmd.Flags().StringVarP(&file, "file", "f", defaultConfigFile, "path to configuration file")
	cmd.Flags().BoolVar(&mock, "mock", false, "use the offline demo reader instead of a live RPC endpoint")
	return cmd
}

func newExportCmd() *cobra.Command {
	var file, out, format string
	var mock bool
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Generate a plan and export it as an executable transaction batch",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if format != "safe" {
				return fmt.Errorf("unsupported export format %q (supported: safe)", format)
			}
			p, err := buildPlan(cmd, file, mock)
			if err != nil {
				return err
			}

			w := cmd.OutOrStdout()
			if out != "" && out != "-" {
				f, err := os.Create(out)
				if err != nil {
					return err
				}
				defer f.Close()
				w = f
			}

			if err := export.Safe(w, p, time.Now().UnixMilli()); err != nil {
				return err
			}
			if out != "" && out != "-" {
				fmt.Fprintf(cmd.ErrOrStderr(), "Wrote %d transaction(s) to %s\n", len(p.Operations), out)
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&file, "file", "f", defaultConfigFile, "path to configuration file")
	cmd.Flags().StringVarP(&out, "out", "o", "", "output file (default: stdout)")
	cmd.Flags().StringVar(&format, "format", "safe", "export format (safe)")
	cmd.Flags().BoolVar(&mock, "mock", false, "use the offline demo reader instead of a live RPC endpoint")
	return cmd
}

func newVersionCmd(version string) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the ChainForm version",
		Run: func(cmd *cobra.Command, _ []string) {
			fmt.Fprintln(cmd.OutOrStdout(), version)
		},
	}
}

// buildPlan loads config, opens a reader, and runs one reconciliation pass.
func buildPlan(cmd *cobra.Command, file string, mock bool) (*plan.Plan, error) {
	cfg, err := config.Load(file)
	if err != nil {
		return nil, err
	}
	reader, closeReader, err := openReader(cmd.Context(), cfg, mock)
	if err != nil {
		return nil, err
	}
	defer closeReader()
	return plan.NewPlanner(cfg, reader).Run(cmd.Context())
}
