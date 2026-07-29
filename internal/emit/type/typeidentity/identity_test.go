package typeidentity

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"
)

func TestLocalComponentsIncludesInterfaceMethodContracts(t *testing.T) {
	fileSet := token.NewFileSet()
	source, err := parser.ParseFile(fileSet, "source.go", `package local

func use() {
	type Local int32
	var _ interface {
		Take(Local)
		Return() Local
	}
}
`, 0)
	if err != nil {
		t.Fatal(err)
	}
	info := &types.Info{
		Defs:  make(map[*ast.Ident]types.Object),
		Types: make(map[ast.Expr]types.TypeAndValue),
	}
	checked, err := new(types.Config).Check(
		"example.com/local",
		fileSet,
		[]*ast.File{source},
		info,
	)
	if err != nil {
		t.Fatal(err)
	}
	var local *types.TypeName
	var contract *types.Interface
	for identifier, object := range info.Defs {
		if identifier.Name == "Local" {
			local, _ = object.(*types.TypeName)
		}
	}
	ast.Inspect(source, func(node ast.Node) bool {
		target, ok := node.(*ast.InterfaceType)
		if !ok {
			return true
		}
		represented, ok := info.Types[target].Type.(*types.Interface)
		if ok {
			contract = represented
		}
		return true
	})
	if local == nil || contract == nil || local.Pkg() != checked {
		t.Fatal("interface-local-component fixture is incomplete")
	}
	components := LocalComponents(contract)
	if len(components) != 1 || components[0] != local {
		t.Fatalf("local components = %#v, want Local", components)
	}
}
