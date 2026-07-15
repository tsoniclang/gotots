// Package inventory implements census pass 1: a complete, identity-bearing
// module and file inventory.
//
// The tracked Git tree — not go list — defines the source universe: every
// tracked .go file receives exactly one classification, including files in
// nested tool modules and testdata fixtures that the go list driver omits
// by design. The go list driver then enriches that universe with
// build/package semantics under the selected build profile, and provides
// the toolchain's own standard-library/module evidence for every external
// dependency (Standard/Module fields), never path-shape guessing.
//
// Dependency evidence is scope-attributed: the whole-module enumeration and
// the product dependency closures are different graphs, and externals
// reachable only through hard-excluded or unselected source must not
// inflate product stub obligations.
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

// listPackage is the subset of `go list -json` output the inventory consumes.
type listPackage struct {
	ImportPath        string
	Dir               string
	Name              string
	Standard          bool
	ForTest           string
	Incomplete        bool
	Module            *listModule
	GoFiles           []string
	CgoFiles          []string
	OtherFiles        []string
	EmbedFiles        []string
	TestEmbedFiles    []string
	XTestEmbedFiles   []string
	IgnoredGoFiles    []string
	IgnoredOtherFiles []string
	TestGoFiles       []string
	XTestGoFiles      []string
	Imports           []string
	TestImports       []string
	XTestImports      []string
	DepsErrors        []*listError
	Error             *listError
}

type listModule struct {
	Path    string
	Version string
	Main    bool
	Dir     string
	Replace *listModule
}

type listError struct {
	Err string
}

// ModulePackage is one package of the pinned module with its complete file
// inventory. File classes are kept distinct — merging them would destroy
// the evidence needed for later dispositions. Paths are module-relative.
type ModulePackage struct {
	ImportPath string `json:"importPath"`
	Class      string `json:"class"`
	Category   string `json:"category,omitempty"`
	// GoFiles are the ordinary Go files selected by the active profile.
	GoFiles []string `json:"goFiles,omitempty"`
	// CgoFiles are selected files that import "C".
	CgoFiles []string `json:"cgoFiles,omitempty"`
	// TestGoFiles are selected in-package test files.
	TestGoFiles []string `json:"testGoFiles,omitempty"`
	// XTestGoFiles are selected black-box test files (package p_test).
	XTestGoFiles []string `json:"xtestGoFiles,omitempty"`
	// IgnoredGoFiles are present but not selected by the active profile.
	IgnoredGoFiles []string `json:"ignoredGoFiles,omitempty"`
	// OtherFiles are selected non-Go build inputs (assembly, C headers).
	OtherFiles []string `json:"otherFiles,omitempty"`
	// IgnoredOtherFiles are non-Go build inputs excluded by the profile.
	IgnoredOtherFiles []string `json:"ignoredOtherFiles,omitempty"`
	// EmbedFiles are files referenced by //go:embed directives, including
	// from test files.
	EmbedFiles  []string `json:"embedFiles,omitempty"`
	Imports     []string `json:"imports,omitempty"`
	TestImports []string `json:"testImports,omitempty"`
	LoadError   string   `json:"loadError,omitempty"`
}

// ExternalPackage is one package outside the pinned module, classified with
// toolchain evidence and attributed to the scopes that can reach it.
type ExternalPackage struct {
	ImportPath    string `json:"importPath"`
	Standard      bool   `json:"standard"`
	ModulePath    string `json:"modulePath,omitempty"`
	ModuleVersion string `json:"moduleVersion,omitempty"`
	// ReachableFromProduction: importable through owned production source.
	ReachableFromProduction bool `json:"reachableFromProduction"`
	// ReachableFromTest: importable through owned tests or test support.
	ReachableFromTest bool `json:"reachableFromTest"`
	// ExcludedOrUnselectedOnly: reached only through hard-excluded or
	// unselected source; never a product stub obligation.
	ExcludedOrUnselectedOnly bool     `json:"excludedOrUnselectedOnly"`
	imports                  []string // for closure computation, not serialized
}

// TrackedGoFile is one tracked .go file outside every go list package
// (nested modules, testdata fixtures, toolchain-excluded directories).
type TrackedGoFile struct {
	Path  string `json:"path"`
	Class string `json:"class"` // tooling | testdata | toolchain-excluded
}

// Universe reconciles the tracked Git tree with the package inventory.
type Universe struct {
	TrackedGoFiles   int             `json:"trackedGoFiles"`
	InPackages       int             `json:"inPackages"`
	OutsidePackages  []TrackedGoFile `json:"outsidePackages"`
	SubmoduleGitlink []string        `json:"submoduleGitlinks,omitempty"`
}

