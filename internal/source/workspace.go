// Package source owns the resolved source universe: it turns a compilation
// request (go.work/go.mod, selected toolchain, patterns, overlays, build
// configuration) into the complete transitive package closure with validated
// identity, provenance, acquisition, language-disposition, version, and file
// facts.
//
// Source owns identity, provenance, acquisition, files, bytes, transient
// checker lifetime, and versions.
// Output paths and implementation ownership belong to later planning and are
// deliberately absent here.
package source

import (
	"fmt"

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
	// ProviderContract selects the versioned provider contract by IDENTITY;
	// the compiler never assumes a default. ProviderContractDigest, when set,
	// must match the resolved contract's fingerprint. ProviderContractArtifact
	// is separate acquisition data: the path of a contract-artifact file that
	// must declare exactly the selected identity; empty means the built-in
	// registry.
	ProviderContract         string
	ProviderContractDigest   string
	ProviderContractArtifact string
	// Overlay maps OS file paths to replacement contents.
	Overlay map[string][]byte
	// Env is extra environment (build configuration) appended to the ambient
	// environment, e.g. GOOS/GOARCH/GOFLAGS/GOMODCACHE entries.
	Env []string
	// BuildFlags are extra flags passed to the underlying go tool.
	BuildFlags []string
	// ProviderStructureArtifact and ProviderSemanticArtifact are the separate
	// certified provider authorities selected by ordinary compilation. Both
	// are required whenever the source plan selects provider-owned evidence.
	// Their externally selected digests, not their internal seals, establish
	// authority.
	ProviderStructureArtifact string
	ProviderStructureDigest   string
	ProviderSemanticArtifact  string
	ProviderSemanticDigest    string
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

	numLanguageDispositions
)

