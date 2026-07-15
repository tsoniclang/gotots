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

// GateReport is the machine-readable full-gate outcome over the
// seventeen normative gates of testing-and-acceptance.md. Passed means
// complete: every gate passes. Blocked gates name their missing
// subsystem — the contract is never silently narrowed to the
// implemented subset.
type GateReport struct {
	SchemaVersion int          `json:"schemaVersion"`
	Passed        bool         `json:"passed"`
	Failed        int          `json:"failed"`
	Blocked       int          `json:"blocked"`
	Gates         []GateResult `json:"gates"`
}

// runGate represents every one of the seventeen normative acceptance
// layers in order as pass, fail, or blocked, and writes a
// machine-readable report. It exits nonzero for any failed gate;
// blocked layers are reported honestly and keep the overall report
// incomplete (Passed=false) without masking implemented-gate failures.
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

	report := &GateReport{SchemaVersion: 2}
	run := func(name string, gate func() (string, []string, error)) {
		started := time.Now()
		status, details, err := gate()
		if err != nil {
			status = "fail"
			details = append(details, err.Error())
		}
		switch status {
		case "fail":
			report.Failed++
		case "blocked":
			report.Blocked++
		}
		report.Gates = append(report.Gates, GateResult{
			Name: name, Status: status,
			DurationMs: time.Since(started).Milliseconds(),
			Details:    details,
		})
		fmt.Printf("gate %-44s %s\n", name, status)
	}
	blocked := func(name, reason string) {
		report.Blocked++
		report.Gates = append(report.Gates, GateResult{
			Name: name, Status: "blocked", Details: []string{reason},
		})
		fmt.Printf("gate %-44s blocked\n", name)
	}

	command := func(name string, args ...string) (string, []string, error) {
		out, err := runInRepo(*repoDir, name, args...)
		if err != nil {
			return "fail", splitLines(out), err
		}
		return "pass", nil, nil
	}

	// Gate 1: repository and specification policy (formatting, vetting,
	// the complete unit/fixture suite with race detection, and the
	// policy gates that run inside it).
	run("01-policy/format", func() (string, []string, error) {
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
	run("01-policy/vet", func() (string, []string, error) { return command("go", "vet", "./...") })
	run("01-policy/diff-check", func() (string, []string, error) { return command("git", "diff", "--check") })
	run("01-policy/tests-race", func() (string, []string, error) {
		return command("go", "test", "-count=1", "-race", "./...")
	})

	// Gates 2-3: input attestation and census dispositions through the
	// census pipeline against the pin.
	var firstRun *census.Result
	run("02-input-attestation", func() (string, []string, error) {
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
	run("03-census-dispositions", func() (string, []string, error) {
		if firstRun == nil {
			return "blocked", []string{"census did not run"}, nil
		}
		if len(firstRun.Report.Blockers) > 0 {
			return "blocked", firstRun.Report.Blockers, nil
		}
		return "pass", nil, nil
	})
	blocked("04-declaration-and-external-contracts",
		"external-contract implementation-status gate not implemented (typed inventory and generated stubs only)")
	blocked("05-body-ir-and-ownership-verification",
		"corpus body-IR acceptance gate not implemented (translate-probe is diagnostic evidence only)")
	blocked("06-representation-plan-verification",
		"two-outcome representation proof not implemented (conservative-v1 is the only plan)")
	blocked("07-lowering-and-generated-artifact-verification",
		"full-unit generated-artifact gate not implemented")
	run("08-deterministic-regeneration", func() (string, []string, error) {
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
	run("09-go-semantic-differential-oracles", func() (string, []string, error) {
		return command("go", "test", "-count=1",
			"-run", "TestOracle|TestReviewRegression|TestCrossPackage|TestExternalStubShape|TestGenerationIsDeterministic",
			"./internal/translate/")
	})
	blocked("10-generated-ts-strict-validation",
		"strict tsc typecheck gate over generated output not implemented (Node execution strips types)")
	blocked("11-no-extension-tsgo-differential-validation",
		"whole-compiler no-extension differential not implemented")
	blocked("12-tsts-extension-and-product-validation",
		"extension seams and product composition not implemented")
	blocked("13-complete-selected-compiler-corpus",
		"corpus generation incomplete (see translate-probe evidence)")
	blocked("14-proof-and-common-downstream-projects",
		"proof projects not implemented")
	blocked("15-self-compilation-and-native-target-probes",
		"self-compilation and C#/Rust target probes not implemented")
	blocked("16-performance-acceptance",
		"performance baselines and regression gates not implemented")
	blocked("17-upgrade-repeatability-proof",
		"upgrade repeatability proof not implemented")

	report.Passed = report.Failed == 0 && report.Blocked == 0
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(*reportPath, append(data, '\n'), 0o644); err != nil {
		return err
	}
	passCount := len(report.Gates) - report.Failed - report.Blocked
	fmt.Printf("gates: %d pass, %d blocked, %d fail; report at %s\n",
		passCount, report.Blocked, report.Failed, *reportPath)
	if report.Failed > 0 {
		return fmt.Errorf("gate failures; report at %s", *reportPath)
	}
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
