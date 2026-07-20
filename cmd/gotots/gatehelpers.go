// Gate helper utilities: running commands inside the staging repo,
// output splitting, file digests, and census package enumeration.
package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"sort"
	"strings"

	"github.com/tsoniclang/gotots/internal/census"
	"github.com/tsoniclang/gotots/internal/ir"
	"github.com/tsoniclang/gotots/internal/profile"
	"github.com/tsoniclang/gotots/internal/translate"

	"github.com/tsoniclang/gotots/contracts"
	"github.com/tsoniclang/gotots/internal/reconcile"
)

func runInRepo(dir, name string, args ...string) (string, error) {
	command := exec.Command(name, args...)
	// The gate run is the evidence-production context: the attestation-
	// currency policy exempts it (it produces the next report).
	command.Env = append(os.Environ(), "GOTOTS_ATTESTING=1")
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

// ownedProductionPackages lists the owned production package paths in the
// census, for the module-retained denominator.
func ownedProductionPackages(run *census.Result) []string {
	seen := map[string]bool{}
	var out []string
	for _, decl := range run.Report.Declarations {
		if decl.Scope == "production" && !seen[decl.Package] {
			seen[decl.Package] = true
			out = append(out, decl.Package)
		}
	}
	return out
}

// blockExecutionStages records the execution, differential, product,
// performance, upgrade, and publication gates (12-18) as blocked: they
// require runtime and product infrastructure beyond static output.
func blockExecutionStages(blocked func(string, string)) {
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
}

// acceptanceStageNames is the exact ordered 18-stage acceptance
// contract.
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

// reconcileDispositions joins every production declaration in the
// census to exactly one disposition, detects ledger defects (duplicate
// records, conflicting states, phantom generated-file references), and
// computes the per-declaration evidence-stage counts (module-retained
// versus module-retained-blocked) — the valid per-body stage join.
func reconcileDispositions(prof *profile.Profile, firstRun *census.Result, generated *translate.Generated) (map[string]int, []string, []string, []string) {
	covered := map[string]string{}
	var conflicts []string
	supportSeen := map[string]bool{}
	for _, support := range generated.Support {
		// Exactly one support record per identity: even a same-state
		// duplicate is a ledger defect.
		if supportSeen[support.ID] {
			conflicts = append(conflicts, fmt.Sprintf("%s: duplicate support record", support.ID))
		}
		supportSeen[support.ID] = true
		if prior, dup := covered[support.ID]; dup && prior != string(support.State) {
			conflicts = append(conflicts, fmt.Sprintf("%s: %s vs %s", support.ID, prior, support.State))
		}
		covered[support.ID] = string(support.State)
	}
	proofSeen := map[string]bool{}
	for _, proof := range generated.Proofs {
		if proofSeen[proof.ID] {
			conflicts = append(conflicts, fmt.Sprintf("%s: duplicate proof record", proof.ID))
		}
		proofSeen[proof.ID] = true
		// Bidirectional stage verification of the proof's OWN claim:
		// moduleRetained ⇔ package MATERIALIZED ∧ generated file exists
		// (the generator's finalization already proved symbol presence; the
		// gate re-derives the rest independently). A materialized-but-
		// publication-withheld package legitimately retains its bodies —
		// its analyzable file exists and is independently typechecked;
		// publication is a separate, later selection. Only a package that
		// produced no analyzable file (NotMaterialized) must not claim
		// retention.
		_, pkgBlocked := generated.NotMaterialized[proof.Package]
		_, fileExists := generated.Files[proof.GeneratedFile]
		switch {
		case proof.ModuleRetained && pkgBlocked:
			conflicts = append(conflicts, fmt.Sprintf("%s: claims module-retained inside non-materialized package %s", proof.ID, proof.Package))
		case proof.ModuleRetained && (proof.GeneratedFile == "" || !fileExists):
			conflicts = append(conflicts, fmt.Sprintf("%s: claims module-retained with no emitted file", proof.ID))
		case !proof.ModuleRetained && !pkgBlocked && !proof.NoOutput:
			conflicts = append(conflicts, fmt.Sprintf("%s: unretained despite a materialized package and no no-output disposition", proof.ID))
		case proof.NoOutput && proof.GeneratedFile != "":
			conflicts = append(conflicts, fmt.Sprintf("%s: no-output disposition with a generated file reference", proof.ID))
		case proof.ModuleRetained && proof.GeneratedSymbol == "" && !proof.EffectOnly:
			conflicts = append(conflicts, fmt.Sprintf("%s: retained without a generated symbol and no enumerated effect-only reason", proof.ID))
		}
		if proof.GeneratedFile != "" && !fileExists {
			conflicts = append(conflicts, fmt.Sprintf("%s: phantom generated file %s", proof.ID, proof.GeneratedFile))
		}
	}
	for _, proof := range generated.Proofs {
		// A proof states the identity was LOWERED (IR-admitted, its typed IR
		// constructed); it must not contradict a recorded unimplemented
		// support state for the same identity. "Lowered" is an analysis
		// disposition and makes no emission claim — the proof's own
		// module-retention stage is verified separately above.
		if prior, has := covered[proof.ID]; has {
			if prior != string(ir.SupportIRAdmitted) {
				conflicts = append(conflicts, fmt.Sprintf("%s: proof=ir-admitted vs support=%s", proof.ID, prior))
			}
			continue
		}
		covered[proof.ID] = string(ir.SupportIRAdmitted)
	}
	retention := map[string]string{}
	for _, proof := range generated.Proofs {
		switch {
		case proof.ModuleRetained:
			retention[proof.ID] = "retained"
		case proof.NoOutput:
			retention[proof.ID] = "no-output"
		default:
			retention[proof.ID] = "blocked"
		}
	}
	counts := map[string]int{}
	var unreconciled []string
	for _, decl := range firstRun.Report.Declarations {
		if decl.Scope != "production" {
			counts["test-scope"]++
			continue
		}
		if prof != nil {
			if class, _ := prof.Classify(decl.Package); class != profile.ClassOwned {
				counts["unowned"]++
				continue
			}
		}
		if decl.Kind == "const" {
			// A constant is never auto-accepted: its explicit no-output
			// proof must exist (deleting constant evidence fails).
			if retention[decl.ID] == "no-output" {
				counts["const-fold-at-use"]++
				continue
			}
			counts["unreconciled"]++
			if len(unreconciled) < 25 {
				unreconciled = append(unreconciled, decl.ID+" (constant without no-output proof)")
			}
			continue
		}
		if state, has := covered[decl.ID]; has {
			counts[state]++
			// The valid per-declaration stage join, from each proof's own
			// verified stage claim (bidirectionally checked above).
			switch retention[decl.ID] {
			case "retained":
				counts["stage:module-retained"]++
			case "no-output":
				counts["stage:no-output"]++
			default:
				counts["stage:module-retained-blocked"]++
			}
			continue
		}
		if _, withheld := generated.Withheld[decl.Package]; withheld {
			// Withholding is an artifact decision, never a disposition:
			// an undisposed declaration in a withheld package is still
			// missing evidence and fails the gate.
			counts["withheld-undisposed"]++
		}
		counts["unreconciled"]++
		if len(unreconciled) < 25 {
			unreconciled = append(unreconciled, decl.ID)
		}
	}
	// Identity-level ledger reconciliation (no number is ADDED across
	// ledgers; each is a partition of one identity set). The support
	// ledger's unimplemented UNITS partition by declaration kind; every
	// non-declaration unit (a blank import, which is an ordered effect and
	// not a declaration) exactly explains the support ledger's excess over
	// the unimplemented DECLARATION count. An unexplained delta fails.
	supportUnimplByKind := map[string]int{}
	supportUnimplTotal := 0
	for _, b := range generated.Support {
		if string(b.State) != "unimplemented" {
			continue
		}
		supportUnimplTotal++
		kind := "?"
		if parts := strings.SplitN(b.ID, "::", 3); len(parts) >= 2 {
			kind = parts[1]
		}
		supportUnimplByKind[kind]++
	}
	// Declaration-kind unimplemented units in the support ledger must equal
	// the unimplemented DECLARATIONS the reconciliation counted; the
	// difference is exactly the non-declaration (import) units.
	nonDeclKinds := supportUnimplByKind["import"]
	declKindUnimpl := supportUnimplTotal - nonDeclKinds
	if declKindUnimpl != counts["unimplemented"] {
		conflicts = append(conflicts, fmt.Sprintf(
			"ledger reconciliation: support-ledger declaration-kind unimplemented %d != reconciled unimplemented declarations %d (unexplained delta)",
			declKindUnimpl, counts["unimplemented"]))
	}
	details := make([]string, 0, len(counts)+len(unreconciled)+6)
	keys := make([]string, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		details = append(details, fmt.Sprintf("%s: %d", key, counts[key]))
	}
	kindKeys := make([]string, 0, len(supportUnimplByKind))
	for k := range supportUnimplByKind {
		kindKeys = append(kindKeys, k)
	}
	sort.Strings(kindKeys)
	kindParts := make([]string, 0, len(kindKeys))
	for _, k := range kindKeys {
		kindParts = append(kindParts, fmt.Sprintf("%s=%d", k, supportUnimplByKind[k]))
	}
	details = append(details, fmt.Sprintf(
		"ledger reconciliation: support-ledger unimplemented %d = declaration-kinds %d + non-declaration(import) %d; by kind [%s]",
		supportUnimplTotal, declKindUnimpl, nonDeclKinds, strings.Join(kindParts, " ")))
	// Honest evidence stages (spec 00): ir-admitted is not
	// module-retained. Report every denominator explicitly:
	// MATERIALIZED (analyzable file exists, independently typechecked)
	// versus PUBLISHED (in the runnable product) versus withheld.
	publishedPackages, materializedPackages := 0, 0
	for _, path := range ownedProductionPackages(firstRun) {
		if _, blocked := generated.NotMaterialized[path]; !blocked {
			materializedPackages++
		}
		if _, withheld := generated.Withheld[path]; !withheld {
			publishedPackages++
		}
	}
	// Separate output denominators (never conflated with semantic
	// completeness): modules vs analysis artifacts vs placeholder and
	// translator-limit populations, each counted from the emitted text by
	// its exact marker.
	tsModules, bodyArtifacts := 0, 0
	bodyPlaceholders, externPlaceholders, translatorLimitStops, bannedKeyHelpers := 0, 0, 0, 0
	for path, content := range generated.Files {
		switch generated.Ownership[path] {
		case "generated-core":
			tsModules++
			bodyPlaceholders += strings.Count(content, "goBodyUnimplemented(")
			translatorLimitStops += strings.Count(content, "goKeyUnreachable(")
			bannedKeyHelpers += strings.Count(content, "goKeyOpaque(") + strings.Count(content, "goKeyScalar(")
		case "generated-external-contracts":
			externPlaceholders += strings.Count(content, "goExternalUnimplemented(")
		case "analysis-body":
			bodyArtifacts++
		}
	}
	details = append(details,
		fmt.Sprintf("output denominators: %d TS modules, %d per-body analysis artifacts, %d body placeholders, %d external stub placeholders, %d machine-claimed translator-limit stops (goKeyUnreachable; union claims finalize-verified against the value-box log), %d banned key helpers (must be 0)",
			tsModules, bodyArtifacts, bodyPlaceholders, externPlaceholders, translatorLimitStops, bannedKeyHelpers))
	if bannedKeyHelpers > 0 {
		conflicts = append(conflicts, fmt.Sprintf("%d banned key-helper occurrences (goKeyOpaque/goKeyScalar) in generated core", bannedKeyHelpers))
	}
	// The canonical identity-join reconciliation: every count carries a
	// named denominator, every join delta a disposition; a reconciliation
	// defect (an identity on one side without an explaining disposition)
	// is a conflict.
	// The head identity is bound by the enclosing gate report; the
	// identity join itself is revision-agnostic.
	identityReport := reconcile.Build("", firstRun, generated)
	for _, defect := range identityReport.Defects() {
		conflicts = append(conflicts, "identity reconciliation: "+defect.ID+" — "+defect.Disposition)
	}
	for _, denominator := range identityReport.Denominators {
		details = append(details, "denominator "+denominator.Name+": "+fmt.Sprintf("%d", denominator.Count)+" ("+denominator.Definition+")")
	}

	// The §C manual-completion contract: every unimplemented BODY carries
	// exactly one reviewed disposition, with its exact blocker-construct
	// set — an undisposed body, a stale entry, or blocker drift is a
	// conflict.
	completions, err := contracts.LoadManualCompletions()
	if err != nil {
		conflicts = append(conflicts, err.Error())
	} else {
		unimplementedBodies := map[string][]string{}
		for _, s := range generated.Support {
			if string(s.State) != "unimplemented" {
				continue
			}
			constructSet := map[string]bool{}
			for _, site := range s.Sites {
				constructSet[site.Construct] = true
			}
			constructs := make([]string, 0, len(constructSet))
			for construct := range constructSet {
				constructs = append(constructs, construct)
			}
			sort.Strings(constructs)
			unimplementedBodies[s.ID] = constructs
		}
		if err := completions.VerifyDispositions(unimplementedBodies); err != nil {
			conflicts = append(conflicts, err.Error())
		}
		byResolution := map[string]int{}
		for _, d := range completions.Dispositions {
			byResolution[d.Resolution]++
		}
		details = append(details, fmt.Sprintf(
			"manual-completion contract: %d unimplemented bodies disposed (%d accepted-manual, %d product-policy, %d deferred), 0 undisposed",
			len(completions.Dispositions), byResolution["accepted-manual"], byResolution["product-policy"], byResolution["deferred"]))
	}
	details = append(details,
		fmt.Sprintf("evidence-stage ir-admitted (declarations disposed): %d", counts[string(ir.SupportIRAdmitted)]+counts["accepted-manual"]),
		fmt.Sprintf("evidence-stage module-retained (disposed, materialized artifact): %d", counts["stage:module-retained"]),
		fmt.Sprintf("evidence-stage module-retained-blocked (disposed, no materialized artifact): %d", counts["stage:module-retained-blocked"]),
		fmt.Sprintf("packages materialized (analyzable, typechecked): %d", materializedPackages),
		fmt.Sprintf("packages published (runnable product): %d", publishedPackages),
		fmt.Sprintf("packages withheld from publication (honest unimplemented): %d", len(generated.Withheld)),
		fmt.Sprintf("emitter defects: %d", len(generated.EmitterDefects)))
	return counts, details, conflicts, unreconciled
}

// committedTestFunctions indexes every Go test function name in the
// repository's committed test files, the resolution target for
// necessity-record oracle and mutation evidence.
func committedTestFunctions(repoDir string) (map[string]bool, error) {
	index := map[string]bool{}
	err := filepath.WalkDir(repoDir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			name := entry.Name()
			if name == ".git" || name == "node_modules" || name == ".temp" || name == ".analysis" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(entry.Name(), "_test.go") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, line := range strings.Split(string(data), "\n") {
			trimmed := strings.TrimSpace(line)
			if rest, ok := strings.CutPrefix(trimmed, "func Test"); ok {
				if paren := strings.Index(rest, "("); paren > 0 {
					index["Test"+rest[:paren]] = true
				}
			}
		}
		return nil
	})
	return index, err
}

