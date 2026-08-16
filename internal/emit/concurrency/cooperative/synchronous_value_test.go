package cooperative

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"
)

func TestConcreteMethodSelectionRejectsPromotedInterfaceMethod(t *testing.T) {
	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, "source.go", `package source

type comparer interface { compare(int, int) int }
type promoted struct { comparer }
type concrete struct{}

func (concrete) compare(int, int) int { return 0 }

func use(dynamic promoted, direct concrete) {
	_ = dynamic.compare
	_ = direct.compare
}
`, 0)
	if err != nil {
		t.Fatal(err)
	}
	info := &types.Info{Selections: make(map[*ast.SelectorExpr]*types.Selection)}
	configuration := &types.Config{}
	if _, err := configuration.Check(
		"example.com/source",
		fileSet,
		[]*ast.File{file},
		info,
	); err != nil {
		t.Fatal(err)
	}
	selected := make(map[string]bool)
	for expression, selection := range info.Selections {
		selected[expression.X.(*ast.Ident).Name] =
			concreteMethodSelection(selection)
	}
	if selected["dynamic"] {
		t.Fatal("promoted interface method was classified as a concrete callable")
	}
	if !selected["direct"] {
		t.Fatal("direct concrete method was not classified as a concrete callable")
	}
}
