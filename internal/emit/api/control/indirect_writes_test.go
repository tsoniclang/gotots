package control

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"
)

func TestIndirectWritesUseExactLexicalIdentity(test *testing.T) {
	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, "source.go", `package fixture
type Count int
func (value *Count) Change() { *value = 2 }
func Run() {
	addressed := 1
	_ = &addressed
	captured := 2
	callback := func() { captured++ }
	callback()
	plain := 3
	_ = plain
	{ addressed := 4; _ = addressed }
	var receiver Count
	receiver.Change()
}`, 0)
	if err != nil {
		test.Fatal(err)
	}
	info := &types.Info{Types: make(map[ast.Expr]types.TypeAndValue),
		Defs: make(map[*ast.Ident]types.Object), Uses: make(map[*ast.Ident]types.Object),
		Selections: make(map[*ast.SelectorExpr]*types.Selection)}
	if _, err := new(types.Config).Check("fixture", fileSet, []*ast.File{file}, info); err != nil {
		test.Fatal(err)
	}
	callable := file.Decls[2].(*ast.FuncDecl)
	writes := IndirectWrites(callable, info)
	if len(writes) != 3 {
		test.Fatalf("indirect writes = %d, want addressed, captured and receiver", len(writes))
	}
	for identifier, object := range info.Uses {
		variable, ok := object.(*types.Var)
		if !ok || identifier.Pos() < callable.Pos() {
			continue
		}
		_, exposed := writes[variable]
		actual := IndirectlyMutable(identifier, info, writes)
		identifier.Name = "forged"
		if actual != exposed || IndirectlyMutable(identifier, info, writes) != exposed {
			test.Fatal("indirect mutation followed source spelling or a shadowed binding")
		}
	}
}
