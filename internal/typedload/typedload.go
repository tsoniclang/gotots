// Package typedload performs the hermetic typed load of owned and
// test-only roots shared by the census and the translator: syntax and type
// information for root packages only, dependencies consumed as compiled
// export data, and package roles resolved from loader evidence.
package typedload

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"

	"github.com/tsoniclang/gotots/internal/profile"
)

// Load performs the typed pass over owned and test-only roots.
//
// NeedDeps is deliberately absent: with it, x/tools parses and type-checks
// every transitive dependency from source, violating the external-body
// boundary. Without it, syntax and type information are produced for the
// root packages only; dependencies are consumed as compiled export data.
func Load(prof *profile.Profile, env []string, sourceDir string) ([]*packages.Package, error) {
	patterns := append(prof.SourceUniverse.SelectedLoadPatterns(), prof.SourceUniverse.TestOnlyLoadPatterns()...)
	sort.Strings(patterns)

	config := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles |
			packages.NeedImports | packages.NeedModule | packages.NeedForTest |
			packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo,
		Dir:   sourceDir,
		Env:   env,
		Tests: true,
	}

	loaded, err := packages.Load(config, patterns...)
	if err != nil {
		return nil, fmt.Errorf("load packages: %w", err)
	}

	var loadErrors []string
	packages.Visit(loaded, nil, func(p *packages.Package) {
		for _, e := range p.Errors {
			loadErrors = append(loadErrors, fmt.Sprintf("%s: %s", p.ID, e))
		}
	})
	if len(loadErrors) > 0 {
		sort.Strings(loadErrors)
		return nil, fmt.Errorf("typed load fails closed on %d load/type errors:\n%s",
			len(loadErrors), strings.Join(loadErrors, "\n"))
	}
	return loaded, nil
}

// Role distinguishes the go/packages test variants using the loader's
// ForTest field plus explicit file-location evidence for the synthesized
// test binary — never package IDs or path-suffix parsing.
type Role int

const (
	RoleProduction Role = iota
	RoleInPackageTest
	RoleExternalTest
	RoleSynthesizedTestMain
)

// RoleOf classifies one loaded package variant.
func RoleOf(p *packages.Package, sourceDir string) Role {
	if p.ForTest != "" {
		if p.PkgPath == p.ForTest {
			return RoleInPackageTest
		}
		return RoleExternalTest
	}
	if p.Name == "main" && len(p.GoFiles) > 0 {
		// The toolchain-synthesized test binary consists entirely of
		// generated files (its _testmain.go) outside the pinned checkout.
		// A genuine owned main package has in-tree files and remains
		// production source.
		allOutside := true
		for _, file := range p.GoFiles {
			if relative, err := filepath.Rel(sourceDir, file); err == nil && !strings.HasPrefix(relative, "..") {
				allOutside = false
				break
			}
		}
		if allOutside {
			return RoleSynthesizedTestMain
		}
	}
	return RoleProduction
}

// OwnerPath is the package whose profile scope owns this variant: the
// package under test for test variants, the package itself otherwise.
func OwnerPath(p *packages.Package) string {
	if p.ForTest != "" {
		return p.ForTest
	}
	return p.PkgPath
}
