package cli_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/aleksandarknezevic/chainform/internal/cli"
)

// The vault example exercises arrays, bytes32, bytes and an enum against the
// offline demo reader: two attributes have drifted, the rest match.
func TestPlanCmd_VaultExampleRichTypes(t *testing.T) {
	t.Chdir(repoRoot(t))
	example := filepath.Join("examples", "vault.hcl")

	root := cli.NewRootCmd("test")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"plan", "-f", example, "--mock"})

	err := root.Execute()
	var exitErr *cli.ExitError
	if !errors.As(err, &exitErr) || exitErr.Code != 1 {
		t.Fatalf("expected exit code 1 on drift, got %v", err)
	}
	for _, want := range []string{
		"setKeepers(",
		"0x4444444444444444444444444444444444444444",
		"setMode(1)",
	} {
		if !bytes.Contains(out.Bytes(), []byte(want)) {
			t.Errorf("plan output missing %q:\n%s", want, out.String())
		}
	}
	for _, unwanted := range []string{"setMerkleRoot", "setTierCaps", "setExtraData"} {
		if bytes.Contains(out.Bytes(), []byte(unwanted)) {
			t.Errorf("plan proposed %q for an attribute that matches:\n%s", unwanted, out.String())
		}
	}
}

// Importing a contract with richer types must still round-trip: the snapshot it
// writes has to plan clean against the same state.
func TestImportCmd_RichTypesRoundTripToNoDrift(t *testing.T) {
	t.Chdir(repoRoot(t))
	out := filepath.Join(t.TempDir(), "vault.hcl")

	root := cli.NewRootCmd("test")
	root.SetArgs([]string{
		"import",
		"--address", "0x9a7f8e1c2b3d4e5f60718293a4b5c6d7e8f90123",
		"--abi", filepath.Join("testdata", "vault.abi.json"),
		"--name", "vault", "--chain-id", "1", "--mock", "-o", out,
	})
	if err := root.Execute(); err != nil {
		t.Fatalf("import: %v", err)
	}

	written, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`keepers    = ["0x1111111111111111111111111111111111111111"`,
		`merkleRoot = "0xa1b2c3d4`,
		`extraData  = "0xdeadbeef"`,
		"tierCaps   = [1000, 5000, 10000]",
		"guardians = [",
	} {
		if !bytes.Contains(written, []byte(want)) {
			t.Errorf("imported config missing %q:\n%s", want, written)
		}
	}
	// The struct-valued getter has no value representation, so it is skipped.
	if bytes.Contains(written, []byte("riskParams")) {
		t.Errorf("imported config includes the struct getter:\n%s", written)
	}

	plan := cli.NewRootCmd("test")
	plan.SetArgs([]string{"plan", "-f", out, "--mock"})
	if err := plan.Execute(); err != nil {
		t.Fatalf("plan against imported snapshot: want no drift, got %v", err)
	}
}
