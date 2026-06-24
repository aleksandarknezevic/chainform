package cli_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/aleksandarknezevic/chainform/internal/cli"
	"github.com/aleksandarknezevic/chainform/internal/config"

	_ "github.com/aleksandarknezevic/chainform/internal/resource"
)

func TestValidateConfig_RejectsUnknownType(t *testing.T) {
	cfg := &config.Config{
		Chain: config.Chain{ChainID: 1},
		Resources: []config.ResourceConfig{{
			Type:    "bogus",
			Name:    "x",
			Address: "0x0000000000000000000000000000000000000001",
		}},
	}
	if err := cli.ValidateConfig(cfg); err == nil {
		t.Fatal("expected error for unknown resource type")
	}
}

func TestValidateConfig_RejectsMissingABI(t *testing.T) {
	cfg := &config.Config{
		Chain: config.Chain{ChainID: 1},
		Resources: []config.ResourceConfig{{
			Type:    "contract",
			Name:    "main",
			Address: "0x0000000000000000000000000000000000000001",
			Spec:    map[string]any{"feeBps": 30},
		}},
	}
	if err := cli.ValidateConfig(cfg); err == nil {
		t.Fatal("expected error for missing abi attribute")
	}
}

func TestValidateConfig_AcceptsProtocol(t *testing.T) {
	cfg := &config.Config{
		Chain: config.Chain{ChainID: 1, Name: "ethereum"},
		Resources: []config.ResourceConfig{{
			Type:    "protocol",
			Name:    "main",
			Address: "0x0000000000000000000000000000000000000001",
			Spec:    map[string]any{"feeBps": 30},
		}},
	}
	if err := cli.ValidateConfig(cfg); err != nil {
		t.Fatalf("ValidateConfig: %v", err)
	}
}

func TestValidateConfig_AcceptsContract(t *testing.T) {
	abiPath := filepath.Join("..", "..", "testdata", "protocol.abi.json")
	cfg := &config.Config{
		Chain: config.Chain{ChainID: 1, Name: "ethereum"},
		Resources: []config.ResourceConfig{{
			Type:    "contract",
			Name:    "main",
			Address: "0x0000000000000000000000000000000000000001",
			Spec: map[string]any{
				"abi":    abiPath,
				"feeBps": 30,
			},
		}},
	}
	if err := cli.ValidateConfig(cfg); err != nil {
		t.Fatalf("ValidateConfig: %v", err)
	}
}

func TestValidateConfig_RejectsUnsettableAttribute(t *testing.T) {
	abiPath := filepath.Join("..", "..", "testdata", "protocol.abi.json")
	cfg := &config.Config{
		Chain: config.Chain{ChainID: 1},
		Resources: []config.ResourceConfig{{
			Type:    "contract",
			Name:    "main",
			Address: "0x0000000000000000000000000000000000000001",
			Spec: map[string]any{
				"abi":  abiPath,
				"name": "Protocol",
			},
		}},
	}
	if err := cli.ValidateConfig(cfg); err == nil {
		t.Fatal("expected error for read-only attribute declared as managed")
	}
}

func writeTempConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "chainform.hcl")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestValidateConfig_AcceptsMainnetExample(t *testing.T) {
	abiLido := filepath.Join("..", "..", "testdata", "lido-steth.abi.json")
	abiFeed := filepath.Join("..", "..", "testdata", "chainlink-eth-usd.abi.json")
	cfg := &config.Config{
		Chain: config.Chain{ChainID: 1, Name: "ethereum"},
		Resources: []config.ResourceConfig{
			{
				Type:    "contract",
				Name:    "lidoSteth",
				Address: "0xae7ab96520DE3A18E5e111B5EaAb095312D7fE84",
				Spec:    map[string]any{"abi": abiLido},
				Expect: map[string]any{
					"getFee": 999, "isStopped": false, "decimals": 18,
					"name": "Liquid staked Ether 2.0", "symbol": "stETH",
				},
			},
			{
				Type:    "contract",
				Name:    "ethUsdOracle",
				Address: "0x5f4eC3Df9cbd43714FE2740f5E3616155c5b8419",
				Spec:    map[string]any{"abi": abiFeed},
				Expect:  map[string]any{"decimals": 8, "description": "ETH / USD", "version": 6},
			},
		},
	}
	if err := cli.ValidateConfig(cfg); err != nil {
		t.Fatalf("ValidateConfig: %v", err)
	}
}

func TestValidateCmd_MainnetExampleFile(t *testing.T) {
	t.Chdir(filepath.Join("..", ".."))
	root := cli.NewRootCmd("test")
	root.SetArgs([]string{"validate", "-f", "examples/mainnet.hcl"})
	if err := root.Execute(); err != nil {
		t.Fatalf("validate mainnet example: %v", err)
	}
}

func TestValidateCmd_OK(t *testing.T) {
	path := writeTempConfig(t, `
chain { chain_id = 1 }
resource "protocol" "main" {
  address = "0x0000000000000000000000000000000000000001"
  feeBps  = 30
}
`)
	root := cli.NewRootCmd("test")
	root.SetArgs([]string{"validate", "-f", path})
	if err := root.Execute(); err != nil {
		t.Fatalf("validate: %v", err)
	}
}

func TestValidateCmd_UnknownType(t *testing.T) {
	path := writeTempConfig(t, `
chain { chain_id = 1 }
resource "bogus" "x" {
  address = "0x0000000000000000000000000000000000000001"
}
`)
	root := cli.NewRootCmd("test")
	root.SetArgs([]string{"validate", "-f", path})
	if err := root.Execute(); err == nil {
		t.Fatal("expected validate to fail for unknown resource type")
	}
}
