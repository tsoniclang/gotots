package inventory

import "github.com/tsoniclang/gotots/internal/profile"

// attributeReachability walks the dependency graph from owned production,
// owned test/test-support, and hard-excluded/unselected scopes separately,
// marking every external package with the scopes that can actually reach
// it. Only external edges are traversed transitively; module-internal
// traversal stays within the seeding scope because a cross-scope module
// edge is a contradiction, not a dependency channel.
func attributeReachability(prof *profile.Profile, modulePackages map[string]*ModulePackage, externals map[string]*ExternalPackage) {
	markExternalClosure := func(seeds []string, mark func(*ExternalPackage)) {
		stack := append([]string{}, seeds...)
		visited := map[string]bool{}
		for len(stack) > 0 {
			current := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			if visited[current] {
				continue
			}
			visited[current] = true
			if entry := externals[current]; entry != nil {
				mark(entry)
				stack = append(stack, entry.Imports...)
			}
		}
	}

	seedExternals := func(imports []string, seeds *[]string) {
		for _, imported := range imports {
			if externals[imported] != nil {
				*seeds = append(*seeds, imported)
			}
		}
	}

	var productionSeeds, testSeeds, excludedSeeds []string
	for _, pkg := range modulePackages {
		switch profile.PackageClass(pkg.Class) {
		case profile.ClassOwned:
			seedExternals(pkg.Imports, &productionSeeds)
			seedExternals(pkg.TestImports, &testSeeds)
			seedExternals(pkg.XTestImports, &testSeeds)
		case profile.ClassTestOnly:
			seedExternals(pkg.Imports, &testSeeds)
			seedExternals(pkg.TestImports, &testSeeds)
			seedExternals(pkg.XTestImports, &testSeeds)
		case profile.ClassHardExcluded, profile.ClassUnselected:
			seedExternals(pkg.Imports, &excludedSeeds)
			seedExternals(pkg.TestImports, &excludedSeeds)
			seedExternals(pkg.XTestImports, &excludedSeeds)
		}
	}
	markExternalClosure(productionSeeds, func(e *ExternalPackage) { e.ReachableFromProduction = true })
	markExternalClosure(testSeeds, func(e *ExternalPackage) { e.ReachableFromTest = true })
	markExternalClosure(excludedSeeds, func(e *ExternalPackage) { e.ReachableFromExcluded = true })
}
