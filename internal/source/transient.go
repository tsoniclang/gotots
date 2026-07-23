package source

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/identity"
)

// LoadedPackage is the transient package record shared by metadata resolution
// and the one selective semantic hydration. Resolution fills stable
// acquisition facts only. Hydration attaches one coherent go/types node and,
// only for source-plan-local packages, syntax and types.Info. The finalized
// Package is a different validated type.
type LoadedPackage struct {
	id              identity.PackageID
	provenance      Provenance
	acquisition     Acquisition
	disposition     LanguageDisposition
	moduleGoVersion string
	requestedRoot   bool
	imports         []string
	files           []*LoadedFile
	inputs          []loadedInput
	embedPatterns   []string
	types           *types.Package
	typesInfo       *types.Info
	hasCheckedView  bool
	// checkedDecls are cgo-transformed declarations outside the source owner
	// root. Language structure owns their definition/origin interpretation;
	// source records only the checked syntax acquisition.
	checkedDecls []checkedDecl
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

// Imports returns the canonical direct import-path set resolved by the
// selected toolchain.
func (p *LoadedPackage) Imports() []string {
	return append([]string(nil), p.imports...)
}

// Files are the transient per-file records.
func (p *LoadedPackage) Files() []*LoadedFile { return append([]*LoadedFile(nil), p.files...) }

// Inputs returns canonical supplemental input evidence without acquisition
// paths.
func (p *LoadedPackage) Inputs() []Input {
	out := make([]Input, 0, len(p.inputs))
	for _, input := range p.inputs {
		out = append(out, Input{
			id: input.id, kind: input.kind, byteDigest: input.byteDigest,
			overlaid: input.overlaid,
		})
	}
	return out
}

// CheckerView is the narrow, transient type-query capability the analyze
// traversal uses over the one checker graph. It is source-owned and never
// survives into the finalized workspace; the finalized API exposes only
// identity-keyed immutable facts.
func (p *LoadedPackage) CheckerView() *TypeInfoView { return newTypeInfoView(p.typesInfo) }

// Types is the package's node in the one coherent type graph.
func (p *LoadedPackage) Types() *types.Package { return p.types }

// HasCheckedView reports whether the selected toolchain replaces at least one
// physical source file with generated checked syntax, as cgo does. This is
// metadata-resolution evidence and is available before hydration.
func (p *LoadedPackage) HasCheckedView() bool { return p.hasCheckedView }

// EmbedPatterns returns the selected package-relative embed patterns.
func (p *LoadedPackage) EmbedPatterns() []string {
	return append([]string(nil), p.embedPatterns...)
}

// LoadedFile is the transient per-file artifact. Resolution owns identity,
// selected-byte digest, version, and checked-view classification. Selective
// hydration adds bytes and syntax only when the source plan selects this file
// locally.
type LoadedFile struct {
	path             string
	id               identity.FileID
	fset             *token.FileSet
	syntax           *ast.File // nil only for cgo originals and intrinsics
	physicalFset     *token.FileSet
	physicalSyntax   *ast.File
	selectedBytes    []byte
	effectiveVersion string
	overlaid         bool
	cgoOriginal      bool // checked view lives in transformed files
	byteDigest       SourceSpanHash
}

// PhysicalSyntax is the selected source syntax measured in physical source
// coordinates. It is transient and may be consumed only by the Stage-1
// structure/executable owners before finalization.
func (f *LoadedFile) PhysicalSyntax() *ast.File { return f.physicalSyntax }

// PhysicalFileSet measures PhysicalSyntax.
func (f *LoadedFile) PhysicalFileSet() *token.FileSet { return f.physicalFset }

// CheckedSyntax is the toolchain-checked syntax for this exact source file.
// Cgo originals return nil because their checked form is represented by the
// package's checked declarations.
func (f *LoadedFile) CheckedSyntax() *ast.File { return f.syntax }

// SelectedBytes returns a copy of the exact overlay-aware source bytes for a
// locally hydrated file. Certified files deliberately return nil.
func (f *LoadedFile) SelectedBytes() []byte {
	return append([]byte(nil), f.selectedBytes...)
}

// ByteDigest is the sha256 of SelectedBytes captured during resolution.
func (f *LoadedFile) ByteDigest() SourceSpanHash { return f.byteDigest }

// Path is the display path.
func (f *LoadedFile) Path() string { return f.path }

// ID is the canonical file identity.
func (f *LoadedFile) ID() identity.FileID { return f.id }

// EffectiveGoVersion is the file's effective language version.
func (f *LoadedFile) EffectiveGoVersion() string { return f.effectiveVersion }

// CgoOriginal reports whether the file's checked view lives in cgo-transformed
// output (so its full units are inventoried through checked counterparts).
func (f *LoadedFile) CgoOriginal() bool { return f.cgoOriginal }

// checkedDecl is one top-level declaration in a cgo checked-view file. Its
// origin unit (or synthetic identity) and C-dependence are derived by
// resolveCgo from //line and type evidence — never stored as classification
// here.
type checkedDecl struct {
	node ast.Node
}

type loadedInput struct {
	path       string
	id         identity.FileID
	kind       InputKind
	byteDigest SourceSpanHash
	overlaid   bool
}

// CheckedDeclarations returns the transient cgo checked-view declarations.
// The slice is copied; nodes remain owned by the one transient source graph.
func (p *LoadedPackage) CheckedDeclarations() []ast.Node {
	out := make([]ast.Node, len(p.checkedDecls))
	for index := range p.checkedDecls {
		out[index] = p.checkedDecls[index].node
	}
	return out
}

// Universe is the transient source universe. ResolveUniverse first constructs
// the complete metadata closure without parsing dependency interiors.
// HydrateUniverse then creates the one checker graph required by the local
// structural-source decisions. Holding a Universe beyond finalization is a
// defect.
type Universe struct {
	fset            *token.FileSet
	toolchain       Toolchain
	packages        []*LoadedPackage
	roots           []*LoadedPackage
	request         Request
	hydrationOwners map[identity.PackageID]bool
	hydrated        bool
	finalized       bool
}

// Fset carries position information for the selective semantic hydration. It
// is nil before hydration.
func (u *Universe) Fset() *token.FileSet { return u.fset }

// Toolchain is the resolved selected toolchain.
func (u *Universe) Toolchain() Toolchain { return u.toolchain }

// Packages is the complete transient closure in deterministic order.
func (u *Universe) Packages() []*LoadedPackage { return append([]*LoadedPackage(nil), u.packages...) }

// Roots are the requested root packages.
func (u *Universe) Roots() []*LoadedPackage { return append([]*LoadedPackage(nil), u.roots...) }

// Request is the normalized originating compilation request.
func (u *Universe) Request() Request { return cloneRequest(u.request) }

// Hydrated reports whether the sole selective semantic load has completed.
func (u *Universe) Hydrated() bool {
	return u != nil && u.hydrated && !u.finalized
}

// Finalized reports whether Finalize has actively severed all transient
// syntax, source-byte, and checker references.
func (u *Universe) Finalized() bool {
	return u != nil && u.finalized
}