// Inventory is the complete pass-1 result.
type Inventory struct {
	Universe Universe          `json:"universe"`
	Module   []ModulePackage   `json:"module"`
	External []ExternalPackage `json:"external"`
}

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

	recordExternal := func(pkg *listPackage) error {
		entry := ExternalPackage{
			ImportPath: pkg.ImportPath,
			Standard:   pkg.Standard,
			imports:    sortedUnique(pkg.Imports),
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
				entry.ModulePath = pkg.Module.Replace.Path
				entry.ModuleVersion = pkg.Module.Replace.Version
			}
		}
		if pkg.Error != nil {
			return fmt.Errorf("external package %s has no usable evidence: %s", pkg.ImportPath, pkg.Error.Err)
		}
		if previous, ok := externals[pkg.ImportPath]; ok {
			// Conflicting evidence for the same import path must be
			// reconciled, never first-wins.
			if previous.Standard != entry.Standard || previous.ModulePath != entry.ModulePath || previous.ModuleVersion != entry.ModuleVersion {
				return fmt.Errorf("conflicting evidence for external package %s", pkg.ImportPath)
			}
			return nil
		}
		externals[pkg.ImportPath] = &entry
		return nil
	}

	decode := func(data []byte, allowModule bool) error {
		decoder := json.NewDecoder(bytes.NewReader(data))
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
				if err := recordExternal(&pkg); err != nil {
					return err
				}
				continue
			}
			if !allowModule {
				continue
			}
			if _, ok := modulePackages[pkg.ImportPath]; ok {
				continue
			}

			relativize := func(files []string) ([]string, error) {
				if len(files) == 0 {
					return nil, nil
				}
				out := make([]string, 0, len(files))
				for _, file := range files {
					absolute := filepath.Join(pkg.Dir, file)
					relative, err := filepath.Rel(sourceDir, absolute)
					if err != nil || strings.HasPrefix(relative, "..") {
						return nil, fmt.Errorf("package %s file %s is outside the pinned source checkout", pkg.ImportPath, absolute)
					}
					out = append(out, filepath.ToSlash(relative))
				}
				sort.Strings(out)
				return out, nil
			}

			entry := ModulePackage{
				ImportPath:  pkg.ImportPath,
				Class:       string(class),
				Category:    category,
				Imports:     sortedUnique(pkg.Imports),
				TestImports: sortedUnique(append(append([]string{}, pkg.TestImports...), pkg.XTestImports...)),
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
				{&entry.OtherFiles, pkg.OtherFiles},
				{&entry.IgnoredOtherFiles, pkg.IgnoredOtherFiles},
				{&entry.EmbedFiles, sortedUnique(append(append(append([]string{}, pkg.EmbedFiles...), pkg.TestEmbedFiles...), pkg.XTestEmbedFiles...))},
			}
			for _, set := range fileSets {
				files, err := relativize(set.source)
				if err != nil {
					return err
				}
				*set.target = files
			}
			if pkg.Error != nil {
				entry.LoadError = pkg.Error.Err
			}
			modulePackages[pkg.ImportPath] = &entry
		}
	}

	if err := decode(output, true); err != nil {
		return nil, err
	}

	// Test-only imports (e.g. gotest.tools) are outside the production
	// -deps closure. Resolve every import string that is still
	// unclassified, with -deps so transitive test dependency evidence is
	// complete by contract rather than by accident.
	var unresolved []string
	unresolvedSeen := map[string]bool{}
	for _, pkg := range modulePackages {
		for _, imports := range [][]string{pkg.Imports, pkg.TestImports} {
			for _, imported := range imports {
				if imported == "C" || unresolvedSeen[imported] || externals[imported] != nil {
					continue
				}
				if _, isModule := modulePackages[imported]; isModule {
					continue
				}
				if class, _ := prof.Classify(imported); class != profile.ClassExternal {
					continue // module package outside ./... cannot happen; classified path stays module-side
				}
				unresolvedSeen[imported] = true
				unresolved = append(unresolved, imported)
			}
		}
	}
	if len(unresolved) > 0 {
		sort.Strings(unresolved)
		extra, err := resolved.Run(sourceDir, env, append([]string{"list", "-e", "-deps", "-json"}, unresolved...)...)
		if err != nil {
			return nil, fmt.Errorf("inventory pass (test imports): %w", err)
		}
		if err := decode(extra, false); err != nil {
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
		return nil, fmt.Errorf("inventory fails closed on %d owned-package errors:\n%s",
			len(blocking), strings.Join(blocking, "\n"))
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

// attributeReachability walks the dependency graph from owned production
// and owned test/test-support scopes separately, marking every external
// package with the scopes that can actually reach it. Hard-excluded and
// unselected packages are never traversed: their dependencies must not
// inflate product obligations.
func attributeReachability(prof *profile.Profile, modulePackages map[string]*ModulePackage, externals map[string]*ExternalPackage) {
	markExternalClosure := func(start []string, mark func(*ExternalPackage)) {
		stack := append([]string{}, start...)
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
				stack = append(stack, entry.imports...)
			}
		}
	}

	var productionSeeds, testSeeds []string
	for _, pkg := range modulePackages {
		switch profile.PackageClass(pkg.Class) {
		case profile.ClassOwned:
			for _, imported := range pkg.Imports {
				if externals[imported] != nil {
					productionSeeds = append(productionSeeds, imported)
				}
			}
			for _, imported := range pkg.TestImports {
				if externals[imported] != nil {
					testSeeds = append(testSeeds, imported)
				}
			}
		case profile.ClassTestOnly:
			for _, imports := range [][]string{pkg.Imports, pkg.TestImports} {
				for _, imported := range imports {
					if externals[imported] != nil {
						testSeeds = append(testSeeds, imported)
					}
				}
			}
		}
	}
	markExternalClosure(productionSeeds, func(e *ExternalPackage) { e.ReachableFromProduction = true })
	markExternalClosure(testSeeds, func(e *ExternalPackage) { e.ReachableFromTest = true })
}

