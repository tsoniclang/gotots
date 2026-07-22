// Package source owns the resolved source universe: it turns a compilation
// request (go.work/go.mod, selected toolchain, patterns, overlays, build
// configuration) into the complete transitive package closure with typed,
// validated identity, provenance, acquisition, language-disposition, version,
// file, and type facts. It is one of two production packages (with
// internal/language/analyze) permitted to import go/ast and go/types.
//
// Source owns identity, provenance, acquisition, files, types, and versions.
// Output paths and implementation ownership belong to later planning and are
// deliberately absent here.
package source

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/identity"
)

// Request is the compilation request the loader resolves. The same request
// shape serves inspection and generation; there is no separate single-file
// route.
type Request struct {
	// Dir is the workspace or module root directory the go tool runs in.
	Dir string
	// Patterns are package patterns; empty means ["./..."].
	Patterns []string
	// GoBinary selects the exact Go toolchain binary; empty resolves "go"
	// from PATH once and records the resolution — the same binary drives the
	// loader and every verifier.
	GoBinary string
	// ProviderContract selects the versioned provider-contract artifact by
	// identity; the compiler never assumes a default. ProviderContractDigest,
	// when set, must match the resolved artifact's fingerprint.
	ProviderContract       string
	ProviderContractDigest string
	// Overlay maps OS file paths to replacement contents.
	Overlay map[string][]byte
	// Env is extra environment (build configuration) appended to the ambient
	// environment, e.g. GOOS/GOARCH/GOFLAGS/GOMODCACHE entries.
	Env []string
	// BuildFlags are extra flags passed to the underlying go tool.
	BuildFlags []string
}

// LoadError is the typed failure of resolving a compilation request.
type LoadError struct {
	Dir    string
	Reason string
}

func (e *LoadError) Error() string {
	return fmt.Sprintf("GOTOTS_SOURCE_LOAD: %s: %s", e.Dir, e.Reason)
}

// Provenance is the closed toolchain-resolved package class.
type Provenance uint8

const (
	ProvenanceInvalid Provenance = iota
	ProvenanceWorkspaceModule
	ProvenanceModuleDependency
	ProvenanceStandardLibrary
	ProvenanceToolchainPackage
	ProvenanceLanguagePseudo

	numProvenances
)

var provenanceNames = [numProvenances]string{
	ProvenanceWorkspaceModule: "workspace-module", ProvenanceModuleDependency: "module-dependency",
	ProvenanceStandardLibrary: "standard-library", ProvenanceToolchainPackage: "toolchain-package",
	ProvenanceLanguagePseudo: "language-pseudo",
}

// Valid reports whether p names a provenance.
func (p Provenance) Valid() bool { return p > ProvenanceInvalid && p < numProvenances }

// String renders p for reports.
func (p Provenance) String() string {
	if p.Valid() {
		return provenanceNames[p]
	}
	return fmt.Sprintf("source.Provenance(%d)", uint8(p))
}

// Acquisition is the closed record of where selected bytes came from. It never
// substitutes for provenance or identity.
type Acquisition uint8

const (
	AcquisitionInvalid Acquisition = iota
	AcquisitionWorkspace
	AcquisitionModuleCache
	AcquisitionVendor
	AcquisitionLocalReplacement
	AcquisitionGOROOT

	numAcquisitions
)

var acquisitionNames = [numAcquisitions]string{
	AcquisitionWorkspace: "workspace", AcquisitionModuleCache: "module-cache",
	AcquisitionVendor: "vendor", AcquisitionLocalReplacement: "local-replacement",
	AcquisitionGOROOT: "goroot",
}

// Valid reports whether a names an acquisition.
func (a Acquisition) Valid() bool { return a > AcquisitionInvalid && a < numAcquisitions }

// String renders a for reports.
func (a Acquisition) String() string {
	if a.Valid() {
		return acquisitionNames[a]
	}
	return fmt.Sprintf("source.Acquisition(%d)", uint8(a))
}

// LanguageDisposition is the closed language/toolchain contract class of a
// package.
type LanguageDisposition uint8

const (
	LanguageDispositionInvalid LanguageDisposition = iota
	DispositionOrdinarySource
	DispositionBuiltinUniverse
	DispositionUnsafeIntrinsic
	DispositionCgoPseudo

	numLanguageDispositions
)

