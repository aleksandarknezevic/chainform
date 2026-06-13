package cli

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"reflect"

	"github.com/ethereum/go-ethereum/common"
	"github.com/spf13/cobra"
	"github.com/zclconf/go-cty/cty"

	"github.com/chainform/chainform/internal/abi"
	"github.com/chainform/chainform/internal/chain"
	"github.com/chainform/chainform/internal/config"
)

func newImportCmd() *cobra.Command {
	var (
		address   string
		abiPath   string
		name      string
		rpc       string
		chainID   uint64
		chainName string
		out       string
		mock      bool
	)
	cmd := &cobra.Command{
		Use:   "import",
		Short: "Generate a configuration from a live contract's current state",
		Long: "Read a deployed contract's current on-chain state and emit a ChainForm\n" +
			"configuration for it. Attributes with a getter and a setX setter become\n" +
			"managed (top-level) attributes; getter-only values become a read-only\n" +
			"`expect` block. Because managed attributes carry their current values, an\n" +
			"immediate `plan` against the same state reports no drift.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !common.IsHexAddress(address) {
				return fmt.Errorf("invalid --address %q", address)
			}
			if abiPath == "" {
				return fmt.Errorf("--abi is required")
			}
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			if chainID == 0 {
				return fmt.Errorf("--chain-id is required")
			}

			parsed, err := abi.Load(abiPath)
			if err != nil {
				return err
			}

			reader, closeReader, err := importReader(cmd.Context(), rpc, mock)
			if err != nil {
				return err
			}
			defer closeReader()

			managedSet := make(map[string]bool)
			for _, a := range abi.Attributes(parsed) {
				managedSet[a.Name] = true
			}

			addr := common.HexToAddress(address)
			managed := map[string]cty.Value{}
			expect := map[string]cty.Value{}
			for _, g := range abi.Getters(parsed) {
				res, err := reader.Read(cmd.Context(), chain.ViewCall{
					To:      addr,
					Method:  g.Name,
					Outputs: []string{g.OutputType},
				})
				if err != nil {
					return fmt.Errorf("read %s: %w", g.Name, err)
				}
				if len(res) != 1 {
					return fmt.Errorf("read %s: got %d values, want 1", g.Name, len(res))
				}
				cv, err := goToCty(res[0])
				if err != nil {
					return fmt.Errorf("%s: %w", g.Name, err)
				}
				if managedSet[g.Name] {
					managed[g.Name] = cv
				} else {
					expect[g.Name] = cv
				}
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

			if err := config.WriteResource(w, config.ResourceDoc{
				ChainName: chainName,
				ChainID:   chainID,
				Type:      "contract",
				Name:      name,
				Address:   address,
				ABIPath:   abiPath,
				Managed:   managed,
				Expect:    expect,
			}); err != nil {
				return err
			}
			if out != "" && out != "-" {
				fmt.Fprintf(cmd.ErrOrStderr(), "Imported %s into %s (%d managed, %d expected)\n",
					address, out, len(managed), len(expect))
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&address, "address", "", "contract address to import")
	cmd.Flags().StringVar(&abiPath, "abi", "", "path to the contract ABI JSON file")
	cmd.Flags().StringVar(&name, "name", "", "local resource name for the generated config")
	cmd.Flags().StringVar(&rpc, "rpc", "", "JSON-RPC endpoint (or use --mock)")
	cmd.Flags().Uint64Var(&chainID, "chain-id", 0, "EIP-155 chain id")
	cmd.Flags().StringVar(&chainName, "chain-name", "", "human-readable chain label (optional)")
	cmd.Flags().StringVarP(&out, "out", "o", "", "output file (default: stdout)")
	cmd.Flags().BoolVar(&mock, "mock", false, "use the offline demo reader instead of a live RPC endpoint")
	return cmd
}

// importReader opens a reader for import: the offline demo reader with --mock,
// otherwise a live JSON-RPC client.
func importReader(ctx context.Context, rpc string, mock bool) (chain.Reader, func(), error) {
	if mock {
		return chain.DemoReader{}, func() {}, nil
	}
	if rpc == "" {
		return nil, nil, fmt.Errorf("--rpc is required (or pass --mock)")
	}
	client, err := chain.Dial(ctx, rpc)
	if err != nil {
		return nil, nil, err
	}
	return client, client.Close, nil
}

// goToCty converts a value decoded from the chain into a cty value for HCL
// serialization. Integers are written as numbers, addresses as hex strings.
func goToCty(v any) (cty.Value, error) {
	switch x := v.(type) {
	case bool:
		return cty.BoolVal(x), nil
	case string:
		return cty.StringVal(x), nil
	case common.Address:
		return cty.StringVal(x.Hex()), nil
	case *big.Int:
		return cty.NumberVal(new(big.Float).SetInt(x)), nil
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return cty.NumberIntVal(rv.Int()), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return cty.NumberUIntVal(rv.Uint()), nil
	default:
		return cty.NilVal, fmt.Errorf("unsupported value type %T", v)
	}
}