// reconcileUniverse proves that every tracked .go file has exactly one
// disposition: either it belongs to an inventoried package, or it is
// classified by an explicit rule (profile tooling roots, the toolchain's
// documented testdata and underscore/dot directory exclusions). Anything
// else fails the census.
func reconcileUniverse(prof *profile.Profile, tree *pinning.Tree, modulePackages map[string]*ModulePackage, universe *Universe) error {
	inPackage := map[string]bool{}
	for _, pkg := range modulePackages {
		for _, set := range [][]string{
			pkg.GoFiles, pkg.CgoFiles, pkg.TestGoFiles, pkg.XTestGoFiles, pkg.IgnoredGoFiles,
		} {
			for _, file := range set {
				inPackage[file] = true
			}
		}
	}

	trackedGo := tree.GoFiles()
	universe.TrackedGoFiles = len(trackedGo)

	underAny := func(path string, roots []string) bool {
		for _, root := range roots {
			if path == root || strings.HasPrefix(path, root+"/") {
				return true
			}
		}
		return false
	}
	var unclassified []string
	for _, path := range trackedGo {
		if inPackage[path] {
			universe.InPackages++
			continue
		}
		var class string
		switch {
		case underAny(path, prof.ToolingRoots):
			class = "tooling"
		case hasSegment(path, "testdata"):
			// The go toolchain's documented rule: testdata directories are
			// never part of any package.
			class = "testdata"
		case hasExcludedSegment(path):
			// The go toolchain's documented rule: directories beginning
			// with "_" or "." are ignored by package discovery.
			class = "toolchain-excluded"
		default:
			unclassified = append(unclassified, path)
			continue
		}
		universe.OutsidePackages = append(universe.OutsidePackages, TrackedGoFile{Path: path, Class: class})
	}
	if len(unclassified) > 0 {
		return fmt.Errorf("universe reconciliation fails closed: %d tracked .go files have no disposition:\n%s",
			len(unclassified), strings.Join(unclassified, "\n"))
	}

	for path := range tree.Submodules {
		universe.SubmoduleGitlink = append(universe.SubmoduleGitlink, path)
	}
	sort.Strings(universe.SubmoduleGitlink)
	return nil
}

func hasSegment(path, segment string) bool {
	for _, part := range strings.Split(path, "/") {
		if part == segment {
			return true
		}
	}
	return false
}

func hasExcludedSegment(path string) bool {
	// The go toolchain ignores directories and files whose names begin
	// with "_" or "." during package discovery.
	for _, part := range strings.Split(path, "/") {
		if strings.HasPrefix(part, "_") || strings.HasPrefix(part, ".") {
			return true
		}
	}
	return false
}

// ExternalIndex returns import path -> evidence for pass-2 classification.
func (inv *Inventory) ExternalIndex() map[string]*ExternalPackage {
	index := make(map[string]*ExternalPackage, len(inv.External))
	for i := range inv.External {
		index[inv.External[i].ImportPath] = &inv.External[i]
	}
	return index
}

func sortedUnique(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	sort.Strings(values)
	out := values[:0]
	previous := ""
	for i, v := range values {
		if i == 0 || v != previous {
			out = append(out, v)
		}
		previous = v
	}
	return out
}