var languageDispositionNames = [numLanguageDispositions]string{
	DispositionOrdinarySource: "ordinary-source", DispositionBuiltinUniverse: "builtin-universe",
	DispositionUnsafeIntrinsic: "unsafe-intrinsic", DispositionCgoPseudo: "cgo-pseudo",
}

// Valid reports whether d names a language disposition; the zero value is
// invalid so an unclassified package cannot masquerade as ordinary source.
func (d LanguageDisposition) Valid() bool {
	return d > LanguageDispositionInvalid && d < numLanguageDispositions
}

// String renders d for reports.
func (d LanguageDisposition) String() string {
	if d.Valid() {
		return languageDispositionNames[d]
	}
	return fmt.Sprintf("source.LanguageDisposition(%d)", uint8(d))
}

// Toolchain is the resolved selected toolchain: the exact binary, its version
// fingerprint, and its GOROOT (an acquisition root, never part of identity).
type Toolchain struct {
	binary  string
	version string
	goroot  string
	goos    string
	goarch  string
}

// Binary is the resolved absolute path of the selected go binary.
func (t Toolchain) Binary() string { return t.binary }

// Version is the toolchain version fingerprint (go env GOVERSION).
func (t Toolchain) Version() string { return t.version }

// GOROOT is the toolchain's GOROOT directory.
func (t Toolchain) GOROOT() string { return t.goroot }

// GOOS is the selected target operating system.
func (t Toolchain) GOOS() string { return t.goos }

// GOARCH is the selected target architecture.
func (t Toolchain) GOARCH() string { return t.goarch }

// Workspace is the typed universe one request resolves to: the complete
// transitive package closure under the selected toolchain.
type Workspace struct {
	fset      *token.FileSet
	toolchain Toolchain
	packages  []*Package // complete closure, deterministic identity order
	roots     []*Package // the requested root packages
}

// Fset carries position information for every loaded file.
func (w *Workspace) Fset() *token.FileSet { return w.fset }

// Toolchain is the resolved selected toolchain.
func (w *Workspace) Toolchain() Toolchain { return w.toolchain }

// Packages is the complete package closure in deterministic order.
func (w *Workspace) Packages() []*Package { return w.packages }

// Roots are the requested root packages. Analysis covers the complete
// source-bearing closure, not only the roots; roots exist for reporting and
// reachability anchoring.
func (w *Workspace) Roots() []*Package { return w.roots }

// Package is one resolved package of the closure. Selected packages carry
// syntax and type information; dependency packages carry identity, provenance,
// acquisition, version, and file facts.
type Package struct {
	id              identity.PackageID
	provenance      Provenance
	acquisition     Acquisition
	disposition     LanguageDisposition
	moduleGoVersion string // module `go` directive; empty for reserved owners
	requestedRoot   bool
	imports         []string // imported package paths, sorted
	files           []*File  // identity-bearing source Go files
	otherFiles      []string // non-Go inputs (C/asm/etc.), owner-relative
	embedFiles      []string // resolved //go:embed inputs, owner-relative
	embedPatterns   []string // declared //go:embed patterns
	mappings        []CheckedUnitMapping
	synthetics      []SyntheticUnit
	types           *types.Package
	typesInfo       *types.Info // uniform-full: checker's Info; mixed: filtered; else nil
}

// admit is the single gate into a workspace: it validates the record and
// appends it. There is no other append site, so an unadmitted record is
// absent from the universe — a dropped input the source-universe verifier
// reports — never silently present unvalidated.
func (w *Workspace) admit(record *Package) error {
	validated, err := finishPackage(record)
	if err != nil {
		return err
	}
	w.packages = append(w.packages, validated)
	if validated.requestedRoot {
		w.roots = append(w.roots, validated)
	}
	return nil
}

