package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tsoniclang/gotots/internal/census"
	"github.com/tsoniclang/gotots/internal/goenv"
	"github.com/tsoniclang/gotots/internal/pinning"
	"github.com/tsoniclang/gotots/internal/policy"
	"github.com/tsoniclang/gotots/internal/productinputs"
	"github.com/tsoniclang/gotots/internal/profile"
	"github.com/tsoniclang/gotots/internal/schema"
	"github.com/tsoniclang/gotots/internal/translate"
)

// GateResult is one acceptance layer's outcome.
type GateResult struct {
	Name    string   `json:"name"`
	Status  string   `json:"status"` // pass | fail | blocked
	Details []string `json:"details,omitempty"`
}

// GateReport is the machine-readable full-gate outcome over the
// ordered stages in docs/spec/11-testing-acceptance.md. Passed means
// complete: every gate passes. Blocked gates name their missing
// subsystem — the contract is never silently narrowed to the
// implemented subset.
type GateReport struct {
	SchemaVersion  int          `json:"schemaVersion"`
	ExpectedStages int          `json:"expectedStages"`
	ReportedStages int          `json:"reportedStages"`
	PassedStages   int          `json:"passedStages"`
	Failed         int          `json:"failed"`
	Blocked        int          `json:"blocked"`
	MissingStages  int          `json:"missingStages"`
	Passed         bool         `json:"passed"`
	Inputs         GateInputs   `json:"inputs"`
	Gates          []GateResult `json:"gates"`
}

type GateInputs struct {
	ImplementationRevision      string   `json:"implementationRevision"`
	SpecificationManifestSha256 string   `json:"specificationManifestSha256"`
	DecisionRegistrySha256      string   `json:"decisionRegistrySha256"`
	SourceRevision              string   `json:"sourceRevision,omitempty"`
	ProfileSha256               string   `json:"profileSha256,omitempty"`
	BuildProfile                string   `json:"buildProfile"`
	GoVersion                   string   `json:"goVersion,omitempty"`
	GoExecutableSha256          string   `json:"goExecutableSha256,omitempty"`
	GorootSourceSha256          string   `json:"gorootSourceSha256,omitempty"`
	TypescriptCompiler          string   `json:"typescriptCompiler,omitempty"`
	JavascriptRuntime           string   `json:"javascriptRuntime,omitempty"`
	NodeExecutableSha256        string   `json:"nodeExecutableSha256,omitempty"`
	ModuleResolverPolicy        string   `json:"moduleResolverPolicy,omitempty"`
	ModuleResolverSha256        string   `json:"moduleResolverSha256,omitempty"`
	StrictConfigSha256          string   `json:"strictConfigSha256,omitempty"`
	HelperRuntimeSha256         string   `json:"helperRuntimeSha256,omitempty"`
	Missing                     []string `json:"missing"`
}

var acceptanceStageNames = []string{
	"01-repository-specification-policy",
	"02-input-toolchain-profile-attestation",
	"03-selected-scope-dependency-closure",
	"04-census-denominator-reconciliation",
	"05-declaration-signature-type-completeness",
	"06-semantic-ir-operation-class-completeness",
	"07-ownership-support-state-completeness",
	"08-fixed-point-representation-verification",
	"09-deterministic-staged-generation",
	"10-strict-typescript-staticness",
	"11-semantic-oracles",
	"12-generated-packages-selected-tests",
	"13-no-extension-tsgo-differential",
	"14-extensions-assembled-product",
	"15-compiler-corpus-proof-common-projects",
	"16-performance",
	"17-source-update-repeatability",
	"18-complete-product-publication",
}

