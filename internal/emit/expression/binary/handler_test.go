package binary

import (
	"go/ast"
	"go/token"
	"go/types"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestParentOperatorOwnerDoesNotCreateAnIntegerFallback(t *testing.T) {
	testCases := []struct {
		name       string
		sourceType types.Type
		arch       string
	}{
		{name: "int32", sourceType: types.Typ[types.Int32], arch: "amd64"},
		{name: "32-bit int", sourceType: types.Typ[types.Int], arch: "386"},
		{name: "64-bit int", sourceType: types.Typ[types.Int], arch: "amd64"},
		{name: "int64", sourceType: types.Typ[types.Int64], arch: "386"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			source := &ast.BinaryExpr{
				X:  ast.NewIdent("left"),
				Op: token.MUL,
				Y:  ast.NewIdent("right"),
			}
			info := &types.Info{Types: map[ast.Expr]types.TypeAndValue{
				source:   {Type: testCase.sourceType},
				source.X: {Type: testCase.sourceType},
				source.Y: {Type: testCase.sourceType},
			}}
			context, err := api.NewContext(
				api.RoleReturnResult,
				token.NewFileSet(),
				types.NewPackage("example.com/expression", "expression"),
				info,
				types.SizesFor("gc", testCase.arch),
				tsgo.Factory{},
				unusedNames{},
				unusedValues{},
				api.IntegerRepresentationNumber,
				api.EvaluationOrderDirect,
			)
			if err != nil {
				t.Fatal(err)
			}

			_, _, ok := operationFor(context, source)
			if ok {
				t.Fatal("parent binary owner admitted an integer fallback")
			}
		})
	}
}

type unusedNames struct{}

func (unusedNames) Declare(types.Object) (string, error) {
	panic("unused")
}

func (unusedNames) Parameter(*types.Var, int) (string, error) {
	panic("unused")
}

func (unusedNames) Reference(types.Object) (api.NameReference, error) {
	panic("unused")
}

func (unusedNames) TypeReference(types.Object) (api.NameReference, error) {
	panic("unused")
}

func (unusedNames) PackageVariable(
	*types.Var,
) (api.PackageVariableReference, error) {
	panic("unused")
}

func (unusedNames) NamedStructOperation(
	*types.TypeName,
	api.NamedStructOperation,
) (api.NameReference, error) {
	panic("unused")
}

func (unusedNames) Member(*types.Var) (string, error) {
	panic("unused")
}

func (unusedNames) Primitive(api.PrimitiveAlias) (api.NameReference, error) {
	panic("unused")
}

func (unusedNames) Runtime(
	api.RuntimeSymbol,
	api.ImportPhase,
) (api.NameReference, error) {
	panic("unused")
}

func (unusedNames) Temporary(api.TemporaryKind) (string, error) {
	panic("unused")
}

func (unusedNames) ModuleExport(types.Object) (bool, error) {
	panic("unused")
}

type unusedValues struct{}

func (unusedValues) RequiresCustomEquality(api.Context, types.Type) bool {
	panic("unused")
}

func (unusedValues) RequiresExplicitType(api.Context, types.Type) bool {
	panic("unused")
}

func (unusedValues) Zero(
	api.Context,
	ast.Node,
	types.Type,
) (api.ExpressionEmission, error) {
	panic("unused")
}

func (unusedValues) Copy(
	api.Context,
	ast.Node,
	types.Type,
	api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	panic("unused")
}

func (unusedValues) Assign(
	api.Context,
	ast.Node,
	types.Type,
	tsgo.Expression,
	api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	panic("unused")
}

func (unusedValues) Equal(
	api.Context,
	ast.Node,
	types.Type,
	tsgo.Expression,
	tsgo.Expression,
) (api.ExpressionEmission, error) {
	panic("unused")
}