// finishPackage is the validating constructor of a Package record: it rejects
// incoherent owner/provenance/acquisition/disposition combinations and
// selected records without complete syntax/type evidence.
func finishPackage(p *Package) (*Package, error) {
	fail := func(reason string) (*Package, error) {
		return nil, &LoadError{Dir: "", Reason: "invalid package record " + p.id.String() + ": " + reason}
	}
	if p.id.IsZero() {
		return fail("zero identity")
	}
	if !p.provenance.Valid() {
		return fail("invalid provenance")
	}
	if !p.acquisition.Valid() {
		return fail("invalid acquisition")
	}
	if !p.disposition.Valid() {
		return fail("invalid language disposition")
	}
	owner := p.id.Owner().Class()
	coherent := map[identity.OwnerClass][]Provenance{
		identity.OwnerModule:          {ProvenanceWorkspaceModule, ProvenanceModuleDependency},
		identity.OwnerStandardLibrary: {ProvenanceStandardLibrary},
		identity.OwnerToolchain:       {ProvenanceToolchainPackage},
		identity.OwnerLanguagePseudo:  {ProvenanceLanguagePseudo},
	}
	provenanceOK := false
	for _, allowed := range coherent[owner] {
		provenanceOK = provenanceOK || p.provenance == allowed
	}
	if !provenanceOK {
		return fail("owner " + owner.String() + " incoherent with provenance " + p.provenance.String())
	}
	switch p.provenance {
	case ProvenanceWorkspaceModule:
		if p.acquisition != AcquisitionWorkspace {
			return fail("workspace module with acquisition " + p.acquisition.String())
		}
	case ProvenanceModuleDependency:
		if p.acquisition == AcquisitionWorkspace || p.acquisition == AcquisitionGOROOT {
			return fail("module dependency with acquisition " + p.acquisition.String())
		}
	default:
		if p.acquisition != AcquisitionGOROOT {
			return fail("reserved owner with acquisition " + p.acquisition.String())
		}
	}
	if owner != identity.OwnerModule && p.moduleGoVersion != "" {
		return fail("reserved owner carries a module go directive")
	}
	// Evidence is required wherever the disposition requires it — for every
	// package in the closure, not only requested roots.
	switch p.disposition {
	case DispositionBuiltinUniverse:
		// metadata-only pseudo-package: no type or syntax evidence exists
	case DispositionUnsafeIntrinsic:
		if p.types == nil {
			return fail("unsafe intrinsic record lacks type evidence")
		}
	default:
		if p.types == nil {
			return fail("source record lacks type evidence")
		}
		if p.types.Path() != p.id.ImportPath() {
			return fail("type-graph package path " + p.types.Path() + " diverges from identity")
		}
		if len(p.files) == 0 {
			return fail("source record has no files")
		}
		anyFull := false
		for _, file := range p.files {
			for _, unit := range file.units {
				if !unit.depth.Valid() {
					return fail("unit " + unit.id.String() + " has no selected evidence depth")
				}
				if unit.depth == DepthFullSemantic {
					anyFull = true
				}
			}
			// Structural evidence must match the file's unit depths.
			switch evidence := file.evidence.(type) {
			case FullSyntax:
				if evidence.Syntax == nil {
					return fail("file " + file.ID().String() + " full-syntax evidence without a tree")
				}
				for _, unit := range file.units {
					if unit.depth != DepthFullSemantic {
						return fail("file " + file.ID().String() + " retains full syntax over non-full unit " + unit.id.String())
					}
				}
			case MixedUnits:
				retained := map[identity.SourceUnitID]bool{}
				for _, r := range evidence.Retained {
					retained[r.Unit] = true
				}
				for _, unit := range file.units {
					if (unit.depth == DepthFullSemantic) != retained[unit.id] {
						return fail("file " + file.ID().String() + " retention diverges from depth for " + unit.id.String())
					}
				}
			case ContractOnly:
				for _, unit := range file.units {
					if unit.depth == DepthFullSemantic {
						return fail("file " + file.ID().String() + " drops full-semantic unit " + unit.id.String())
					}
				}
			default:
				return fail("file " + file.ID().String() + " has no structural evidence state")
			}
		}
		if anyFull && p.typesInfo == nil {
			return fail("full-semantic units without type information")
		}
		if !anyFull && p.typesInfo != nil {
			return fail("retained type information without any full-semantic unit")
		}
	}
	return p, nil
}

// ID is the package's canonical identity.
func (p *Package) ID() identity.PackageID { return p.id }

// Provenance is the resolved toolchain provenance class.
func (p *Package) Provenance() Provenance { return p.provenance }

// Acquisition is where the selected bytes came from.
func (p *Package) Acquisition() Acquisition { return p.acquisition }

// Disposition is the language/toolchain contract class.
func (p *Package) Disposition() LanguageDisposition { return p.disposition }

// ModuleGoVersion is the owning module's go directive; empty for reserved
// owners. It is a module fact, never a per-file permission.
func (p *Package) ModuleGoVersion() string { return p.moduleGoVersion }

// RequestedRoot reports whether the package was a requested root.
func (p *Package) RequestedRoot() bool { return p.requestedRoot }

