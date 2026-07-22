// Package source owns workspace loading and toolchain parsing. It is one of
// two packages (with internal/language/analyze) permitted to import go/ast and
// go/token; downstream layers consume its typed File artifact rather than
// reparsing paths themselves.
package source

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"

	"github.com/tsoniclang/gotots/internal/identity"
)

// LoadError is the typed failure of relating a requested file to its
// workspace root.
type LoadError struct {
	Root   string
	Path   string
	Reason string
}

func (e *LoadError) Error() string {
	return fmt.Sprintf("GOTOTS_SOURCE_LOAD: %s under root %s: %s", e.Path, e.Root, e.Reason)
}

// File is the validated artifact of loading one Go source file. Its fields are
// unexported so a File exists only through Load, where the invariants hold: the
// syntax parsed cleanly, and the identity is workspace-relative and
// machine-independent. The OS path is retained separately for display only.
type File struct {
	path   string
	id     identity.FileID
	fset   *token.FileSet
	syntax *ast.File
}

// Load parses one Go source file inside the given workspace root. The file's
// canonical identity is its root-relative slash path, so identical files under
// identical relative locations carry identical identities on any machine.
// Parse errors are returned typed by the toolchain parser and never downgraded
// into a partial tree.
func Load(root, path string) (*File, error) {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return nil, &LoadError{Root: root, Path: path, Reason: "path is not relative to the root: " + err.Error()}
	}
	rel = filepath.ToSlash(rel)
	if rel == ".." || strings.HasPrefix(rel, "../") {
		return nil, &LoadError{Root: root, Path: path, Reason: "path escapes the workspace root"}
	}
	id, err := identity.NewFileID(rel)
	if err != nil {
		return nil, err
	}
	fset := token.NewFileSet()
	syntax, err := parser.ParseFile(fset, path, nil, parser.ParseComments|parser.SkipObjectResolution)
	if err != nil {
		return nil, err
	}
	return &File{path: path, id: id, fset: fset, syntax: syntax}, nil
}

// Path is the OS path the file was loaded from, for display only. It never
// enters an identity.
func (f *File) Path() string { return f.path }

// ID is the canonical workspace-relative identity of the file.
func (f *File) ID() identity.FileID { return f.id }

// Fset carries the file's position information.
func (f *File) Fset() *token.FileSet { return f.fset }

// Syntax is the parsed file.
func (f *File) Syntax() *ast.File { return f.syntax }
