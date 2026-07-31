package cli_test

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aleksandarknezevic/chainform/internal/cli"
)

// The golden path documented in docs/golden-path.md, run offline: import a real
// contract's state, confirm the snapshot plans clean, change one desired value,
// and export the resulting operation as a Safe batch. Keeping it in the test
// suite means the documented flow cannot silently break.
func TestGoldenPath_ImportPlanEditExport(t *testing.T) {
	t.Chdir(repoRoot(t))

	const (
		factory  = "0x1F98431c8aD98523631AE4a59f267346ea31F984"
		newOwner = "0x000000000000000000000000000000000000dEaD"
		// setOwner(address) selector.
		setOwnerSelector = "0x13af4035"
	)
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "factory.hcl")

	run := func(t *testing.T, args ...string) error {
		t.Helper()
		root := cli.NewRootCmd("test")
		root.SetArgs(args)
		return root.Execute()
	}

	// 1. import
	if err := run(t, "import",
		"--address", factory,
		"--abi", filepath.Join("testdata", "uniswap-v3-factory.abi.json"),
		"--name", "factory", "--chain-id", "1", "--chain-name", "ethereum",
		"--mock", "-o", cfgPath,
	); err != nil {
		t.Fatalf("import: %v", err)
	}

	imported, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	// owner()/setOwner(address) is the only managed attribute in this ABI; the
	// argument-taking getters must not appear.
	if !strings.Contains(string(imported), "owner  = ") && !strings.Contains(string(imported), "owner = ") {
		t.Fatalf("imported config has no owner attribute:\n%s", imported)
	}
	for _, unwanted := range []string{"getPool", "feeAmountTickSpacing", "enableFeeAmount"} {
		if strings.Contains(string(imported), unwanted) {
			t.Errorf("imported config includes %q, which is not a managed attribute:\n%s", unwanted, imported)
		}
	}

	// 2. plan against the same state: no drift
	if err := run(t, "plan", "-f", cfgPath, "--mock"); err != nil {
		t.Fatalf("plan on fresh snapshot: want no drift, got %v", err)
	}

	// 3. change the desired owner
	edited := ownerRegexpReplace(t, string(imported), newOwner)
	if err := os.WriteFile(cfgPath, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}

	// 4. plan now proposes exactly one operation, and exits 1
	err = run(t, "plan", "-f", cfgPath, "--mock")
	var exitErr *cli.ExitError
	if !errors.As(err, &exitErr) || exitErr.Code != cli.ExitDrift {
		t.Fatalf("plan after edit: want drift exit %d, got %v", cli.ExitDrift, err)
	}

	// 5. export the Safe batch
	batchPath := filepath.Join(dir, "batch.json")
	if err := run(t, "export", "-f", cfgPath, "--mock", "-o", batchPath); err != nil {
		t.Fatalf("export: %v", err)
	}

	var batch struct {
		ChainID      string `json:"chainId"`
		Transactions []struct {
			To    string `json:"to"`
			Value string `json:"value"`
			Data  string `json:"data"`
		} `json:"transactions"`
	}
	raw, err := os.ReadFile(batchPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &batch); err != nil {
		t.Fatalf("unmarshal batch: %v\n%s", err, raw)
	}
	if batch.ChainID != "1" {
		t.Errorf("batch chainId = %q, want \"1\"", batch.ChainID)
	}
	if len(batch.Transactions) != 1 {
		t.Fatalf("batch has %d transaction(s), want 1: %s", len(batch.Transactions), raw)
	}
	tx := batch.Transactions[0]
	if tx.To != factory || tx.Value != "0" {
		t.Errorf("tx to/value = %s/%s, want %s/0", tx.To, tx.Value, factory)
	}
	if !strings.HasPrefix(tx.Data, setOwnerSelector) {
		t.Errorf("tx data = %s, want the setOwner selector %s", tx.Data, setOwnerSelector)
	}
	if !strings.HasSuffix(strings.ToLower(tx.Data), strings.ToLower(strings.TrimPrefix(newOwner, "0x"))) {
		t.Errorf("tx data = %s, want it to encode %s", tx.Data, newOwner)
	}
}

// ownerRegexpReplace swaps the imported owner value for a new one, the way a
// human editing the config would.
func ownerRegexpReplace(t *testing.T, config, newOwner string) string {
	t.Helper()
	lines := strings.Split(config, "\n")
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "owner") {
			indent := line[:len(line)-len(strings.TrimLeft(line, " "))]
			lines[i] = indent + `owner = "` + newOwner + `"`
			return strings.Join(lines, "\n")
		}
	}
	t.Fatalf("no owner attribute to edit in:\n%s", config)
	return ""
}
