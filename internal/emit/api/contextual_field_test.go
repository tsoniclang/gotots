package api

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"testing"
)

func TestContextualStructFieldPreservesGenericDeclarationAndSelectedTypes(
	t *testing.T,
) {
	info, literals := contextualFieldFixture(t)
	field, ok := DirectTypeInfo(info).StructFieldOf(
		literals[0],
		literals[0].Elts[0].(*ast.KeyValueExpr).Key.(*ast.Ident),
	)
	if !ok {
		key := literals[0].Elts[0].(*ast.KeyValueExpr).Key.(*ast.Ident)
		t.Fatalf(
			"generic keyed field was not resolved: literal=%v key=%v",
			info.TypeOf(literals[0]),
			info.Uses[key],
		)
	}
	if _, ok := field.Declaration().Type().(*types.TypeParam); !ok {
		t.Fatalf("declaration field type = %T", field.Declaration().Type())
	}
	if !types.Identical(field.Selected().Type(), types.Typ[types.Int]) {
		t.Fatalf("selected field type = %v", field.Selected().Type())
	}
}

func TestContextualStructFieldResolvesElidedPointerLiteral(t *testing.T) {
	info, literals := contextualFieldFixture(t)
	field, ok := DirectTypeInfo(info).StructFieldOf(
		literals[1],
		literals[1].Elts[0].(*ast.KeyValueExpr).Key.(*ast.Ident),
	)
	if !ok || field.Index() != 0 || field.Declaration().Name() != "Value" {
		key := literals[1].Elts[0].(*ast.KeyValueExpr).Key.(*ast.Ident)
		t.Fatalf(
			"elided pointer field = %#v, %t: literal=%v key=%v",
			field,
			ok,
			info.TypeOf(literals[1]),
			info.Uses[key],
		)
	}
}

func contextualFieldFixture(t *testing.T) (*types.Info, []*ast.CompositeLit) {
	t.Helper()
	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, "fixture.go", `package fixture
type Box[T any] struct { Value T }
var Generic = Box[int]{Value: 1}
var Elided = []*Box[int]{{Value: 2}}
`, 0)
	if err != nil {
		t.Fatal(err)
	}
	info := &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
		Defs:  make(map[*ast.Ident]types.Object),
		Uses:  make(map[*ast.Ident]types.Object),
	}
	config := types.Config{Importer: importer.Default()}
	if _, err := config.Check("example.com/fixture", fileSet, []*ast.File{file}, info); err != nil {
		t.Fatal(err)
	}
	var literals []*ast.CompositeLit
	ast.Inspect(file, func(node ast.Node) bool {
		literal, ok := node.(*ast.CompositeLit)
		if ok && len(literal.Elts) != 0 {
			if _, keyed := literal.Elts[0].(*ast.KeyValueExpr); keyed {
				literals = append(literals, literal)
			}
		}
		return true
	})
	if len(literals) != 2 {
		t.Fatalf("keyed literals = %d", len(literals))
	}
	return info, literals
}
