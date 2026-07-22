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

// CgoUnsupportedError is the typed disposition of a selected root package that
// requires cgo preprocessing: an explicit unsupported disposition, never a
// silent omission or a generic path failure.
type CgoUnsupportedError struct {
	ImportPath string
}

func (e *CgoUnsupportedError) Error() string {
	return fmt.Sprintf("GOTOTS_CGO_UNSUPPORTED: selected package %s requires cgo preprocessing; cgo bodies are an explicit external obligation, not ordinary source", e.ImportPath)
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
}

// Binary is the resolved absolute path of the selected go binary.
func (t Toolchain) Binary() string { return t.binary }

// Version is the toolchain version fingerprint (go env GOVERSION).
func (t Toolchain) Version() string { return t.version }

// GOROOT is the toolchain's GOROOT directory.
func (t Toolchain) GOROOT() string { return t.goroot }

// Workspace is the typed universe one request resolves to: the complete
// transitive package closure under the selected toolchain.
type Workspace struct {
	fset      *token.FileSet
	toolchain Toolchain
	packages  []*Package // complete closure, deterministic identity order
	selected  []*Package // the requested root packages, syntax-bearing
}

// Fset carries position information for every loaded file.
func (w *Workspace) Fset() *token.FileSet { return w.fset }

// Toolchain is the resolved selected toolchain.
func (w *Workspace) Toolchain() Toolchain { return w.toolchain }

// Packages is the complete package closure in deterministic order.
func (w *Workspace) Packages() []*Package { return w.packages }

// Selected are the requested root packages (syntax- and type-bearing).
func (w *Workspace) Selected() []*Package { return w.selected }

// Package is one resolved package of the closure. Selected packages carry
// syntax and type information; dependency packages carry identity, provenance,
// acquisition, version, and file facts.
type Package struct {
	id              identity.PackageID
	provenance      Provenance
	acquisition     Acquisition
	disposition     LanguageDisposition
	moduleGoVersion string // module `go` directive; empty for reserved owners
	selected        bool
	imports         []string // imported package paths, sorted
	files           []*File
	types           *types.Package
	typesInfo       *types.Info
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
	if validated.selected {
		w.selected = append(w.selected, validated)
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
	if p.selected {
		if p.types == nil || p.typesInfo == nil {
			return fail("selected record lacks type evidence")
		}
		for _, file := range p.files {
			if file.Syntax() == nil {
				return fail("selected record file " + file.ID().String() + " lacks syntax")
			}
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

// Selected reports whether the package was a requested root.
func (p *Package) Selected() bool { return p.selected }

// Imports are the imported package paths, sorted.
func (p *Package) Imports() []string { return p.imports }

// Files are the package's files in deterministic order. Selected packages
// carry parsed syntax; dependency records carry identity and path only.
func (p *Package) Files() []*File { return p.files }

// Types is the package's type evidence: the fully checked package for
// selected roots, and the toolchain's export-data declaration types for
// dependency and standard-library records.
func (p *Package) Types() *types.Package { return p.types }

// TypesInfo is the package's type information (selected packages only).
func (p *Package) TypesInfo() *types.Info { return p.typesInfo }

// File is one resolved source file. Selected packages' files carry parsed
// syntax and an effective language version; dependency files carry identity
// and display path only.
type File struct {
	path             string
	id               identity.FileID
	fset             *token.FileSet
	syntax           *ast.File
	effectiveVersion string
}

// Path is the OS path the file was loaded from, for display only.
func (f *File) Path() string { return f.path }

// ID is the canonical owner-relative identity of the file.
func (f *File) ID() identity.FileID { return f.id }

// Fset carries the file's position information (selected files only).
func (f *File) Fset() *token.FileSet { return f.fset }

// Syntax is the parsed file; nil for dependency metadata records.
func (f *File) Syntax() *ast.File { return f.syntax }

// EffectiveGoVersion is the file's effective language version from typed
// toolchain evidence (go/types file-version tracking); it governs construct
// admission for occurrences in this file. Empty for dependency metadata
// records.
func (f *File) EffectiveGoVersion() string { return f.effectiveVersion }
