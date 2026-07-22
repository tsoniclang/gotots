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
	// joined by //line origin evidence at census time.
	checkedDecls []checkedDecl
	synthetics   []SyntheticUnit
	mappings     []CheckedUnitMapping
	// implicitUnits are the package's unspelled implicit executable units,
	// censused with typed catalog identities before scope selection.
	implicitUnits []identity.ImplicitUnitID
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
}

// Path is the display path.
func (f *LoadedFile) Path() string { return f.path }

// ID is the canonical file identity.
func (f *LoadedFile) ID() identity.FileID { return f.id }

// Units is the censused total unit ledger of the file.
func (f *LoadedFile) Units() []SourceUnit { return append([]SourceUnit(nil), f.units...) }

// checkedDecl is one top-level declaration in a cgo checked-view file,
// carrying its origin classification.
type checkedDecl struct {
	node     ast.Node
	name     string
	origin   identity.SourceUnitID // zero for synthetic decls
	span     Span                  // span within the checked view
	fromFile string                // checked file display path (transient only)
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
