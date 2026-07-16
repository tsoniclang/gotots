// Gate helper utilities: running commands inside the staging repo,
// output splitting, file digests, and census package enumeration.
package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/tsoniclang/gotots/internal/census"
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

// detectMechanisms derives the custom runtime mechanisms present in
// generated output structurally, by their emitted markers — the spec's
// "verifier derives the requirement from emitted AST/helper ownership".
func detectMechanisms(files map[string]string) []string {
	markers := []struct{ mechanism, marker string }{
		{"slice-carrier", "gosl$.goSlice"},
		{"slice-carrier", "new GoSlice("},
		{"interface-box", "goif$.goIfaceBox("},
		{"interface-box", "goif$.GoIface"},
		{"pointer-cell", "$b: goabi$.GoCell"},
		{"pointer-cell", "gort$.goCellNew("},
		{"keyed-map", "gort$.goKeyed"},
		{"keyed-map", "GoKeyedMap"},
	}
	seen := map[string]bool{}
	for _, content := range files {
		for _, m := range markers {
			if !seen[m.mechanism] && strings.Contains(content, m.marker) {
				seen[m.mechanism] = true
			}
		}
	}
	out := make([]string, 0, len(seen))
	for mechanism := range seen {
		out = append(out, mechanism)
	}
	sort.Strings(out)
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
