// Acceptance stage 11: semantic-class oracles with a recorded
// test-result ledger. The differential oracle suite executes (Go vs
// generated TypeScript, strict-typechecked), each run appends a ledger
// entry whose covered operation classes are DERIVED from the fixture's
// translation proofs (never hand-declared), and the gate requires every
// implemented class in the support registry to be covered by at least
// one passing oracle. The ledger artifact is retained beside the gate
// report — the executed-result evidence stage 08 binds against.
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tsoniclang/gotots/contracts"
	"github.com/tsoniclang/gotots/internal/oracle"
)

// runSemanticOracleGate executes the oracle suite with ledger recording
// and verifies complete implemented-class coverage.
func runSemanticOracleGate(repoDir, reportPath string) (string, []string, error) {
	registry, err := contracts.LoadRegistry()
	if err != nil {
		return "fail", nil, err
	}
	implemented := map[string]bool{}
	for _, class := range registry.Classes {
		if class.State == "generated" {
			implemented[class.Key] = true
		}
	}
	if len(implemented) == 0 {
		return "fail", nil, fmt.Errorf("support registry lists no implemented classes")
	}

	ledgerPath := strings.TrimSuffix(reportPath, filepath.Ext(reportPath)) + ".oracle-ledger.jsonl"
	// The oracle tests run with each test package's own working
	// directory: the ledger path must be absolute or every oracle.Run
	// fails to open it.
	ledgerPath, err = filepath.Abs(ledgerPath)
	if err != nil {
		return "fail", nil, err
	}
	if err := os.RemoveAll(ledgerPath); err != nil {
		return "fail", nil, err
	}
	// The oracle suite lives in the translate package's differential
	// tests; every oracle.Run call appends to the ledger.
	command := []string{"go", "test", "-count=1", "./internal/translate/"}
	output, testErr := runInRepoEnv(repoDir, map[string]string{"GOTOTS_ORACLE_LEDGER": ledgerPath}, command[0], command[1:]...)
	if testErr != nil {
		tail := output
		if len(tail) > 2000 {
			tail = tail[len(tail)-2000:]
		}
		return "fail", []string{tail}, fmt.Errorf("oracle suite failed: %w", testErr)
	}

	file, err := os.Open(ledgerPath)
	if err != nil {
		return "fail", nil, fmt.Errorf("oracle ledger absent after a green suite: %w", err)
	}
	defer file.Close()
	covered := map[string]bool{}
	entries, cases, mismatches := 0, 0, 0
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)
	for scanner.Scan() {
		var entry oracle.LedgerEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			return "fail", nil, fmt.Errorf("oracle ledger line %d: %w", entries+1, err)
		}
		entries++
		cases += entry.Cases
		if !entry.Match {
			mismatches++
			continue
		}
		for _, class := range entry.Classes {
			covered[class] = true
		}
	}
	if err := scanner.Err(); err != nil {
		return "fail", nil, err
	}
	if mismatches > 0 {
		return "fail", nil, fmt.Errorf("%d oracle runs recorded differential mismatches", mismatches)
	}
	var missing []string
	for class := range implemented {
		if !covered[class] {
			missing = append(missing, class)
		}
	}
	details := []string{
		fmt.Sprintf("oracle runs recorded: %d (%d cases), all differential-matched and strict-typechecked", entries, cases),
		fmt.Sprintf("implemented classes covered by executed oracles: %d/%d", len(implemented)-len(missing), len(implemented)),
		fmt.Sprintf("ledger artifact: %s", filepath.Base(ledgerPath)),
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		shown := missing
		if len(shown) > 15 {
			shown = shown[:15]
		}
		details = append(details, "uncovered implemented classes: "+strings.Join(shown, ", "))
		return "fail", details, fmt.Errorf("%d implemented classes have no passing executed oracle", len(missing))
	}
	return "pass", details, nil
}

// runInRepoEnv is runInRepo with additional environment entries.
func runInRepoEnv(dir string, extra map[string]string, name string, args ...string) (string, error) {
	command := exec.Command(name, args...)
	command.Env = append(os.Environ(), "GOTOTS_ATTESTING=1")
	for key, value := range extra {
		command.Env = append(command.Env, key+"="+value)
	}
	command.Dir = dir
	var out bytes.Buffer
	command.Stdout = &out
	command.Stderr = &out
	err := command.Run()
	return out.String(), err
}
