package main

import (
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"encoding/hex"
	"github.com/tsoniclang/gotots/contracts"
	"github.com/tsoniclang/gotots/internal/census"
	"github.com/tsoniclang/gotots/internal/goenv"
	"github.com/tsoniclang/gotots/internal/ir"
	"github.com/tsoniclang/gotots/internal/pinning"
	"github.com/tsoniclang/gotots/internal/policy"
	"github.com/tsoniclang/gotots/internal/productinputs"
	"github.com/tsoniclang/gotots/internal/profile"
	"github.com/tsoniclang/gotots/internal/schema"
	"github.com/tsoniclang/gotots/internal/staticness"
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
	var corpusGenerated *translate.Generated
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
	run("04-census-denominator-reconciliation", func() (string, []string, error) {
		if firstRun == nil {
			return "blocked", []string{"census did not run"}, nil
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
		corpusGenerated = generated
		// Every production declaration of every owned package must hold a
		// disposition: a support-ledger state, a generation proof, the
		// fold-at-use constant rule, or its package's withholding.
		// Exactly one support disposition per identity: a duplicate or a
		// support/proof state conflict fails closed.
		counts, details, conflicts, unreconciled := reconcileDispositions(prof, firstRun, generated)
		if len(conflicts) > 0 {
			sort.Strings(conflicts)
			if len(conflicts) > 20 {
				conflicts = append(conflicts[:20], fmt.Sprintf("... and %d more", len(conflicts)-20))
			}
			return "fail", conflicts, fmt.Errorf("%d ledger defects (duplicates, conflicts, phantom references)", len(conflicts))
		}
		if counts["unreconciled"] > 0 {
			details = append(details, "first unreconciled declarations:")
			details = append(details, unreconciled...)
			return "fail", details, fmt.Errorf("%d production declarations hold no disposition", counts["unreconciled"])
		}
		return "pass", details, nil
	})
	run("05-declaration-signature-type-completeness", func() (string, []string, error) {
		if firstRun == nil || firstRun.Shapes == nil || corpusGenerated == nil {
			return "blocked", []string{"census shapes or corpus generation did not run"}, nil
		}
		// Independent verification: the census's frontend pass spelled
		// every declaration's canonical signature; each generated
		// function's proof hash must match the census spelling exactly.
		censusSignature := map[string]string{}
		for _, shape := range firstRun.Shapes.Functions {
			digest := sha256.Sum256([]byte(shape.Signature))
			censusSignature[shape.ID] = hex.EncodeToString(digest[:])
		}
		verified, missing := 0, 0
		var mismatches []string
		for _, proof := range corpusGenerated.Proofs {
			if proof.SignatureHash == "" {
				continue // types and variables carry no function signature
			}
			expected, has := censusSignature[proof.ID]
			if !has {
				missing++
				if len(mismatches) < 10 {
					mismatches = append(mismatches, "no census shape for "+proof.ID)
				}
				continue
			}
			if expected != proof.SignatureHash {
				if len(mismatches) < 10 {
					mismatches = append(mismatches, "signature drift at "+proof.ID)
				}
				continue
			}
			verified++
		}
		if len(mismatches) > 0 {
			return "fail", mismatches, fmt.Errorf("%d signature identities failed independent verification", len(mismatches))
		}
		return "pass", []string{
			fmt.Sprintf("function and method signatures independently verified against the census spelling: %d", verified),
		}, nil
	})
	run("06-semantic-ir-operation-class-completeness", func() (string, []string, error) {
		if corpusGenerated == nil {
			return "blocked", []string{"corpus generation did not run"}, nil
		}
		registry, err := contracts.LoadRegistry()
		if err != nil {
			return "fail", nil, err
		}
		// Translation fails closed on any unregistered class, so a
		// completed corpus run proves the join; the gate re-verifies from
		// the proofs and reports registry coverage.
		used := map[string]bool{}
		for _, proof := range corpusGenerated.Proofs {
			for _, operation := range proof.Operations {
				if !registry.Generated(operation) {
					return "fail", []string{proof.ID}, fmt.Errorf("generated body uses unregistered operation class %q", operation)
				}
				used[operation] = true
			}
		}
		var unused []string
		for _, key := range registry.Keys() {
			if !used[key] {
				unused = append(unused, key)
			}
		}
		details := []string{
			fmt.Sprintf("operation classes used by generated bodies: %d", len(used)),
			fmt.Sprintf("reviewed classes not exercised by this corpus: %d", len(unused)),
		}
		return "pass", details, nil
	})
	run("07-ownership-support-state-completeness", func() (string, []string, error) {
		if corpusGenerated == nil {
			return "blocked", []string{"corpus generation did not run"}, nil
		}
		ownership := map[string]int{}
		for path, owner := range corpusGenerated.Ownership {
			switch owner {
			case "generated-core", "generated-language-abi", "generated-external-contracts":
				ownership[owner]++
			default:
				return "fail", []string{path}, fmt.Errorf("file carries unreviewed ownership class %q", owner)
			}
		}
		for path := range corpusGenerated.Files {
			if _, has := corpusGenerated.Ownership[path]; !has {
				return "fail", []string{path}, fmt.Errorf("generated file carries no ownership class")
			}
		}
		states := map[string]int{}
		for _, support := range corpusGenerated.Support {
			switch support.State {
			case ir.SupportGenerated, ir.SupportUnimplemented:
				states[string(support.State)]++
			default:
				return "fail", []string{support.ID}, fmt.Errorf("unit carries unreviewed support state %q", support.State)
			}
			if support.State == ir.SupportUnimplemented && len(support.Sites) == 0 {
				return "fail", []string{support.ID}, fmt.Errorf("unimplemented unit records no sites")
			}
		}
		details := []string{
			fmt.Sprintf("ownership: core %d, language-abi %d, external-contracts %d",
				ownership["generated-core"], ownership["generated-language-abi"], ownership["generated-external-contracts"]),
			fmt.Sprintf("support: generated %d, unimplemented %d (every site on record)",
				states["generated"], states["unimplemented"]),
			fmt.Sprintf("withheld packages: %d (transitive closure over dependents)", len(corpusGenerated.Withheld)),
			"manual and extension ownership classes: none declared yet (no artifacts to verify)",
		}
		return "pass", details, nil
	})
	run("08-fixed-point-representation-verification", func() (string, []string, error) {
		if corpusGenerated == nil {
			return "blocked", []string{"corpus generation did not run"}, nil
		}
		// The slice family's fixed point is live: every generated body's
		// proof records its regions' selections, and the deterministic
		// re-run inside this gate's second generation (stage 09) already
		// proved plan stability. Here the selections themselves are
		// verified: a native selection may appear only with the direct
		// lowering evidence, never alongside carrier operations.
		planned := map[string]int{}
		for _, proof := range corpusGenerated.Proofs {
			for key, value := range proof.Representations {
				if len(key) > 12 && key[:12] == "slice-local:" {
					planned[value]++
				}
			}
		}
		if planned["native-array"]+planned["goslice-carrier"] == 0 {
			return "blocked", []string{"no slice regions recorded; planner evidence missing"}, nil
		}
		// Every custom runtime mechanism present in generated output must
		// have a machine-checked necessity record (spec 10): the record
		// carries the counterexample, ordinary-TypeScript mismatch,
		// rejected alternatives, oracle/mutation/perf evidence,
		// invalidation, acceptance, and reopening — a bare "proven" flag
		// is rejected. The requirement is derived structurally from the
		// emitted output, not a generator flag.
		necessity, err := contracts.LoadNecessityRecords()
		if err != nil {
			return "fail", nil, err
		}
		// Mechanisms are derived from the typed AST of the generated
		// output (resolved identifiers, not substrings), and every
		// record's oracle and mutation evidence must resolve to actual
		// committed test functions — prose is not evidence.
		typescriptModule := filepath.Join(*repoDir, "product", "node_modules", "typescript")
		astReport, err := staticness.VerifyAST(corpusGenerated.Files, typescriptModule)
		if err != nil {
			return "fail", nil, err
		}
		present := make([]string, 0, len(astReport.Mechanisms))
		for mechanism := range astReport.Mechanisms {
			present = append(present, mechanism)
		}
		sort.Strings(present)
		var problems []string
		testIndex, err := committedTestFunctions(*repoDir)
		if err != nil {
			return "fail", nil, err
		}
		for _, mechanism := range present {
			record, ok := necessity.RecordFor(mechanism)
			if !ok {
				problems = append(problems, mechanism+": no necessity record")
				continue
			}
			for _, test := range record.OracleTests {
				if !testIndex[test] {
					problems = append(problems, fmt.Sprintf("%s: oracle evidence %q is not a committed test function", mechanism, test))
				}
			}
			for _, test := range record.MutationTests {
				if !testIndex[test] {
					problems = append(problems, fmt.Sprintf("%s: mutation evidence %q is not a committed test function", mechanism, test))
				}
			}
		}
		if len(problems) > 0 {
			sort.Strings(problems)
			return "fail", problems, fmt.Errorf("%d necessity-record defects", len(problems))
		}
		details := []string{
			fmt.Sprintf("slice regions planned: native-array %d, goslice-carrier %d",
				planned["native-array"], planned["goslice-carrier"]),
			fmt.Sprintf("custom mechanisms with verified necessity records: %v", present),
			"remaining families (pointers, interfaces, strings, maps) plan through their reviewed single candidates",
		}
		return "pass", details, nil
	})
	run("09-deterministic-staged-generation", func() (string, []string, error) {
		return runStagedGenerationGate(*repoDir, *profilePath, *buildProfile, *sourceDir, report, firstRun, corpusGenerated)
	})
	run("10-strict-typescript-staticness", func() (string, []string, error) {
		return runTscGate(*repoDir, *profilePath, *buildProfile, *sourceDir, report, productPin, firstRun)
	})
	blockExecutionStages(blocked)

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
