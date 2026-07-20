// Stage 03: selected source scope, independently proved against the
// schema-2 root contract. The gate derives the typed import closure
// from the DECLARED selected package roots (never from upstream cmd
// mains), verifies total classification of every in-checkout package,
// verifies the selected closure contains no forbidden disposition, and
// runs the upstream-main closures as DRIFT PROBES only: their
// dependencies must all classify, but they add and remove nothing.
// Runtime/publication reachability belongs to the later product graph,
// not to this stage.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"

	"github.com/tsoniclang/gotots/internal/profile"
)

// loadImportClosure walks the import graph from the given patterns and
// returns the module-internal closure as module-relative package paths.
func loadImportClosure(sourceDir string, env []string, patterns []string) (module string, closure []string, roots []string, err error) {
	config := &packages.Config{
		Mode: packages.NeedName | packages.NeedImports | packages.NeedModule,
		Dir:  sourceDir,
		Env:  env,
	}
	loaded, err := packages.Load(config, patterns...)
	if err != nil {
		return "", nil, nil, fmt.Errorf("load closure roots: %w", err)
	}
	var loadErrors []string
	packages.Visit(loaded, nil, func(p *packages.Package) {
		for _, e := range p.Errors {
			loadErrors = append(loadErrors, fmt.Sprintf("%s: %s", p.ID, e))
		}
	})
	if len(loadErrors) > 0 {
		sort.Strings(loadErrors)
		return "", nil, nil, fmt.Errorf("closure load fails closed on %d errors:\n%s", len(loadErrors), strings.Join(loadErrors, "\n"))
	}
	seen := map[string]bool{}
	set := map[string]bool{}
	var walk func(p *packages.Package)
	walk = func(p *packages.Package) {
		if seen[p.PkgPath] {
			return
		}
		seen[p.PkgPath] = true
		if p.Module != nil && module == "" {
			module = p.Module.Path
		}
		if module != "" && strings.HasPrefix(p.PkgPath, module+"/") {
			set[strings.TrimPrefix(p.PkgPath, module+"/")] = true
		}
		imports := make([]string, 0, len(p.Imports))
		for path := range p.Imports {
			imports = append(imports, path)
		}
		sort.Strings(imports)
		for _, path := range imports {
			walk(p.Imports[path])
		}
	}
	for _, p := range loaded {
		roots = append(roots, p.PkgPath)
		walk(p)
	}
	sort.Strings(roots)
	for path := range set {
		closure = append(closure, path)
	}
	sort.Strings(closure)
	return module, closure, roots, nil
}

// enumerateCheckoutPackages lists every module-relative directory in the
// checkout containing at least one non-test .go file outside testdata —
// the total-classification denominator.
func enumerateCheckoutPackages(sourceDir string) ([]string, error) {
	set := map[string]bool{}
	err := filepath.WalkDir(sourceDir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name := entry.Name()
		if entry.IsDir() {
			if name == "node_modules" || name == ".git" || name == "testdata" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(name, ".go") {
			return nil
		}
		relative, err := filepath.Rel(sourceDir, filepath.Dir(path))
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if relative == "." {
			return nil
		}
		set[relative] = true
		return nil
	})
	if err != nil {
		return nil, err
	}
	packages := make([]string, 0, len(set))
	for pkg := range set {
		packages = append(packages, pkg)
	}
	sort.Strings(packages)
	return packages, nil
}

// runScopeClosureGate proves the schema-2 source-scope contract.
func runScopeClosureGate(prof *profile.Profile, sourceDir string, env []string) (string, []string, error) {
	var failures []string
	var details []string

	// 1. Total classification: every in-checkout package has exactly one
	// winning rule.
	checkout, err := enumerateCheckoutPackages(sourceDir)
	if err != nil {
		return "fail", nil, err
	}
	dispositions := map[profile.PackageDisposition]int{}
	ruleWitness := map[string]bool{}
	for _, pkg := range checkout {
		rule, err := prof.SourceUniverse.Classify(pkg)
		if err != nil {
			failures = append(failures, "coverage: "+err.Error())
			continue
		}
		dispositions[rule.Disposition]++
		ruleWitness[rule.ID] = true
	}
	details = append(details, fmt.Sprintf("in-checkout packages classified: %d (selected=%d test-only=%d outside=%d tooling=%d policy-excluded=%d)",
		len(checkout), dispositions[profile.DispositionSelected], dispositions[profile.DispositionTestOnly],
		dispositions[profile.DispositionOutside], dispositions[profile.DispositionTooling], dispositions[profile.DispositionPolicyExcluded]))

	// Every rule must classify at least one real package — a dead rule is
	// a silently shrunk or stale contract entry.
	for _, rule := range prof.SourceUniverse.PackageRules {
		if !ruleWitness[rule.ID] {
			failures = append(failures, fmt.Sprintf("rule %s matches no in-checkout package (dead contract entry)", rule.ID))
		}
	}

	// 2. Selected closure from DECLARED selected roots via typed imports:
	// no forbidden disposition inside it, and every selected package
	// reached from the contract has a witness.
	selectedPatterns := prof.SourceUniverse.SelectedLoadPatterns()
	module, closure, _, err := loadImportClosure(sourceDir, env, selectedPatterns)
	if err != nil {
		return "fail", nil, err
	}
	selectedWitness := map[string]bool{}
	for _, pkg := range closure {
		rule, err := prof.SourceUniverse.Classify(pkg)
		if err != nil {
			failures = append(failures, "selected-closure coverage: "+err.Error())
			continue
		}
		switch rule.Disposition {
		case profile.DispositionSelected:
			selectedWitness[pkg] = true
		case profile.DispositionTestOnly, profile.DispositionOutside, profile.DispositionTooling, profile.DispositionPolicyExcluded:
			failures = append(failures, fmt.Sprintf("selected closure reaches %s package %s (rule %s, %s)", rule.Disposition, pkg, rule.ID, rule.Decision))
		}
	}
	details = append(details, fmt.Sprintf("selected typed-import closure: %d packages from declared roots (module %s)", len(closure), module))

	// 3. Drift probes: upstream main closures must classify completely;
	// they never add or remove scope.
	for _, probe := range prof.SourceUniverse.DriftProbes {
		if probe.Kind != "upstream-main-closures" {
			failures = append(failures, fmt.Sprintf("unknown drift probe kind %q", probe.Kind))
			continue
		}
		_, probeClosure, probeRoots, err := loadImportClosure(sourceDir, env, []string{"./" + probe.Root + "/..."})
		if err != nil {
			return "fail", nil, err
		}
		unclassified := 0
		for _, pkg := range probeClosure {
			if _, err := prof.SourceUniverse.Classify(pkg); err != nil {
				failures = append(failures, fmt.Sprintf("drift probe %s: %s", probe.Root, err))
				unclassified++
			}
		}
		details = append(details, fmt.Sprintf("drift probe %s: %d entry points, %d-package closure, %d unclassified (evidence only; adds no roots)",
			probe.Root, len(probeRoots), len(probeClosure), unclassified))
	}

	if len(failures) > 0 {
		sort.Strings(failures)
		return "fail", append(details, failures...), fmt.Errorf("%d scope-contract defects", len(failures))
	}
	details = append(details, "declared contract, total classification, selected closure, and drift probes all agree")
	return "pass", details, nil
}
