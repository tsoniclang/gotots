package source

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/identity"
)

// LoadedPackage is the TRANSIENT per-package artifact of the authoritative
// semantic load. It may hold syntax and package-wide types.Info for every
// file regardless of eventual evidence depth; it exists only between
// LoadUniverse and Finalize and must never be retained as application
// evidence. The finalized Package is a different validated type.
type LoadedPackage struct {
	id              identity.PackageID
	provenance      Provenance
	acquisition     Acquisition
	disposition     LanguageDisposition
	moduleGoVersion string
	requestedRoot   bool
	imports         []string
	files           []*LoadedFile
	otherFiles      []string
	embedFiles      []string
	embedPatterns   []string
	types           *types.Package
	typesInfo       *types.Info
	// cgo evidence (transient): checked-view decls outside the owner root,
	// joined by the exact origin graph at census time.
	checkedDecls []checkedDecl
	synthetics   []SyntheticUnit
	mappings     []CheckedUnitMapping
	// checkedNodes maps each cgo-original unit to its exact checked-view
	// counterpart (transient; retention and the C-dependence derivation
	// consume it).
	checkedNodes map[identity.SourceUnitID]checkedCounterpart
	// implicitUnits are the package's unspelled implicit executable units,
	// censused with typed catalog identities before scope selection.
	implicitUnits []identity.ImplicitUnitID
}

// checkedCounterpart is one origin unit's exact checked-view counterpart node
// and its span in checked-view coordinates.
type checkedCounterpart struct {
	node ast.Node
	span Span
}

// ID is the package's canonical identity.
func (p *LoadedPackage) ID() identity.PackageID { return p.id }

// Provenance is the resolved provenance class.
func (p *LoadedPackage) Provenance() Provenance { return p.provenance }

// Acquisition is the resolved acquisition class.
func (p *LoadedPackage) Acquisition() Acquisition { return p.acquisition }

// Disposition is the language/toolchain contract class.
func (p *LoadedPackage) Disposition() LanguageDisposition { return p.disposition }

// ModuleGoVersion is the owning module's go directive.
func (p *LoadedPackage) ModuleGoVersion() string { return p.moduleGoVersion }

// RequestedRoot reports whether the package was a requested root.
func (p *LoadedPackage) RequestedRoot() bool { return p.requestedRoot }

// Files are the transient per-file records.
func (p *LoadedPackage) Files() []*LoadedFile { return append([]*LoadedFile(nil), p.files...) }

// ImplicitUnits are the package's censused implicit executable units.
func (p *LoadedPackage) ImplicitUnits() []identity.ImplicitUnitID {
	return append([]identity.ImplicitUnitID(nil), p.implicitUnits...)
}

// CheckerView is the narrow, transient type-query capability the analyze
// traversal uses over the one checker graph. It is source-owned and never
// survives into the finalized workspace; the finalized API exposes only
// identity-keyed immutable facts.
func (p *LoadedPackage) CheckerView() *TypeInfoView { return newTypeInfoView(p.typesInfo) }

// Types is the package's node in the one coherent type graph.
func (p *LoadedPackage) Types() *types.Package { return p.types }

// LoadedFile is the TRANSIENT per-file artifact: identity plus parsed syntax
// for whatever the toolchain checked, plus the total top-level unit census.
type LoadedFile struct {
	path             string
	id               identity.FileID
	fset             *token.FileSet
	syntax           *ast.File // nil only for cgo originals and intrinsics
	effectiveVersion string
	overlaid         bool
	cgoOriginal      bool       // checked view lives in transformed files
	censusMode       CensusMode // contract-derived unit acquisition, resolved before load
	byteDigest       SourceSpanHash
	units            []SourceUnit
	// unitRoots maps each unit's root syntax node to its identity, recorded
	// during census from the exact AST edge (never span containment). The
	// analyze traversal consumes it to detect child-implementation boundaries.
	// For cgo originals the roots are the checked-view counterpart nodes.
	unitRoots map[ast.Node]identity.SourceUnitID
	// unitSignatures carries each function-body unit's declaration signature so
	// the body region (rooted at the body, with the signature owned by the file
	// declaration region) can still resolve return forms. Function-literal
	// signatures are in-region and absent here.
	unitSignatures map[identity.SourceUnitID]*ast.FuncType
	// traversalSyntax is the syntax the analyze traversal walks: the file's own
	// checked syntax, or (for cgo originals) the shared origin-graph tree.
	traversalSyntax *ast.File
	traversalFset   *token.FileSet
}

