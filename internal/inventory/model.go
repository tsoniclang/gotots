// Package inventory implements census pass 1: a complete, identity-bearing
// module and file inventory.
//
// The tracked Git tree — not go list — defines the source universe: every
// tracked file receives exactly one classification, including files in
// nested tool modules and testdata fixtures that the go list driver omits
// by design. The go list driver then enriches that universe with
// build/package semantics under the selected build profile, and provides
// the toolchain's own standard-library/module evidence for every external
// dependency (Standard/Module fields), never path-shape guessing.
//
// Every build input any inventoried package names — selected or ignored,
// Go or not, embed payloads included — is attested byte-for-byte against
// the pinned commit tree, and dependency evidence is scope-attributed so
// externals reachable only through unselected source
// cannot inflate product obligations.
//
// File layout: model.go owns the data model; run.go drives the go list
// passes; attest.go reconciles build inputs against the pinned tree;
// universe.go classifies the tracked tree; reachability.go attributes the
// dependency graph by scope.
package inventory

import "sort"

// listPackage is the subset of `go list -json` output the inventory
// consumes. Non-Go build inputs are captured per toolchain category so no
// input class escapes attestation.
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
	CFiles            []string
	CXXFiles          []string
	MFiles            []string
	HFiles            []string
	FFiles            []string
	SFiles            []string
	SwigFiles         []string
	SwigCXXFiles      []string
	SysoFiles         []string
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

// otherFiles unions the selected non-Go build-input categories.
func (p *listPackage) otherFiles() []string {
	var out []string
	for _, set := range [][]string{p.CFiles, p.CXXFiles, p.MFiles, p.HFiles, p.FFiles, p.SFiles, p.SwigFiles, p.SwigCXXFiles, p.SysoFiles} {
		out = append(out, set...)
	}
	return out
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
	// OtherFiles are selected non-Go build inputs (assembly, C, headers).
	OtherFiles []string `json:"otherFiles,omitempty"`
	// IgnoredOtherFiles are non-Go build inputs excluded by the profile.
	IgnoredOtherFiles []string `json:"ignoredOtherFiles,omitempty"`
	// Embed payloads, kept distinct per referencing scope.
	EmbedFiles      []string `json:"embedFiles,omitempty"`
	TestEmbedFiles  []string `json:"testEmbedFiles,omitempty"`
	XTestEmbedFiles []string `json:"xtestEmbedFiles,omitempty"`
	// Dependency edges, kept distinct per importing scope.
	Imports      []string `json:"imports,omitempty"`
	TestImports  []string `json:"testImports,omitempty"`
	XTestImports []string `json:"xtestImports,omitempty"`
	LoadError    string   `json:"loadError,omitempty"`
}

// buildInputFiles returns every file this package names, across all
// classes, for tree attestation.
func (p *ModulePackage) buildInputFiles() []string {
	var out []string
	for _, set := range [][]string{
		p.GoFiles, p.CgoFiles, p.TestGoFiles, p.XTestGoFiles, p.IgnoredGoFiles,
		p.OtherFiles, p.IgnoredOtherFiles,
		p.EmbedFiles, p.TestEmbedFiles, p.XTestEmbedFiles,
	} {
		out = append(out, set...)
	}
	return out
}

// ExternalPackage is one package outside the pinned module, classified
// with toolchain evidence. Requested module identity is preserved
// separately from any replacement, and canonical dependency edges are
// serialized so closures remain independently auditable.
type ExternalPackage struct {
	ImportPath string `json:"importPath"`
	Standard   bool   `json:"standard"`
	// Requested module identity (what the source demanded).
	ModulePath    string `json:"modulePath,omitempty"`
	ModuleVersion string `json:"moduleVersion,omitempty"`
	// Effective identity when a replace directive applies.
	ReplacementPath    string `json:"replacementPath,omitempty"`
	ReplacementVersion string `json:"replacementVersion,omitempty"`
	// Imports are this package's own dependency edges.
	Imports []string `json:"imports,omitempty"`
	// Scope attribution: which parts of the partition can reach this
	// package through the dependency graph.
	ReachableFromProduction bool `json:"reachableFromProduction"`
	ReachableFromTest       bool `json:"reachableFromTest"`
	ReachableFromExcluded   bool `json:"reachableFromExcluded"`
	// ExcludedOrUnselectedOnly: never a product stub obligation.
	ExcludedOrUnselectedOnly bool `json:"excludedOrUnselectedOnly"`
}

// SubmoduleRecord pins one gitlink: the nested repository path and its
// immutable commit revision.
type SubmoduleRecord struct {
	Path     string `json:"path"`
	Revision string `json:"revision"`
}

// TrackedFile is one tracked source-bearing file outside every inventoried
// package, with its explicit disposition.
type TrackedFile struct {
	Path  string `json:"path"`
	Class string `json:"class"` // tooling | testdata | toolchain-excluded
}

// Universe reconciles the tracked Git tree with the package inventory:
// every tracked file has exactly one disposition.
type Universe struct {
	TrackedFiles int `json:"trackedFiles"`
	// OutsideUniverseFiles counts tracked files filtered before census
	// because they lie under declared outside-universe roots. They have
	// no per-file records: only these counts (per category) and the
	// aggregate class digest attest what the filter removed.
	OutsideUniverseFiles int            `json:"outsideUniverseFiles"`
	OutsideUniverse      map[string]int `json:"outsideUniverse,omitempty"`
	// InPackages counts tracked files claimed as build inputs (any class,
	// embeds included) by inventoried packages.
	InPackages int `json:"inPackages"`
	// GoOutsidePackages lists every tracked .go file outside all packages,
	// each with an explicit class; an unclassifiable .go file fails.
	GoOutsidePackages []TrackedFile `json:"goOutsidePackages,omitempty"`
	// ModuleMetadata lists module definition files (go.mod, go.sum,
	// go.work, go.work.sum) anywhere in the tree.
	ModuleMetadata []string `json:"moduleMetadata,omitempty"`
	// NonBuildFiles counts tracked files that are not build inputs of any
	// package and carry no Go source: documentation, fixtures, CI
	// configuration. They are enumerated by extension as evidence.
	NonBuildFiles       int            `json:"nonBuildFiles"`
	NonBuildByExtension map[string]int `json:"nonBuildByExtension,omitempty"`
	// ClassDigests prove exact per-class membership without enumerating
	// tens of thousands of paths: sha256 over the sorted member paths of
	// each universe class. Any membership change changes the digest, and
	// the attested git tree supplies the per-file identity.
	ClassDigests map[string]string `json:"classDigests"`
	Submodules   []SubmoduleRecord `json:"submodules,omitempty"`
}

// Inventory is the complete pass-1 result.
type Inventory struct {
	SchemaVersion int               `json:"schemaVersion"`
	Universe      Universe          `json:"universe"`
	Module        []ModulePackage   `json:"module"`
	External      []ExternalPackage `json:"external"`
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

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
