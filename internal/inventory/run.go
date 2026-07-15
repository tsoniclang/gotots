package inventory

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tsoniclang/gotots/internal/goenv"
	"github.com/tsoniclang/gotots/internal/pinning"
	"github.com/tsoniclang/gotots/internal/profile"
)

// Run executes pass 1 in sourceDir under the hermetic environment against
// the verified tracked tree.
func Run(prof *profile.Profile, resolved *goenv.Resolved, env []string, sourceDir string, tree *pinning.Tree) (*Inventory, error) {
	// -e tolerates packages that cannot build (hard-excluded roots may have
	// profile-specific breakage); errors are recorded per package and owned
	// packages with errors fail the census.
	// -deps expands the transitive closure, providing Standard/Module
	// evidence for every external dependency of selected production code.
	output, err := resolved.Run(sourceDir, env, "list", "-e", "-deps", "-json", "./...")
	if err != nil {
		return nil, fmt.Errorf("inventory pass: %w", err)
	}

	inventory := &Inventory{}
	modulePackages := map[string]*ModulePackage{}
	externals := map[string]*ExternalPackage{}

	if err := decodeList(prof, sourceDir, output, true, modulePackages, externals); err != nil {
		return nil, err
	}

	// Test-only imports (e.g. gotest.tools) are outside the production
	// -deps closure. Resolve every import string that is still
	// unclassified, with -deps so transitive test dependency evidence is
	// complete by contract rather than by accident.
	unresolved := unresolvedImports(prof, modulePackages, externals)
	if len(unresolved) > 0 {
		extra, err := resolved.Run(sourceDir, env, append([]string{"list", "-e", "-deps", "-json"}, unresolved...)...)
		if err != nil {
			return nil, fmt.Errorf("inventory pass (test imports): %w", err)
		}
		if err := decodeList(prof, sourceDir, extra, false, modulePackages, externals); err != nil {
			return nil, err
		}
	}

	for _, pkg := range modulePackages {
		inventory.Module = append(inventory.Module, *pkg)
	}
	sort.Slice(inventory.Module, func(i, j int) bool {
		return inventory.Module[i].ImportPath < inventory.Module[j].ImportPath
	})

	// Owned or test-only packages must load cleanly in pass 1; excluded and
	// unselected packages may carry errors (they are outside the product).
	var blocking []string
	for _, pkg := range inventory.Module {
		if pkg.LoadError == "" {
			continue
		}
		if pkg.Class == string(profile.ClassOwned) || pkg.Class == string(profile.ClassTestOnly) {
			blocking = append(blocking, pkg.ImportPath+": "+pkg.LoadError)
		}
	}
	if len(blocking) > 0 {
		sort.Strings(blocking)
		return nil, fmt.Errorf("inventory fails closed on %d owned-package errors:\n%s",
			len(blocking), strings.Join(blocking, "\n"))
	}

	if err := attestBuildInputs(tree, inventory.Module, sourceDir); err != nil {
		return nil, err
	}

	attributeReachability(prof, modulePackages, externals)
	for _, entry := range externals {
		entry.ExcludedOrUnselectedOnly = !entry.ReachableFromProduction && !entry.ReachableFromTest
		inventory.External = append(inventory.External, *entry)
	}
	sort.Slice(inventory.External, func(i, j int) bool {
		return inventory.External[i].ImportPath < inventory.External[j].ImportPath
	})

	if err := reconcileUniverse(prof, tree, modulePackages, &inventory.Universe); err != nil {
		return nil, err
	}
	return inventory, nil
}

// decodeList consumes one go list -json stream. Module packages are only
// recorded when allowModule is set (the targeted second resolution may
// never introduce new module packages).
func decodeList(prof *profile.Profile, sourceDir string, output []byte, allowModule bool, modulePackages map[string]*ModulePackage, externals map[string]*ExternalPackage) error {
	decoder := json.NewDecoder(bytes.NewReader(output))
	for {
		var pkg listPackage
		if err := decoder.Decode(&pkg); err == io.EOF {
			return nil
		} else if err != nil {
			return fmt.Errorf("parse go list output: %w", err)
		}
		// Test variants cannot occur: -test is never passed in pass 1.
		if pkg.ForTest != "" {
			return fmt.Errorf("unexpected test variant %s in pass 1", pkg.ImportPath)
		}

		class, category := prof.Classify(pkg.ImportPath)
		if class == profile.ClassExternal {
			if err := recordExternal(externals, &pkg); err != nil {
				return err
			}
			continue
		}
		if !allowModule {
			return fmt.Errorf("module package %s appeared during external test-import resolution", pkg.ImportPath)
		}
		if _, ok := modulePackages[pkg.ImportPath]; ok {
			continue
		}
		entry, err := moduleEntry(&pkg, class, category, sourceDir)
		if err != nil {
			return err
		}
		modulePackages[pkg.ImportPath] = entry
	}
}