var languageDispositionNames = [numLanguageDispositions]string{
	DispositionOrdinarySource: "ordinary-source", DispositionBuiltinUniverse: "builtin-universe",
	DispositionUnsafeIntrinsic: "unsafe-intrinsic",
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

// Toolchain is the resolved selected toolchain: the exact binary with its
// content digest, its version fingerprint, its GOROOT (an acquisition root,
// never part of identity), and the complete resolved build-selection
// environment (GOOS/GOARCH/GOEXPERIMENT/GOFLAGS/CGO_ENABLED).
type Toolchain struct {
	binary            string
	binaryDigest      string
	version           string
	goroot            string
	goos              string
	goarch            string
	experiments       string
	goflags           string
	cgoEnabled        string
	buildConfigDigest string
}

// Binary is the resolved absolute path of the selected go binary.
func (t Toolchain) Binary() string { return t.binary }

// BinaryDigest is the sha256 of the selected go binary's bytes.
func (t Toolchain) BinaryDigest() string { return t.binaryDigest }

// Experiments is the resolved GOEXPERIMENT selection.
func (t Toolchain) Experiments() string { return t.experiments }

// GoFlags is the resolved GOFLAGS selection.
func (t Toolchain) GoFlags() string { return t.goflags }

// CgoEnabled is the resolved CGO_ENABLED selection.
func (t Toolchain) CgoEnabled() string { return t.cgoEnabled }

// BuildConfigurationDigest binds every target/compiler/cgo configuration
// value that can change selected or checked Go source while excluding
// acquisition/cache paths.
func (t Toolchain) BuildConfigurationDigest() string {
	return t.buildConfigDigest
}

// Version is the toolchain version fingerprint (go env GOVERSION).
func (t Toolchain) Version() string { return t.version }

// GOROOT is the toolchain's GOROOT directory.
func (t Toolchain) GOROOT() string { return t.goroot }

// GOOS is the selected target operating system.
func (t Toolchain) GOOS() string { return t.goos }

// GOARCH is the selected target architecture.
func (t Toolchain) GOARCH() string { return t.goarch }

// Workspace is the finalized source-acquisition universe. Definition,
// selection, and executable artifacts remain separate phase outputs.
type Workspace struct {
	toolchain             Toolchain
	resolutionFingerprint string
	packages              []*Package
	packagesByID          map[identity.PackageID]*Package
	roots                 []*Package
}

func (w *Workspace) Toolchain() Toolchain { return w.toolchain }
func (w *Workspace) Packages() []*Package {
	return append([]*Package(nil), w.packages...)
}
func (w *Workspace) Roots() []*Package { return append([]*Package(nil), w.roots...) }
func (w *Workspace) ResolutionFingerprint() string {
	return w.resolutionFingerprint
}

// Package is immutable physical/acquisition evidence only.
type Package struct {
	id              identity.PackageID
	provenance      Provenance
	acquisition     Acquisition
	disposition     LanguageDisposition
	moduleGoVersion string
	requestedRoot   bool
	imports         []string
	files           []*File
	inputs          []Input
	embedPatterns   []string
	hasCheckedView  bool
}

func (w *Workspace) admit(record *Package) error {
	validated, err := finishPackage(record)
	if err != nil {
		return err
	}
	if w.packagesByID == nil {
		w.packagesByID = map[identity.PackageID]*Package{}
	}
	if _, duplicate := w.packagesByID[validated.id]; duplicate {
		return &LoadError{
			Reason: "duplicate package identity " + validated.id.String(),
		}
	}
	w.packagesByID[validated.id] = validated
	w.packages = append(w.packages, validated)
	if validated.requestedRoot {
		w.roots = append(w.roots, validated)
	}
	return nil
}

func finishPackage(p *Package) (*Package, error) {
	fail := func(reason string) (*Package, error) {
		return nil, &LoadError{Reason: "invalid package " + p.id.String() + ": " + reason}
	}
	if p.id.IsZero() || !p.provenance.Valid() || !p.acquisition.Valid() ||
		!p.disposition.Valid() {
		return fail("identity, provenance, acquisition, or disposition is invalid")
	}
	coherent := map[identity.OwnerClass][]Provenance{
		identity.OwnerModule:          {ProvenanceWorkspaceModule, ProvenanceModuleDependency},
		identity.OwnerStandardLibrary: {ProvenanceStandardLibrary},
		identity.OwnerToolchain:       {ProvenanceToolchainPackage},
		identity.OwnerLanguagePseudo:  {ProvenanceLanguagePseudo},
	}
	validProvenance := false
	for _, allowed := range coherent[p.id.Owner().Class()] {
		validProvenance = validProvenance || p.provenance == allowed
	}
	if !validProvenance {
		return fail("owner and provenance disagree")
	}
	if p.provenance == ProvenanceWorkspaceModule && p.acquisition != AcquisitionWorkspace {
		return fail("workspace module is not workspace-acquired")
	}
	if p.provenance == ProvenanceModuleDependency &&
		(p.acquisition == AcquisitionWorkspace || p.acquisition == AcquisitionGOROOT) {
		return fail("module dependency has impossible acquisition")
	}
	if p.id.Owner().Class() != identity.OwnerModule &&
		p.acquisition != AcquisitionGOROOT {
		return fail("reserved owner is not GOROOT-acquired")
	}
	if p.id.Owner().Class() != identity.OwnerModule && p.moduleGoVersion != "" {
		return fail("reserved owner carries a module go directive")
	}
	switch p.disposition {
	case DispositionBuiltinUniverse:
		if p.id.Owner().Class() != identity.OwnerLanguagePseudo ||
			p.id.ImportPath() != "builtin" {
			return fail("builtin disposition has the wrong identity")
		}
	case DispositionUnsafeIntrinsic:
		if p.id.Owner().Class() != identity.OwnerStandardLibrary ||
			p.id.ImportPath() != "unsafe" {
			return fail("unsafe disposition has the wrong identity")
		}
	case DispositionOrdinarySource:
		if p.id.ImportPath() == "builtin" || p.id.ImportPath() == "unsafe" {
			return fail("intrinsic package has ordinary disposition")
		}
	}
	if p.disposition != DispositionBuiltinUniverse && len(p.files) == 0 {
		return fail("source package has no files")
	}
	if p.hasCheckedView && p.disposition != DispositionOrdinarySource {
		return fail("non-ordinary package claims a checked source view")
	}
	if !strictlySortedUnique(p.imports) ||
		!strictlySortedUnique(p.embedPatterns) {
		return fail("imports or embed patterns are not canonical")
	}
	for _, imported := range p.imports {
		if imported == "" {
			return fail("imports contain an invalid semantic edge")
		}
	}
	seenFiles := map[identity.FileID]bool{}
	var previousFile identity.FileID
	for _, file := range p.files {
		if file == nil ||
			file.id.IsZero() ||
			file.id.Owner() != p.id.Owner() ||
			file.byteDigest.IsZero() ||
			seenFiles[file.id] ||
			(!previousFile.IsZero() &&
				file.id.Compare(previousFile) <= 0) {
			return fail("source file is invalid, duplicated, or noncanonical")
		}
		if file.effectiveVersion == "" &&
			!file.cgoOriginal &&
			p.disposition != DispositionUnsafeIntrinsic {
			return fail("checked Go file lacks an effective language version")
		}
		seenFiles[file.id] = true
		previousFile = file.id
	}
	for _, file := range p.files {
		if file.cgoOriginal && !p.hasCheckedView {
			return fail("cgo source exists without a checked package view")
		}
	}
	var previousInput identity.FileID
	for _, input := range p.inputs {
		if input.id.IsZero() ||
			input.id.Owner() != p.id.Owner() ||
			!input.kind.Valid() ||
			input.byteDigest.IsZero() ||
			seenFiles[input.id] ||
			(!previousInput.IsZero() &&
				input.id.Compare(previousInput) <= 0) {
			return fail(
				"supplemental input is invalid, colliding, or noncanonical",
			)
		}
		seenFiles[input.id] = true
		previousInput = input.id
	}
	return p, nil
}

func strictlySortedUnique(values []string) bool {
	for index, value := range values {
		if value == "" || (index > 0 && value <= values[index-1]) {
			return false
		}
	}
	return true
}

func (p *Package) ID() identity.PackageID           { return p.id }
func (p *Package) Provenance() Provenance           { return p.provenance }
func (p *Package) Acquisition() Acquisition         { return p.acquisition }
func (p *Package) Disposition() LanguageDisposition { return p.disposition }
func (p *Package) ModuleGoVersion() string          { return p.moduleGoVersion }
func (p *Package) RequestedRoot() bool              { return p.requestedRoot }
func (p *Package) Imports() []string                { return append([]string(nil), p.imports...) }
func (p *Package) Files() []*File                   { return append([]*File(nil), p.files...) }
func (p *Package) Inputs() []Input                  { return append([]Input(nil), p.inputs...) }
func (p *Package) EmbedPatterns() []string {
	return append([]string(nil), p.embedPatterns...)
}
func (p *Package) HasCheckedView() bool { return p.hasCheckedView }

// File is immutable selected-file evidence with no AST or definition state.
type File struct {
	id               identity.FileID
	effectiveVersion string
	overlaid         bool
	cgoOriginal      bool
	byteDigest       SourceSpanHash
}

func (f *File) ID() identity.FileID        { return f.id }
func (f *File) Overlaid() bool             { return f.overlaid }
func (f *File) CgoOriginal() bool          { return f.cgoOriginal }
func (f *File) ByteDigest() SourceSpanHash { return f.byteDigest }
func (f *File) EffectiveGoVersion() string { return f.effectiveVersion }

// InputKind is the closed class of non-Go bytes that can affect package
// checking, cgo transformation, embedding, or generated provider evidence.
type InputKind uint8

const (
	InputInvalid InputKind = iota
	InputC
	InputCXX
	InputObjectiveC
	InputHeader
	InputFortran
	InputAssembly
	InputSWIG
	InputSWIGCXX
	InputSyso
	InputEmbed

	numInputKinds
)

func (k InputKind) Valid() bool {
	return k > InputInvalid && k < numInputKinds
}

func (k InputKind) String() string {
	switch k {
	case InputC:
		return "c"
	case InputCXX:
		return "cxx"
	case InputObjectiveC:
		return "objective-c"
	case InputHeader:
		return "header"
	case InputFortran:
		return "fortran"
	case InputAssembly:
		return "assembly"
	case InputSWIG:
		return "swig"
	case InputSWIGCXX:
		return "swig-cxx"
	case InputSyso:
		return "syso"
	case InputEmbed:
		return "embed"
	default:
		return fmt.Sprintf("source.InputKind(%d)", uint8(k))
	}
}

// Input is one canonical supplemental source input. Acquisition paths never
// survive finalization.
type Input struct {
	id         identity.FileID
	kind       InputKind
	byteDigest SourceSpanHash
	overlaid   bool
}

func (i Input) ID() identity.FileID        { return i.id }
func (i Input) Kind() InputKind            { return i.kind }
func (i Input) ByteDigest() SourceSpanHash { return i.byteDigest }
func (i Input) Overlaid() bool             { return i.overlaid }