// RetainsFullSemantic reports whether any unit of the package is
// full-semantic (and therefore contributes retained occurrences).
func (p *Package) RetainsFullSemantic() bool {
	for _, file := range p.files {
		for _, unit := range file.units {
			if unit.depth == DepthFullSemantic {
				return true
			}
		}
	}
	return false
}

// Imports are the imported package paths, sorted (immutable copy).
func (p *Package) Imports() []string { return append([]string(nil), p.imports...) }

// Files are the package's identity-bearing source Go files in deterministic
// order (immutable copy of the collection; File records are themselves
// accessor-only).
func (p *Package) Files() []*File { return append([]*File(nil), p.files...) }

// OtherFiles are the package's non-Go inputs (immutable copy).
func (p *Package) OtherFiles() []string { return append([]string(nil), p.otherFiles...) }

// EmbedFiles are the resolved //go:embed inputs (immutable copy).
func (p *Package) EmbedFiles() []string { return append([]string(nil), p.embedFiles...) }

// EmbedPatterns are the declared //go:embed patterns (immutable copy).
func (p *Package) EmbedPatterns() []string { return append([]string(nil), p.embedPatterns...) }

// CheckedUnitMappings joins source-derived checked declarations to their
// origin units (immutable copy). Checked paths never appear.
func (p *Package) CheckedUnitMappings() []CheckedUnitMapping {
	return append([]CheckedUnitMapping(nil), p.mappings...)
}

// SyntheticUnits are the package-synthetic checked declarations with typed
// origin-derived identities (immutable copy).
func (p *Package) SyntheticUnits() []SyntheticUnit {
	return append([]SyntheticUnit(nil), p.synthetics...)
}

// Units enumerates the package's censused units with selected depths.
func (p *Package) Units() []SourceUnit {
	var out []SourceUnit
	for _, file := range p.files {
		out = append(out, file.units...)
	}
	return out
}

// Types is the package's type evidence: the fully checked package for
// selected roots, and the toolchain's export-data declaration types for
// dependency and standard-library records.
func (p *Package) Types() *types.Package { return p.types }

// TypesInfo is the finalized read-only type-information view; nil when the
// package retains no full-semantic units. The mutable checker maps are never
// exposed.
func (p *Package) TypesInfo() *TypeInfoView { return newTypeInfoView(p.typesInfo) }

// File is one resolved source file. Selected packages' files carry parsed
// syntax and an effective language version; dependency files carry identity
// and display path only.
type File struct {
	path             string
	id               identity.FileID
	fset             *token.FileSet
	evidence         FileEvidence
	effectiveVersion string
	overlaid         bool
	cgoOriginal      bool
	byteDigest       SourceSpanHash // sha256 of the file's selected bytes
	units            []SourceUnit
}

// Path is the OS path the file was loaded from, for display only.
func (f *File) Path() string { return f.path }

// ID is the canonical owner-relative identity of the file.
func (f *File) ID() identity.FileID { return f.id }

// Fset carries the file's position information (selected files only).
func (f *File) Fset() *token.FileSet { return f.fset }

// Evidence is the file's sealed structural evidence state.
func (f *File) Evidence() FileEvidence { return f.evidence }

// FullSyntax returns the complete checked tree when — and only when — every
// unit of the file is full-semantic.
func (f *File) FullSyntax() (*ast.File, bool) {
	full, ok := f.evidence.(FullSyntax)
	if !ok {
		return nil, false
	}
	return full.Syntax, true
}

// Units is the file's total censused unit ledger with selected depths
// (immutable copy).
func (f *File) Units() []SourceUnit { return append([]SourceUnit(nil), f.units...) }

// Overlaid reports whether the file's selected bytes came from an overlay.
func (f *File) Overlaid() bool { return f.overlaid }

// CgoOriginal reports whether the file's checked view lives in cgo
// transformed output.
func (f *File) CgoOriginal() bool { return f.cgoOriginal }

// ByteDigest is the sha256 of the file's selected bytes, captured during
// resolution — consumers never rescan to verify content addressing.
func (f *File) ByteDigest() SourceSpanHash { return f.byteDigest }

// EffectiveGoVersion is the file's effective language version from typed
// toolchain evidence (go/types file-version tracking); it governs construct
// admission for occurrences in this file. Empty for dependency metadata
// records.
func (f *File) EffectiveGoVersion() string { return f.effectiveVersion }
