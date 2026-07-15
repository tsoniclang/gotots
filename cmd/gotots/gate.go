package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/tsoniclang/gotots/internal/census"
	"github.com/tsoniclang/gotots/internal/policy"
	"github.com/tsoniclang/gotots/internal/profile"
)

// GateResult is one acceptance layer's outcome.
type GateResult struct {
	Name       string   `json:"name"`
	Status     string   `json:"status"` // pass | fail | blocked
	DurationMs int64    `json:"durationMs"`
	Details    []string `json:"details,omitempty"`
}

// GateReport is the machine-readable full-gate outcome.
type GateReport struct {
	SchemaVersion int          `json:"schemaVersion"`
	Passed        bool         `json:"passed"`
	Gates         []GateResult `json:"gates"`
}

// runGate executes every applicable acceptance layer in the normative
// order and writes a machine-readable report. It exits nonzero for any
// failed or blocked gate: unresolved dispositions are honest blockers,
// never warnings.
func runGate(args []string) error {
	flags := flag.NewFlagSet("gate", flag.ExitOnError)
	repoDir := flags.String("repo", ".", "gotots repository root")
	profilePath := flags.String("profile", "profiles/tsts/project.json", "project profile path (repo-relative)")
	sourceDir := flags.String("source", "", "pinned source checkout (required)")
	buildProfile := flags.String("build-profile", "linux-amd64", "build profile name")
	reportPath := flags.String("report", "", "gate report output path (required)")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *sourceDir == "" || *reportPath == "" {
		return fmt.Errorf("--source and --report are required")
	}

	report := &GateReport{SchemaVersion: 1, Passed: true}
	run := func(name string, gate func() (string, []string, error)) {
		started := time.Now()
		status, details, err := gate()
		if err != nil {
			status = "fail"
			details = append(details, err.Error())
		}
		if status != "pass" {
			report.Passed = false
		}
		report.Gates = append(report.Gates, GateResult{
			Name: name, Status: status,
			DurationMs: time.Since(started).Milliseconds(),
			Details:    details,
		})
		fmt.Printf("gate %-28s %s\n", name, status)
	}

	command := func(name string, args ...string) (string, []string, error) {
		out, err := runInRepo(*repoDir, name, args...)
		if err != nil {
			return "fail", splitLines(out), err
		}
		return "pass", nil, nil
	}

	// 1. Repository and specification policy (formatting, vetting, the
	// complete unit/fixture/oracle suite with race detection, and the
	// policy gates that run inside it).
	run("format", func() (string, []string, error) {
		// The formatting gate covers exactly the hand-maintained source
		// tree (the shared policy definition), never scratch or evidence
		// directories.
		files, err := policy.SourceFiles(*repoDir)
		if err != nil {
			return "fail", nil, err
		}
		out, err := runInRepo(*repoDir, "gofmt", append([]string{"-l"}, files...)...)
		if err != nil {
			return "fail", splitLines(out), err
		}
		if strings.TrimSpace(out) != "" {
			return "fail", splitLines(out), fmt.Errorf("unformatted files")
		}
		return "pass", nil, nil
	})
	run("vet", func() (string, []string, error) { return command("go", "vet", "./...") })
	run("diff-check", func() (string, []string, error) { return command("git", "diff", "--check") })
	run("test-race", func() (string, []string, error) {
		return command("go", "test", "-count=1", "-race", "./...")
	})

	// 2-8. Input attestation, census dispositions, and deterministic
	// regeneration, all through the census pipeline against the pin.
	var firstRun *census.Result
	run("census-attestation", func() (string, []string, error) {
		prof, err := profile.Load(filepath.Join(*repoDir, filepath.FromSlash(*profilePath)))
		if err != nil {
			return "fail", nil, err
		}
		firstRun, err = census.Run(prof, *sourceDir, *buildProfile)
		if err != nil {
			return "fail", nil, err
		}
		return "pass", nil, nil
	})
	run("census-dispositions", func() (string, []string, error) {
		if firstRun == nil {
			return "blocked", []string{"census did not run"}, nil
		}
		if len(firstRun.Report.Blockers) > 0 {
			return "blocked", firstRun.Report.Blockers, nil
		}
		return "pass", nil, nil
	})
	run("deterministic-regeneration", func() (string, []string, error) {
		if firstRun == nil {
			return "blocked", []string{"census did not run"}, nil
		}
		prof, err := profile.Load(filepath.Join(*repoDir, filepath.FromSlash(*profilePath)))
		if err != nil {
			return "fail", nil, err
		}
		secondRun, err := census.Run(prof, *sourceDir, *buildProfile)
		if err != nil {
			return "fail", nil, err
		}
		staging := filepath.Join(os.TempDir(), fmt.Sprintf("gotots-gate-%d", os.Getpid()))
		defer os.RemoveAll(staging)
		first := filepath.Join(staging, "run1")
		second := filepath.Join(staging, "run2")
		if err := census.WriteReports(firstRun, first); err != nil {
			return "fail", nil, err
		}
		if err := census.WriteReports(secondRun, second); err != nil {
			return "fail", nil, err
		}
		var mismatches []string
		for _, name := range []string{"inventory.json", "census.json", "declarations.json", "externals.json"} {
			firstBytes, err := os.ReadFile(filepath.Join(first, name))
			if err != nil {
				return "fail", nil, err
			}
			secondBytes, err := os.ReadFile(filepath.Join(second, name))
			if err != nil {
				return "fail", nil, err
			}
			if !bytes.Equal(firstBytes, secondBytes) {
				mismatches = append(mismatches, name)
			}
		}
		if len(mismatches) > 0 {
			return "fail", mismatches, fmt.Errorf("nondeterministic reports")
		}
		if _, err := census.VerifyBundle(first); err != nil {
			return "fail", nil, err
		}
		return "pass", nil, nil
	})

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(*reportPath, append(data, '\n'), 0o644); err != nil {
		return err
	}
	if !report.Passed {
		return fmt.Errorf("gate failed; report at %s", *reportPath)
	}
	fmt.Printf("all gates passed; report at %s\n", *reportPath)
	return nil
}

func runInRepo(dir, name string, args ...string) (string, error) {
	command := exec.Command(name, args...)
	command.Dir = dir
	var out bytes.Buffer
	command.Stdout = &out
	command.Stderr = &out
	err := command.Run()
	return out.String(), err
}

func splitLines(out string) []string {
	var lines []string
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if line != "" {
			lines = append(lines, line)
		}
	}
	if len(lines) > 40 {
		lines = lines[len(lines)-40:]
	}
	return lines
}
