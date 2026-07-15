// Package inventory implements census pass 1: a complete, identity-bearing
// module and file inventory produced by the go list driver without loading
// any syntax.
//
// Unlike a reachability walk from owned roots, this pass enumerates every
// package in the pinned module — including hard-excluded and unselected
// packages that nothing owned imports — so the partition provably covers the
// whole module. It also enumerates the external dependency closure with the
// toolchain's own standard-library/module evidence (go list Standard and
// Module fields), never path-shape guessing.
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
	Replace *listModule
}

type listError struct {
	Err string
}

// ModulePackage is one package of the pinned module with its complete file
// inventory. Paths are module-relative.
type ModulePackage struct {
	ImportPath string `json:"importPath"`
	Class      string `json:"class"`
	Category   string `json:"category,omitempty"`
	// Selected Go files under the active build profile.
	GoFiles     []string `json:"goFiles,omitempty"`
	TestGoFiles []string `json:"testGoFiles,omitempty"`
	// Files present in the package directory but not selected by the active
	// build profile (other GOOS/GOARCH, build tags, cgo-disabled files).
	IgnoredGoFiles []string `json:"ignoredGoFiles,omitempty"`
	// Non-Go files the build would consider (assembly, C); any entry in an
	// owned package needs an explicit disposition before translation.
	OtherFiles []string `json:"otherFiles,omitempty"`
	// Files referenced by //go:embed directives.
	EmbedFiles []string `json:"embedFiles,omitempty"`
	Imports    []string `json:"imports,omitempty"`
	// TestImports unions in-package and external test-file imports.
	TestImports []string `json:"testImports,omitempty"`
	LoadError   string   `json:"loadError,omitempty"`
}

// ExternalPackage is one package outside the pinned module, classified with
// toolchain evidence.
type ExternalPackage struct {
	ImportPath    string `json:"importPath"`
	Standard      bool   `json:"standard"`
	ModulePath    string `json:"modulePath,omitempty"`
	ModuleVersion string `json:"moduleVersion,omitempty"`
	// Replaced records a module replace directive target when present.
	Replaced  string `json:"replaced,omitempty"`
	LoadError string `json:"loadError,omitempty"`
}

// Inventory is the complete pass-1 result.
type Inventory struct {
	Module   []ModulePackage   `json:"module"`
	External []ExternalPackage `json:"external"`
}

// classOf returns an ExternalPackage's profile class using loader evidence.
func (e *ExternalPackage) Class() string {
	if e.Standard {
		return "external-std"
	}
	return "external-module"
}

