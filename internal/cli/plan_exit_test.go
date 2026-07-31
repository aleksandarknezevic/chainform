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

// A run that cannot complete must not look like drift: it returns an ordinary
// error, which the entrypoint maps to ExitFailure rather than ExitDrift. CI
// gates (and the shipped action) rely on telling the two apart.
func TestPlanCmd_FailureIsNotDrift(t *testing.T) {
	t.Chdir(repoRoot(t))

	root := cli.NewRootCmd("test")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"plan", "-f", filepath.Join(t.TempDir(), "missing.hcl"), "--mock"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected an error for a missing configuration file")
	}
	var exitErr *cli.ExitError
	if errors.As(err, &exitErr) {
		t.Fatalf("missing config reported as drift (ExitError code %d)", exitErr.Code)
	}
	if cli.ExitDrift == cli.ExitFailure {
		t.Fatal("drift and failure must use distinct exit codes")
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
