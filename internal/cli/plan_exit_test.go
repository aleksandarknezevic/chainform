package cli_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/aleksandarknezevic/chainform/internal/cli"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	return filepath.Join("..", "..")
}

func TestPlanCmd_ExitCodeOnDrift(t *testing.T) {
	rootDir := repoRoot(t)
	t.Chdir(rootDir)
	example := filepath.Join("examples", "contract.hcl")
	if _, err := os.Stat(example); err != nil {
		t.Fatalf("stat example: %v", err)
	}

	root := cli.NewRootCmd("test")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"plan", "-f", example, "--mock"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected non-nil error (exit code) when drift is present")
	}
	var exitErr *cli.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected ExitError, got %T: %v", err, err)
	}
	if exitErr.Code != 1 {
		t.Errorf("exit code = %d, want 1", exitErr.Code)
	}
	if !bytes.Contains(out.Bytes(), []byte("setFeeBps")) {
		t.Errorf("plan output missing operations:\n%s", out.String())
	}
}

func TestPlanCmd_ExitCodeNoDrift(t *testing.T) {
	rootDir := repoRoot(t)
	t.Chdir(rootDir)
	abiPath := filepath.Join("testdata", "protocol.abi.json")
	content := `version = "1"

chain {
  name     = "ethereum"
  chain_id = 1
}

resource "contract" "main" {
  address = "0xF38D8Be3E0A7B3c94C00a25b4A443ca062f343f5"
  abi     = "` + abiPath + `"

  feeBps = 50
  paused = true
}
`
	path := filepath.Join(t.TempDir(), "chainform.hcl")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	root := cli.NewRootCmd("test")
	root.SetArgs([]string{"plan", "-f", path, "--mock"})
	if err := root.Execute(); err != nil {
		t.Fatalf("expected exit 0 on no drift, got: %v", err)
	}
}
