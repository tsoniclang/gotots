package call

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
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
	if _, err := configuration.Check("example.com/source", fileSet, []*ast.File{file}, info); err != nil {
		t.Fatal(err)
	}
	selected := make(map[string]bool)
	for expression, selection := range info.Selections {
		selected[expression.X.(*ast.Ident).Name] = concreteMethodSelection(selection)
	}
	if selected["dynamic"] {
		t.Fatal("promoted interface method was classified as a concrete callable")
	}
	if !selected["direct"] {
		t.Fatal("direct concrete method was not classified as a concrete callable")
	}
}

func TestCalleeObjectRejectsEverySelectedToolchainBuiltin(t *testing.T) {
	info := &types.Info{Uses: make(map[*ast.Ident]types.Object)}
	builtins := 0
	for _, name := range types.Universe.Names() {
		object := types.Universe.Lookup(name)
		if _, ok := object.(*types.Builtin); !ok {
			continue
		}
		builtins++
		identifier := &ast.Ident{Name: name}
		info.Uses[identifier] = object
		if actual, ok := calleeObject(api.DirectTypeInfo(info), identifier); ok {
			t.Fatalf("builtin %s resolved as source function %v", name, actual)
		}
	}
	if builtins == 0 {
		t.Fatal("selected toolchain exposes no builtins")
	}

	sourcePackage := types.NewPackage("example.com/source", "source")
	function := types.NewFunc(
		token.Pos(1),
		sourcePackage,
		"Run",
		types.NewSignatureType(nil, nil, nil, nil, nil, false),
	)
	sourcePackage.Scope().Insert(function)
	identifier := &ast.Ident{Name: "Run"}
	info.Uses[identifier] = function
	if actual, ok := calleeObject(api.DirectTypeInfo(info), identifier); !ok || actual != function {
		t.Fatalf("source function resolved as %v, %v", actual, ok)
	}
}
