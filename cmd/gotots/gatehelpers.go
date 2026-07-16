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
	"sort"
	"strings"

	"github.com/tsoniclang/gotots/internal/census"
	"github.com/tsoniclang/gotots/internal/profile"
	"github.com/tsoniclang/gotots/internal/translate"
)

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
// performance, upgrade, and publication gates (11-18) as blocked: they
// require runtime and product infrastructure beyond static output.
func blockExecutionStages(blocked func(string, string)) {
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
		// Every retained generated-file reference must resolve to an
		// emitted file — no phantom references.
		if proof.GeneratedFile != "" {
			if _, exists := generated.Files[proof.GeneratedFile]; !exists {
				conflicts = append(conflicts, fmt.Sprintf("%s: phantom generated file %s", proof.ID, proof.GeneratedFile))
			}
		}
	}
	for _, proof := range generated.Proofs {
		// A proof states "generated"; it must not contradict a
		// recorded unimplemented support state for the same identity.
		if prior, has := covered[proof.ID]; has {
			if prior != "generated" {
				conflicts = append(conflicts, fmt.Sprintf("%s: proof=generated vs support=%s", proof.ID, prior))
			}
			continue
		}
		covered[proof.ID] = "generated"
	}
	counts := map[string]int{}
	var unreconciled []string
	for _, decl := range firstRun.Report.Declarations {
		if decl.Scope != "production" {
			counts["test-scope"]++
			continue
		}
		if class, _ := prof.Classify(decl.Package); class != profile.ClassOwned {
			counts["unowned"]++
			continue
		}
		if decl.Kind == "const" {
			counts["const-fold-at-use"]++
			continue
		}
		if state, has := covered[decl.ID]; has {
			counts[state]++
			// The valid per-declaration stage join: a disposed
			// declaration is module-retained-blocked exactly when its
			// package is withheld.
			if _, withheld := generated.Withheld[decl.Package]; withheld {
				counts["stage:module-retained-blocked"]++
			} else {
				counts["stage:module-retained"]++
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
	details := make([]string, 0, len(counts)+len(unreconciled)+4)
	keys := make([]string, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		details = append(details, fmt.Sprintf("%s: %d", key, counts[key]))
	}
	// Honest evidence stages (spec 00): ir-admitted is not
	// module-retained. Report both denominators explicitly.
	emittedPackages := 0
	for _, path := range ownedProductionPackages(firstRun) {
		if _, withheld := generated.Withheld[path]; !withheld {
			emittedPackages++
		}
	}
	details = append(details,
		fmt.Sprintf("evidence-stage ir-admitted (declarations disposed): %d", counts["generated"]+counts["accepted-manual"]),
		fmt.Sprintf("evidence-stage module-retained (disposed, in emitted packages): %d", counts["stage:module-retained"]),
		fmt.Sprintf("evidence-stage module-retained-blocked (disposed, in withheld packages): %d", counts["stage:module-retained-blocked"]),
		fmt.Sprintf("packages emitted (module-retained): %d", emittedPackages),
		fmt.Sprintf("packages withheld (honest unimplemented): %d", len(generated.Withheld)))
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