// binaryBuildProvenance reads the executing binary's VCS stamp: the
// revision it was built from and whether the tree was modified. Empty
// when the build carries no stamp (e.g. go run of a dirty local tree
// without VCS info).
func binaryBuildProvenance() (revision string, modified bool) {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "", false
	}
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			revision = setting.Value
		case "vcs.modified":
			modified = setting.Value == "true"
		}
	}
	return revision, modified
}

// checkBinaryProvenance fails closed unless the executing binary's
// VCS-stamped revision exists and matches the checkout: an unstamped or
// stale binary can never label a gate report.
func checkBinaryProvenance(binaryRevision, checkoutRevision string) error {
	if binaryRevision == "" {
		return fmt.Errorf("executing binary carries no VCS build stamp; build inside the repository so provenance is attested")
	}
	if binaryRevision != checkoutRevision {
		return fmt.Errorf("executing binary was built from %s but the checkout is %s; rebuild before gating", binaryRevision, checkoutRevision)
	}
	return nil
}

// emitterDefectDetails renders the emitter-defect hard-failure evidence:
// any body or package whose emission failed on a non-typed error is a
// compiler defect with its exact identity on record, and every gate that
// consumes generated output fails while one exists.
func emitterDefectDetails(generated *translate.Generated) ([]string, bool) {
	if len(generated.EmitterDefects) == 0 {
		return nil, false
	}
	details := make([]string, 0, len(generated.EmitterDefects)+1)
	details = append(details, fmt.Sprintf("emitter defects: %d (hard failure)", len(generated.EmitterDefects)))
	for i, defect := range generated.EmitterDefects {
		if i >= 20 {
			details = append(details, fmt.Sprintf("... and %d more", len(generated.EmitterDefects)-20))
			break
		}
		details = append(details, "  "+defect.ID+": "+defect.Err)
	}
	return details, true
}