// runGate represents the exact 18-stage acceptance contract as pass, fail, or
// blocked and writes a machine-readable report. It exits nonzero unless every
// required stage passes; blocked is an incomplete build, never command success.
func runGate(args []string) error {
	flags := flag.NewFlagSet("gate", flag.ExitOnError)
	repoDir := flags.String("repo", ".", "gotots repository root")
	profilePath := flags.String("profile", "profiles/tsts/project.json", "project profile path (repo-relative)")
	sourceDir := flags.String("source", "", "pinned source checkout (required)")
	buildProfile := flags.String("build-profile", "linux-amd64", "build profile name")
	reportPath := flags.String("report", "", "gate report output path (required)")
	productToolchainPath := flags.String("product-toolchain", "pins/product-toolchain.json", "product toolchain pin path (repo-relative)")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *sourceDir == "" || *reportPath == "" {
		return fmt.Errorf("--source and --report are required")
	}

	implementationRevision, err := runInRepo(*repoDir, "git", "rev-parse", "HEAD")
	if err != nil {
		return fmt.Errorf("resolve implementation revision: %w", err)
	}
	specificationDigest, err := digestFile(filepath.Join(*repoDir, "docs", "spec", "manifest.json"))
	if err != nil {
		return err
	}
	decisionDigest, err := digestFile(filepath.Join(*repoDir, "docs", "decisions", "registry.json"))
	if err != nil {
		return err
	}
	report := &GateReport{
		SchemaVersion:  4,
		ExpectedStages: len(acceptanceStageNames),
		Inputs: GateInputs{
			ImplementationRevision:      strings.TrimSpace(implementationRevision),
			SpecificationManifestSha256: specificationDigest,
			DecisionRegistrySha256:      decisionDigest,
			BuildProfile:                *buildProfile,
			Missing: []string{
				"typescriptCompiler",
				"javascriptRuntime",
				"moduleResolver",
				"strictTypeScriptConfig",
				"generatedHelperRuntime",
			},
		},
	}
	run := func(name string, gate func() (string, []string, error)) {
		status, details, err := gate()
		if err != nil {
			status = "fail"
			details = append(details, err.Error())
		}
		switch status {
		case "pass":
		case "fail":
			report.Failed++
		case "blocked":
			report.Blocked++
		default:
			status = "fail"
			report.Failed++
			details = append(details, "gate returned invalid status")
		}
		report.Gates = append(report.Gates, GateResult{
			Name: name, Status: status, Details: details,
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

	// Gate 1: repository and specification policy (formatting, vetting,
	// diff hygiene, and the complete unit/fixture suite with race detection).
	run("01-repository-specification-policy", func() (string, []string, error) {
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
		for _, step := range []struct {
			name string
			args []string
		}{
			{name: "go", args: []string{"vet", "./..."}},
			{name: "git", args: []string{"diff", "--check"}},
			{name: "go", args: []string{"test", "-count=1", "./..."}},
			{name: "go", args: []string{"test", "-count=1", "-race", "./..."}},
		} {
			stepOut, stepErr := runInRepo(*repoDir, step.name, step.args...)
			if stepErr != nil {
				return "fail", splitLines(stepOut), stepErr
			}
		}
		statusOut, statusErr := runInRepo(*repoDir, "git", "status", "--porcelain")
		if statusErr != nil {
			return "fail", splitLines(statusOut), statusErr
		}
		if strings.TrimSpace(statusOut) != "" {
			return "fail", splitLines(statusOut), fmt.Errorf("repository is not clean; implementation revision does not attest the working tree")
		}
		return "pass", nil, nil
	})

	// Gates 2-4: input attestation and census dispositions through the
	// census pipeline against the pin.
	var firstRun *census.Result
	var productPin *productinputs.Pin
	run("02-input-toolchain-profile-attestation", func() (string, []string, error) {
		prof, err := profile.Load(filepath.Join(*repoDir, filepath.FromSlash(*profilePath)))
		if err != nil {
			return "fail", nil, err
		}
		firstRun, err = census.Run(prof, *sourceDir, *buildProfile)
		if err != nil {
			return "fail", nil, err
		}
		report.Inputs.SourceRevision = firstRun.Report.Pin.Revision
		report.Inputs.ProfileSha256 = firstRun.Report.Profile.Hash
		report.Inputs.GoVersion = firstRun.Report.Pin.Toolchain.Version
		report.Inputs.GoExecutableSha256 = firstRun.Report.Pin.Toolchain.GoExecutableSha256
		report.Inputs.GorootSourceSha256 = firstRun.Report.Pin.Toolchain.GorootSrcDigest

		// The non-Go half of the closed input contract: TypeScript
		// compiler, JavaScript runtime, module resolver, strict
		// configuration, and generated-helper-runtime identities, all
		// declared in the committed pin and verified against their
		// materializations.
		productPin, err = productinputs.Load(filepath.Join(*repoDir, filepath.FromSlash(*productToolchainPath)))
		if err != nil {
			return "fail", nil, err
		}
		if err := productPin.Verify(*repoDir); err != nil {
			return "fail", nil, err
		}
		report.Inputs.TypescriptCompiler = productPin.TypescriptCompiler.Package + "@" + productPin.TypescriptCompiler.Version
		report.Inputs.JavascriptRuntime = productPin.JavascriptRuntime.Name + " " + productPin.Verified.NodeVersion
		report.Inputs.NodeExecutableSha256 = productPin.Verified.NodeExecutableSha256
		report.Inputs.ModuleResolverPolicy = productPin.ModuleResolver.Policy
		report.Inputs.ModuleResolverSha256 = productPin.ModuleResolver.LoaderSha256
		report.Inputs.StrictConfigSha256 = productPin.StrictConfig.Sha256
		report.Inputs.HelperRuntimeSha256 = productPin.HelperRuntime.Sha256
		report.Inputs.Missing = []string{}
		return "pass", []string{
			"typescript compiler identity declared: " + report.Inputs.TypescriptCompiler + " (materialization verified at stage 10)",
			"javascript runtime verified: " + report.Inputs.JavascriptRuntime,
		}, nil
	})
	run("03-selected-scope-dependency-closure", func() (string, []string, error) {
		if firstRun == nil {
			return "blocked", []string{"census did not run"}, nil
		}
		// The census fails closed on GOTOTS_SCOPE_DEPENDENCY_OUTSIDE, so a
		// completed run proves the selected dependency closure avoids every
		// outside-universe root; the filter evidence is the per-category
		// counts of pre-census-filtered files.
		universe := firstRun.Inventory.Universe
		details := []string{fmt.Sprintf("outside-universe files filtered before census: %d", universe.OutsideUniverseFiles)}
		categories := make([]string, 0, len(universe.OutsideUniverse))
		for category := range universe.OutsideUniverse {
			categories = append(categories, category)
		}
		sort.Strings(categories)
		for _, category := range categories {
			details = append(details, fmt.Sprintf("  %s: %d files", category, universe.OutsideUniverse[category]))
		}
		if universe.OutsideUniverseFiles == 0 && len(categories) == 0 {
			details = append(details, "profile declares no outside-universe roots")
		}
		return "pass", details, nil
	})
	blocked("04-census-denominator-reconciliation",
		"complete operation, ownership, support, and selected-test ledgers are not implemented")
	blocked("05-declaration-signature-type-completeness",
		"complete independently verified declaration/signature/type-identity gate not implemented")
	blocked("06-semantic-ir-operation-class-completeness",
		"complete semantic-IR operation schema/verifier not implemented")
	blocked("07-ownership-support-state-completeness",
		"complete generated/manual/unimplemented/external/extension ownership gate not implemented")
	blocked("08-fixed-point-representation-verification",
		"constraint propagation, representation, necessity, and boundary proof verifier not implemented")
	run("09-deterministic-staged-generation", func() (string, []string, error) {
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
		return "blocked", []string{"census evidence is deterministic; complete generated TypeScript staged emission is not implemented"}, nil
	})
	run("10-strict-typescript-staticness", func() (string, []string, error) {
		if firstRun == nil || productPin == nil {
			return "blocked", []string{"input attestation did not complete"}, nil
		}
		// The compiler materialization must match the declared identity
		// before it is trusted to judge staticness.
		tscJs, err := filepath.Abs(filepath.Join(*repoDir, "product", "node_modules", "typescript", "lib", "tsc.js"))
		if err != nil {
			return "fail", nil, err
		}
		tscDigest, err := digestFile(tscJs)
		if err != nil {
			return "blocked", []string{"typescript compiler not materialized under product/node_modules (install the pinned typescript version)"}, nil
		}
		if tscDigest != productPin.TypescriptCompiler.TscJsSha256 {
			return "fail", nil, fmt.Errorf("materialized tsc.js digest %s does not match pinned %s", tscDigest, productPin.TypescriptCompiler.TscJsSha256)
		}
		prof, err := profile.Load(filepath.Join(*repoDir, filepath.FromSlash(*profilePath)))
		if err != nil {
			return "fail", nil, err
		}
		build, err := prof.BuildProfileByName(*buildProfile)
		if err != nil {
			return "fail", nil, err
		}
		resolved, err := pinning.VerifyToolchain(prof.Pin)
		if err != nil {
			return "fail", nil, err
		}
		env := resolved.Environ(goenv.EnvOptions{
			GOOS: build.GOOS, GOARCH: build.GOARCH,
			GOAMD64: build.GOAMD64, GOARM64: build.GOARM64,
		})
		generated, err := translate.Corpus(prof, env, *sourceDir, translate.Options{
			SourceRevision: report.Inputs.SourceRevision,
			ProfileHash:    report.Inputs.ProfileSha256,
		})
		if err != nil {
			return "fail", nil, err
		}
		staging, err := os.MkdirTemp("", "gotots-tsc-")
		if err != nil {
			return "fail", nil, err
		}
		defer os.RemoveAll(staging)
		for path, content := range generated.Files {
			target := filepath.Join(staging, filepath.FromSlash(path))
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return "fail", nil, err
			}
			if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
				return "fail", nil, err
			}
		}
		strictConfig, err := os.ReadFile(filepath.Join(*repoDir, "product", "tsconfig.strict.json"))
		if err != nil {
			return "fail", nil, err
		}
		if err := os.WriteFile(filepath.Join(staging, "tsconfig.json"), strictConfig, 0o644); err != nil {
			return "fail", nil, err
		}
		// The generated output is strict ESM.
		if err := os.WriteFile(filepath.Join(staging, "package.json"), []byte("{\n  \"type\": \"module\"\n}\n"), 0o644); err != nil {
			return "fail", nil, err
		}
		out, err := runInRepo(staging, "node", tscJs, "-p", ".")
		if err != nil {
			lines := splitLines(out)
			if len(lines) > 40 {
				lines = append(lines[:40], fmt.Sprintf("... and %d more lines", len(lines)-40))
			}
			return "fail", lines, fmt.Errorf("strict typecheck rejected the generated output")
		}
		return "pass", []string{
			fmt.Sprintf("strict tsc (%s@%s) accepted %d generated files", productPin.TypescriptCompiler.Package, productPin.TypescriptCompiler.Version, len(generated.Files)),
			fmt.Sprintf("withheld packages (honest unimplemented, not typechecked): %d", len(generated.Withheld)),
		}, nil
	})
	blocked("11-semantic-oracles",
		"ordered static-output prerequisites are incomplete; local oracle tests run under gate 01 only")
	blocked("12-generated-packages-selected-tests",
		"complete generated-package build and translated-test gate not implemented")
	blocked("13-no-extension-tsgo-differential",
		"whole-compiler no-extension differential not implemented")
	blocked("14-extensions-assembled-product",
		"extension seams and product composition not implemented")
	blocked("15-compiler-corpus-proof-common-projects",
		"complete compiler corpus, proof projects, and common projects not implemented")
	blocked("16-performance",
		"performance baselines and regression gates not implemented")
	blocked("17-source-update-repeatability",
		"upgrade repeatability proof not implemented")
	blocked("18-complete-product-publication",
		"atomic complete product publication gate not implemented")

	report.ReportedStages = len(report.Gates)
	report.MissingStages = report.ExpectedStages - report.ReportedStages
	if report.MissingStages < 0 {
		report.MissingStages = 0
	}
	if report.ReportedStages != report.ExpectedStages {
		return fmt.Errorf("internal gate schema error: reported %d stages; want %d", len(report.Gates), len(acceptanceStageNames))
	}
	for index, gate := range report.Gates {
		if gate.Name != acceptanceStageNames[index] {
			return fmt.Errorf("internal gate schema error: stage %d is %q; want %q", index+1, gate.Name, acceptanceStageNames[index])
		}
	}

	report.PassedStages = report.ReportedStages - report.Failed - report.Blocked
	report.Passed = report.Failed == 0 && report.Blocked == 0 && report.MissingStages == 0
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	reportSchema, err := schema.Load(filepath.Join(*repoDir, "schemas", "gate-report.schema.json"))
	if err != nil {
		return err
	}
	if err := reportSchema.Validate(data); err != nil {
		return fmt.Errorf("gate report does not satisfy its schema: %w", err)
	}
	if err := os.WriteFile(*reportPath, append(data, '\n'), 0o644); err != nil {
		return err
	}
	fmt.Printf("gates: %d pass, %d blocked, %d fail; report at %s\n",
		report.PassedStages, report.Blocked, report.Failed, *reportPath)
	if report.Failed > 0 || report.Blocked > 0 {
		return fmt.Errorf("acceptance incomplete: %d failed, %d blocked; report at %s",
			report.Failed, report.Blocked, *reportPath)
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

func digestFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read attested input %s: %w", path, err)
	}
	digest := sha256.Sum256(data)
	return fmt.Sprintf("%x", digest), nil
}
