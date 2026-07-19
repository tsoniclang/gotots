// Stage 03: the selected-scope dependency closure, INDEPENDENTLY derived.
// The owned closure is recomputed from the product's entry points (every
// main package under cmd/ of the pinned module — discovered, never
// hand-listed) by walking the import graph; the profile's hand-maintained
// ownedRoots allowlist must coincide EXACTLY with the derived closure
// minus the declared product-policy exclusions (outside-universe and
// tooling roots). A shrink (a derived package missing from the
// allowlist), an unreachable allowlist entry, an undeclared scope, or a
// test-only root inside the product closure each fail closed.
package main

import (
	"fmt"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"

	"github.com/tsoniclang/gotots/internal/profile"
)

// deriveScopeClosure walks the import graph from every cmd/ main package
// and returns the module-internal import closure as package paths
// relative to the module root.
func deriveScopeClosure(sourceDir string, env []string) (module string, closure []string, mains []string, err error) {
	config := &packages.Config{
		Mode: packages.NeedName | packages.NeedImports | packages.NeedModule,
		Dir:  sourceDir,
		Env:  env,
	}
	loaded, err := packages.Load(config, "./cmd/...")
	if err != nil {
		return "", nil, nil, fmt.Errorf("load product entry points: %w", err)
	}
	var loadErrors []string
	packages.Visit(loaded, nil, func(p *packages.Package) {
		for _, e := range p.Errors {
			loadErrors = append(loadErrors, fmt.Sprintf("%s: %s", p.ID, e))
		}
	})
	if len(loadErrors) > 0 {
		sort.Strings(loadErrors)
		return "", nil, nil, fmt.Errorf("entry-point load fails closed on %d errors:\n%s", len(loadErrors), strings.Join(loadErrors, "\n"))
	}
	seen := map[string]bool{}
	var walk func(p *packages.Package)
	set := map[string]bool{}
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
		if p.Name == "main" {
			mains = append(mains, p.PkgPath)
			walk(p)
		}
	}
	sort.Strings(mains)
	for path := range set {
		closure = append(closure, path)
	}
	sort.Strings(closure)
	return module, closure, mains, nil
}

// scopeRootOf resolves one derived package path to its longest declared
// root prefix across all four policy lists; "" when undeclared.
func scopeRootOf(pkg string, roots ...[]string) string {
	best := ""
	for _, list := range roots {
		for _, root := range list {
			if (pkg == root || strings.HasPrefix(pkg, root+"/")) && len(root) > len(best) {
				best = root
			}
		}
	}
	return best
}

// runScopeClosureGate compares the independently derived closure with the
// profile's declared scope policy.
func runScopeClosureGate(prof *profile.Profile, sourceDir string, env []string) (string, []string, error) {
	outsideRoots := make([]string, 0)
	for _, roots := range prof.OutsideUniverseRoots {
		outsideRoots = append(outsideRoots, roots...)
	}
	sort.Strings(outsideRoots)
	module, closure, mains, err := deriveScopeClosure(sourceDir, env)
	if err != nil {
		return "fail", nil, err
	}
	if len(mains) == 0 {
		return "fail", nil, fmt.Errorf("no product entry points (main packages under cmd/) discovered in %s", module)
	}
	details := []string{
		fmt.Sprintf("product entry points (discovered cmd/ mains): %s", strings.Join(mains, ", ")),
		fmt.Sprintf("derived module-internal closure: %d packages", len(closure)),
	}
	var failures []string
	reachedOwned := map[string]bool{}
	outsideCount, toolingCount := 0, 0
	for _, pkg := range closure {
		if !strings.HasPrefix(pkg, "internal/") {
			continue // cmd/ mains themselves and non-internal helpers
		}
		root := scopeRootOf(pkg, prof.OwnedRoots, prof.TestOnlyRoots, outsideRoots, prof.ToolingRoots)
		switch {
		case root == "":
			failures = append(failures, fmt.Sprintf("derived package %s matches NO declared scope root (undeclared scope)", pkg))
		case scopeRootOf(pkg, prof.OwnedRoots) == root:
			reachedOwned[root] = true
		case scopeRootOf(pkg, prof.TestOnlyRoots) == root:
			failures = append(failures, fmt.Sprintf("test-only root %s is inside the product closure (via %s)", root, pkg))
		case scopeRootOf(pkg, outsideRoots) == root:
			outsideCount++
		case scopeRootOf(pkg, prof.ToolingRoots) == root:
			toolingCount++
		}
	}
	for _, root := range prof.OwnedRoots {
		if !reachedOwned[root] {
			failures = append(failures, fmt.Sprintf("ownedRoots entry %s is UNREACHABLE from every product entry point (dead allowlist entry)", root))
		}
	}
	details = append(details,
		fmt.Sprintf("owned roots reached: %d/%d", len(reachedOwned), len(prof.OwnedRoots)),
		fmt.Sprintf("outside-universe packages excluded by declared policy: %d", outsideCount),
		fmt.Sprintf("tooling packages excluded by declared policy: %d", toolingCount))
	if len(failures) > 0 {
		sort.Strings(failures)
		return "fail", append(details, failures...), fmt.Errorf("%d scope-closure defects", len(failures))
	}
	details = append(details, "the hand-maintained allowlist coincides exactly with the independently derived product closure")
	return "pass", details, nil
}
