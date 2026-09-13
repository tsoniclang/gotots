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
				api.MemoryByteOrderLittleEndian,
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

func TestLogicalOperationAcceptsUntypedBooleanConditionEvidence(t *testing.T) {
	source := &ast.BinaryExpr{
		X:  ast.NewIdent("left"),
		Op: token.LOR,
		Y:  ast.NewIdent("right"),
	}
	untypedBoolean := types.Typ[types.UntypedBool]
	info := &types.Info{Types: map[ast.Expr]types.TypeAndValue{
		source:   {Type: untypedBoolean},
		source.X: {Type: untypedBoolean},
		source.Y: {Type: untypedBoolean},
	}}
	context, err := api.NewContext(
		api.RoleIfCondition,
		token.NewFileSet(),
		types.NewPackage("example.com/expression", "expression"),
		info,
		types.SizesFor("gc", "amd64"),
		api.MemoryByteOrderLittleEndian,
		tsgo.Factory{},
		unusedNames{},
		unusedValues{},
		api.IntegerRepresentationNumber,
		api.EvaluationOrderDirect,
	)
	if err != nil {
		t.Fatal(err)
	}
	_, operandType, ok := operationFor(context, source)
	if !ok || !types.Identical(operandType, types.Typ[types.Bool]) {
		t.Fatalf(
			"untyped logical condition = handled %v, operand %v",
			ok,
			operandType,
		)
	}
}

func TestLogicalRightPrerequisitesStayInsideTheSelectedBranch(t *testing.T) {
	context, err := api.NewContext(
		api.RoleReturnResult,
		token.NewFileSet(),
		types.NewPackage("example.com/expression", "expression"),
		&types.Info{},
		types.SizesFor("gc", "amd64"),
		api.MemoryByteOrderLittleEndian,
		tsgo.Factory{},
		unusedNames{},
		unusedValues{},
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
	declaration := before[0].(tsgo.VariableStatement).DeclarationList().Declarations()[0]
	boolean, ok := declaration.Type().(tsgo.KeywordTypeNode)
	if !ok || boolean.Kind() != tsgo.SyntaxKind(tsgo.KeywordTypeSyntaxKindBooleanKeyword) {
		t.Fatalf("logical temporary requires an explicit boolean type, got %T", declaration.Type())
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