// Syntax is the file's checked syntax for the transient analyze traversal;
// nil for intrinsic/metadata files with no census.
func (f *LoadedFile) Syntax() *ast.File { return f.traversalSyntax }

// TraversalFset is the file set the traversal syntax is measured in.
func (f *LoadedFile) TraversalFset() *token.FileSet { return f.traversalFset }

// UnitRootAt returns the unit a syntax node roots, recorded from the exact AST
// edge during census.
func (f *LoadedFile) UnitRootAt(node ast.Node) (identity.SourceUnitID, bool) {
	id, ok := f.unitRoots[node]
	return id, ok
}

// UnitBoundaries is the transient node->unit map the analyze traversal uses to
// detect child-implementation boundaries (fresh copy).
func (f *LoadedFile) UnitBoundaries() map[ast.Node]identity.SourceUnitID {
	out := make(map[ast.Node]identity.SourceUnitID, len(f.unitRoots))
	for node, id := range f.unitRoots {
		out[node] = id
	}
	return out
}

// UnitRootNode returns the root syntax node of one unit in this file.
func (f *LoadedFile) UnitRootNode(id identity.SourceUnitID) (ast.Node, bool) {
	for node, unit := range f.unitRoots {
		if unit == id {
			return node, true
		}
	}
	return nil, false
}

// UnitSignature returns a function-body unit's declaration signature (the body
// region needs it to resolve return forms); nil for other unit kinds.
func (f *LoadedFile) UnitSignature(id identity.SourceUnitID) *ast.FuncType {
	return f.unitSignatures[id]
}

// Path is the display path.
func (f *LoadedFile) Path() string { return f.path }

// ID is the canonical file identity.
func (f *LoadedFile) ID() identity.FileID { return f.id }

// Units is the censused total unit ledger of the file.
func (f *LoadedFile) Units() []SourceUnit { return append([]SourceUnit(nil), f.units...) }

// EffectiveGoVersion is the file's effective language version.
func (f *LoadedFile) EffectiveGoVersion() string { return f.effectiveVersion }

// checkedDecl is one top-level declaration in a cgo checked-view file. Its
// origin unit (or synthetic identity) and C-dependence are derived by
// resolveCgo from //line and type evidence — never stored as classification
// here.
type checkedDecl struct {
	node ast.Node
}

// Universe is the TRANSIENT resolved source universe: the complete typed
// closure with census, cgo origin evidence, and full syntax/Info. Its only
// consumers are the analysis-scope phase (which selects evidence depths) and
// Finalize (which builds the retained Workspace). Holding a Universe beyond
// finalization is a defect.
type Universe struct {
	fset      *token.FileSet
	toolchain Toolchain
	packages  []*LoadedPackage
	roots     []*LoadedPackage
	request   Request
	manifest  UnitManifest // request-supplied provider unit manifest
}

// Fset carries position information.
func (u *Universe) Fset() *token.FileSet { return u.fset }

// Toolchain is the resolved selected toolchain.
func (u *Universe) Toolchain() Toolchain { return u.toolchain }

// Packages is the complete transient closure in deterministic order.
func (u *Universe) Packages() []*LoadedPackage { return append([]*LoadedPackage(nil), u.packages...) }

// Roots are the requested root packages.
func (u *Universe) Roots() []*LoadedPackage { return append([]*LoadedPackage(nil), u.roots...) }

// Request is the originating compilation request.
func (u *Universe) Request() Request { return u.request }

// Units enumerates the complete unit census of the universe.
func (u *Universe) Units() []SourceUnit {
	var out []SourceUnit
	for _, pkg := range u.packages {
		for _, file := range pkg.files {
			out = append(out, file.units...)
		}
	}
	return out
}
