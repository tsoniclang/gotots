// Package source owns workspace loading and toolchain parsing: it resolves a
// compilation request (go.work/go.mod, selected packages, overlays, build
// configuration) into a typed, validated universe of modules, packages, and
// parsed files with complete type information. It is one of two packages (with
// internal/language/analyze) permitted to import go/ast and go/types;
// downstream layers consume its typed artifacts rather than reparsing.
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
	// Overlay maps OS file paths to replacement contents.
	Overlay map[string][]byte
	// Env is extra environment (build configuration) appended to the ambient
	// environment, e.g. GOOS/GOARCH/GOFLAGS entries.
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

// Workspace is the typed universe one request resolves to.
type Workspace struct {
	fset      *token.FileSet
	packages  []*Package
	goVersion string
}

// Fset carries position information for every loaded file.
func (w *Workspace) Fset() *token.FileSet { return w.fset }

// Packages are the selected packages in deterministic import-path order.
func (w *Workspace) Packages() []*Package { return w.packages }

// GoVersion is the selected language version (highest module go directive).
func (w *Workspace) GoVersion() string { return w.goVersion }

// Package is one selected, fully type-checked package.
type Package struct {
	id        identity.PackageID
	files     []*File
	types     *types.Package
	typesInfo *types.Info
}

// ID is the package's canonical identity.
func (p *Package) ID() identity.PackageID { return p.id }

// Files are the package's parsed files in deterministic path order.
func (p *Package) Files() []*File { return p.files }

// Types is the type-checked package.
func (p *Package) Types() *types.Package { return p.types }

// TypesInfo is the package's complete type information.
func (p *Package) TypesInfo() *types.Info { return p.typesInfo }

// File is the validated artifact of one loaded Go source file. Its fields are
// unexported so a File exists only through LoadWorkspace, where the invariants
// hold: the syntax parsed cleanly and the identity is module-relative and
// machine-independent. The OS path is retained separately for display only.
type File struct {
	path   string
	id     identity.FileID
	fset   *token.FileSet
	syntax *ast.File
}

// Path is the OS path the file was loaded from, for display only.
func (f *File) Path() string { return f.path }

// ID is the canonical module-relative identity of the file.
func (f *File) ID() identity.FileID { return f.id }

// Fset carries the file's position information.
func (f *File) Fset() *token.FileSet { return f.fset }

// Syntax is the parsed file.
func (f *File) Syntax() *ast.File { return f.syntax }
