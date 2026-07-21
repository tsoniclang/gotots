// Package source owns workspace loading and toolchain parsing. It is one of
// two packages (with internal/language/analyze) permitted to import go/ast and
// go/token; downstream layers consume its typed results, never the raw AST.
package source

import (
	"go/ast"
	"go/parser"
	"go/token"
)

// ParseGoFile parses a single Go source file into its AST using the standard
// toolchain parser, retaining comments. Parse errors are returned typed by the
// parser and never downgraded into a partial tree.
func ParseGoFile(path string) (*token.FileSet, *ast.File, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return nil, nil, err
	}
	return fset, file, nil
}
