// Package source owns workspace loading and toolchain parsing. It is one of
// two packages (with internal/language/analyze) permitted to import go/ast and
// go/token; downstream layers consume its typed File artifact rather than
// reparsing paths themselves.
package source

import (
	"go/ast"
	"go/parser"
	"go/token"
)

// File is the typed result of loading one Go source file: its path, the
// position information, and the parsed syntax. Analysis consumes this artifact;
// the orchestrating compiler holds it opaquely and never reparses.
type File struct {
	Path   string
	Fset   *token.FileSet
	Syntax *ast.File
}

// Load parses a single Go source file, retaining comments. Parse errors are
// returned typed by the toolchain parser and never downgraded into a partial
// tree.
func Load(path string) (*File, error) {
	fset := token.NewFileSet()
	syntax, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	return &File{Path: path, Fset: fset, Syntax: syntax}, nil
}