// Run executes pass 1 in sourceDir under the hermetic environment.
func Run(prof *profile.Profile, resolved *goenv.Resolved, env []string, sourceDir string) (*Inventory, error) {
	// -e tolerates packages that cannot build (hard-excluded roots may have
	// profile-specific breakage); errors are recorded per package and owned
	// packages with errors fail the census later.
	// -deps expands the transitive closure, providing Standard/Module
	// evidence for every external dependency.
	output, err := resolved.Run(sourceDir, env, "list", "-e", "-deps", "-test=false", "-json", "./...")
	if err != nil {
		return nil, fmt.Errorf("inventory pass: %w", err)
	}

	inventory := &Inventory{}
	seenModule := map[string]bool{}
	seenExternal := map[string]bool{}

	recordExternal := func(pkg *listPackage) {
		if seenExternal[pkg.ImportPath] {
			return
		}
		seenExternal[pkg.ImportPath] = true
		external := ExternalPackage{
			ImportPath: pkg.ImportPath,
			Standard:   pkg.Standard,
		}
		if pkg.Module != nil {
			external.ModulePath = pkg.Module.Path
			external.ModuleVersion = pkg.Module.Version
			if pkg.Module.Replace != nil {
				external.Replaced = pkg.Module.Replace.Path + "@" + pkg.Module.Replace.Version
			}
		}
		if pkg.Error != nil {
			external.LoadError = pkg.Error.Err
		}
		inventory.External = append(inventory.External, external)
	}

	decoder := json.NewDecoder(bytes.NewReader(output))
	for {
		var pkg listPackage
		if err := decoder.Decode(&pkg); err == io.EOF {
			break
		} else if err != nil {
			return nil, fmt.Errorf("parse go list output: %w", err)
		}
		if pkg.ForTest != "" || strings.HasSuffix(pkg.ImportPath, ".test") {
			continue // test variants carry no additional file identity here
		}

		class, category := prof.Classify(pkg.ImportPath)
		if class == profile.ClassExternal {
			recordExternal(&pkg)
			continue
		}

		if seenModule[pkg.ImportPath] {
			continue
		}
		seenModule[pkg.ImportPath] = true

		relativize := func(files []string) []string {
			if len(files) == 0 {
				return nil
			}
			out := make([]string, 0, len(files))
			for _, file := range files {
				absolute := filepath.Join(pkg.Dir, file)
				relative, err := filepath.Rel(sourceDir, absolute)
				if err != nil || strings.HasPrefix(relative, "..") {
					relative = absolute // fail visible rather than silently wrong
				}
				out = append(out, filepath.ToSlash(relative))
			}
			sort.Strings(out)
			return out
		}

		entry := ModulePackage{
			ImportPath:     pkg.ImportPath,
			Class:          string(class),
			Category:       category,
			GoFiles:        relativize(append(append([]string{}, pkg.GoFiles...), pkg.CgoFiles...)),
			TestGoFiles:    relativize(append(append([]string{}, pkg.TestGoFiles...), pkg.XTestGoFiles...)),
			IgnoredGoFiles: relativize(pkg.IgnoredGoFiles),
			OtherFiles:     relativize(append(append([]string{}, pkg.OtherFiles...), pkg.IgnoredOtherFiles...)),
			EmbedFiles:     relativize(pkg.EmbedFiles),
			Imports:        sortedUnique(pkg.Imports),
			TestImports:    sortedUnique(append(append([]string{}, pkg.TestImports...), pkg.XTestImports...)),
		}
		if pkg.Error != nil {
			entry.LoadError = pkg.Error.Err
		}
		inventory.Module = append(inventory.Module, entry)
	}

	// Test-only imports (e.g. gotest.tools) are outside the -deps closure of
	// -test=false. Resolve every import string that is still unclassified
	// with a targeted second invocation so external evidence is complete.
	var unresolved []string
	unresolvedSeen := map[string]bool{}
	for _, pkg := range inventory.Module {
		for _, imports := range [][]string{pkg.Imports, pkg.TestImports} {
			for _, imported := range imports {
				if imported == "C" || seenModule[imported] || seenExternal[imported] || unresolvedSeen[imported] {
					continue
				}
				unresolvedSeen[imported] = true
				unresolved = append(unresolved, imported)
			}
		}
	}
	if len(unresolved) > 0 {
		sort.Strings(unresolved)
		extra, err := resolved.Run(sourceDir, env, append([]string{"list", "-e", "-json"}, unresolved...)...)
		if err != nil {
			return nil, fmt.Errorf("inventory pass (test imports): %w", err)
		}
		decoder := json.NewDecoder(bytes.NewReader(extra))
		for {
			var pkg listPackage
			if err := decoder.Decode(&pkg); err == io.EOF {
				break
			} else if err != nil {
				return nil, fmt.Errorf("parse go list output: %w", err)
			}
			if class, _ := prof.Classify(pkg.ImportPath); class == profile.ClassExternal {
				recordExternal(&pkg)
			}
		}
	}

	sort.Slice(inventory.Module, func(i, j int) bool {
		return inventory.Module[i].ImportPath < inventory.Module[j].ImportPath
	})
	sort.Slice(inventory.External, func(i, j int) bool {
		return inventory.External[i].ImportPath < inventory.External[j].ImportPath
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
	return inventory, nil
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
