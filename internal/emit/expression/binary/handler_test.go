package binary

import (
	"go/ast"
	"go/token"
	"go/types"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/storage"
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
				storage.Owner{},
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

func TestLogicalRightPrerequisitesStayInsideTheSelectedBranch(t *testing.T) {
	context, err := api.NewContext(
		api.RoleReturnResult,
		token.NewFileSet(),
		types.NewPackage("example.com/expression", "expression"),
		&types.Info{},
		types.SizesFor("gc", "amd64"),
		tsgo.Factory{},
		unusedNames{},
		unusedValues{},
		storage.Owner{},
		api.IntegerRepresentationNumber,
		api.EvaluationOrderDirect,
	)
	if err != nil {
		t.Fatal(err)
	}
	factory := context.Factory()
	rightPrerequisite := factory.ExpressionStatement(
		factory.Identifier("rightPrerequisite"),
	)
	right, err := api.NewExpressionEmission(
		[]tsgo.Statement{rightPrerequisite},
		factory.Identifier("rightValue"),
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	result, err := emitLogical(
		context,
		token.LAND,
		factory.BinaryOperatorToken(
			tsgo.BinaryOperatorAmpersandAmpersandToken,
		),
		api.DirectExpression(factory.Identifier("leftValue")),
		right,
	)
	if err != nil {
		t.Fatal(err)
	}
	before := result.Before()
	if len(before) != 2 {
		t.Fatalf("logical prerequisite statements = %d, want 2", len(before))
	}
	if before[0] == rightPrerequisite || before[1] == rightPrerequisite {
		t.Fatal("right prerequisite escaped to the eager outer statement list")
	}
	branch, ok := before[1].(tsgo.IfStatement)
	if !ok {
		t.Fatalf("logical second prerequisite = %T, want tsgo.IfStatement", before[1])
	}
	block, ok := branch.ThenStatement().(tsgo.Block)
	if !ok {
		t.Fatalf("logical branch = %T, want tsgo.Block", branch.ThenStatement())
	}
	statements := block.Statements()
	if len(statements) != 2 || statements[0] != rightPrerequisite {
		t.Fatalf(
			"logical branch prerequisites = %#v, want right prerequisite then assignment",
			statements,
		)
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

func (unusedNames) AnonymousStruct(
	*types.Struct,
	api.AnonymousStructDemand,
) (api.NameReference, error) {
	panic("unused")
}

func (unusedNames) MapSpecialization(
	types.Type,
	api.MapSpecializationDemand,
) (api.NameReference, error) {
	panic("unused")
}

func (unusedNames) InterfaceAdapter(types.Type) (api.NameReference, error) {
	panic("unused")
}

func (unusedNames) InterfaceDynamicType(types.Type) (api.NameReference, error) {
	panic("unused")
}

func (unusedNames) InterfaceType(types.Type) (api.NameReference, error) {
	panic("unused")
}

func (unusedNames) InterfaceContract(
	types.Type,
) (api.InterfaceContractReference, error) {
	panic("unused")
}

func (unusedNames) InterfaceMethodName(*types.Func) (string, error) {
	panic("unused")
}

func (unusedNames) InterfaceMethodToken(
	*types.Func,
) (api.NameReference, error) {
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

func (unusedNames) NamedStructStorage(
	*types.TypeName,
) (api.NameReference, error) {
	panic("unused")
}

func (unusedNames) AnonymousStructStorage(
	*types.Struct,
) (api.NameReference, error) {
	panic("unused")
}

func (unusedNames) ConstantProjection(
	*types.Const,
	types.BasicKind,
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
	return "__logical", nil
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

func (unusedValues) RequiresStructuralCopy(api.Context, types.Type) bool {
	panic("unused")
}

func (unusedValues) SupportsHash(api.Context, types.Type) bool {
	panic("unused")
}

func (unusedValues) RequiresStorageProjection(api.Context, types.Type) bool {
	panic("unused")
}

func (unusedValues) StorageType(
	api.Context,
	ast.Node,
	types.Type,
) (api.TypeEmission, error) {
	panic("unused")
}

func (unusedValues) ToStorage(
	api.Context,
	ast.Node,
	types.Type,
	api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	panic("unused")
}

func (unusedValues) FromStorage(
	api.Context,
	ast.Node,
	types.Type,
	api.ExpressionEmission,
) (api.ExpressionEmission, error) {
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

func (unusedValues) Hash(
	api.Context,
	ast.Node,
	types.Type,
	tsgo.Expression,
) (api.ExpressionEmission, error) {
	panic("unused")
}

func (unusedValues) BinaryUpdate(
	api.Context,
	ast.Node,
	ast.Expr,
	types.Type,
	types.Type,
	token.Token,
	tsgo.Expression,
	api.ExpressionEmission,
) (api.ExpressionEmission, bool, error) {
	panic("unused")
}

func (unusedValues) Increment(
	api.Context,
	ast.Node,
	types.Type,
	token.Token,
	tsgo.Expression,
) (api.ExpressionEmission, bool, error) {
	panic("unused")
}