func moduleEntry(pkg *listPackage, class profile.PackageClass, category, sourceDir string) (*ModulePackage, error) {
	relativize := func(files []string) ([]string, error) {
		if len(files) == 0 {
			return nil, nil
		}
		out := make([]string, 0, len(files))
		for _, file := range files {
			absolute := file
			if !filepath.IsAbs(absolute) {
				absolute = filepath.Join(pkg.Dir, file)
			}
			relative, err := filepath.Rel(sourceDir, absolute)
			if err != nil || strings.HasPrefix(relative, "..") {
				return nil, fmt.Errorf("package %s file %s is outside the pinned source checkout", pkg.ImportPath, absolute)
			}
			out = append(out, filepath.ToSlash(relative))
		}
		sort.Strings(out)
		return out, nil
	}

	entry := &ModulePackage{
		ImportPath:   pkg.ImportPath,
		Class:        string(class),
		Category:     category,
		Imports:      sortedUnique(pkg.Imports),
		TestImports:  sortedUnique(pkg.TestImports),
		XTestImports: sortedUnique(pkg.XTestImports),
	}
	fileSets := []struct {
		target *[]string
		source []string
	}{
		{&entry.GoFiles, pkg.GoFiles},
		{&entry.CgoFiles, pkg.CgoFiles},
		{&entry.TestGoFiles, pkg.TestGoFiles},
		{&entry.XTestGoFiles, pkg.XTestGoFiles},
		{&entry.IgnoredGoFiles, pkg.IgnoredGoFiles},
		{&entry.OtherFiles, pkg.otherFiles()},
		{&entry.IgnoredOtherFiles, pkg.IgnoredOtherFiles},
		{&entry.EmbedFiles, pkg.EmbedFiles},
		{&entry.TestEmbedFiles, pkg.TestEmbedFiles},
		{&entry.XTestEmbedFiles, pkg.XTestEmbedFiles},
	}
	for _, set := range fileSets {
		files, err := relativize(set.source)
		if err != nil {
			return nil, err
		}
		*set.target = files
	}
	if pkg.Error != nil {
		entry.LoadError = pkg.Error.Err
	}
	return entry, nil
}

// recordExternal captures toolchain evidence for one external package.
// Conflicting evidence for the same import path is reconciled, never
// first-wins; unpinned local replacements and evidence-free packages fail.
func recordExternal(externals map[string]*ExternalPackage, pkg *listPackage) error {
	if pkg.Error != nil {
		return fmt.Errorf("external package %s has no usable evidence: %s", pkg.ImportPath, pkg.Error.Err)
	}
	entry := ExternalPackage{
		ImportPath: pkg.ImportPath,
		Standard:   pkg.Standard,
		Imports:    sortedUnique(pkg.Imports),
	}
	if pkg.Module != nil {
		entry.ModulePath = pkg.Module.Path
		entry.ModuleVersion = pkg.Module.Version
		if pkg.Module.Replace != nil {
			// A replacement without a version is a local filesystem
			// directory: unpinned source that would leak machine paths.
			if pkg.Module.Replace.Version == "" {
				return fmt.Errorf("external package %s resolves through an unpinned local replacement %q; pin the module instead",
					pkg.ImportPath, pkg.Module.Replace.Path)
			}
			entry.ReplacementPath = pkg.Module.Replace.Path
			entry.ReplacementVersion = pkg.Module.Replace.Version
		}
	}
	if previous, ok := externals[pkg.ImportPath]; ok {
		if previous.Standard != entry.Standard ||
			previous.ModulePath != entry.ModulePath || previous.ModuleVersion != entry.ModuleVersion ||
			previous.ReplacementPath != entry.ReplacementPath || previous.ReplacementVersion != entry.ReplacementVersion ||
			!equalStrings(previous.Imports, entry.Imports) {
			return fmt.Errorf("conflicting evidence for external package %s", pkg.ImportPath)
		}
		return nil
	}
	externals[pkg.ImportPath] = &entry
	return nil
}

// unresolvedImports collects every import string mentioned by module
// packages that has no classification yet.
func unresolvedImports(prof *profile.Profile, modulePackages map[string]*ModulePackage, externals map[string]*ExternalPackage) []string {
	var unresolved []string
	seen := map[string]bool{}
	for _, pkg := range modulePackages {
		for _, imports := range [][]string{pkg.Imports, pkg.TestImports, pkg.XTestImports} {
			for _, imported := range imports {
				if imported == "C" || seen[imported] || externals[imported] != nil {
					continue
				}
				if _, isModule := modulePackages[imported]; isModule {
					continue
				}
				if class, _ := prof.Classify(imported); class != profile.ClassExternal {
					continue
				}
				seen[imported] = true
				unresolved = append(unresolved, imported)
			}
		}
	}
	sort.Strings(unresolved)
	return unresolved
}
