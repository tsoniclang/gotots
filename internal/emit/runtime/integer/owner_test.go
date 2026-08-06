package integer

import (
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
	panicruntime "github.com/tsoniclang/gotots/internal/emit/runtime/panic"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestBuildCreatesExactTypedDivideAndRemainderFunctions(t *testing.T) {
	statements, err := Build(
		tsgo.NewFactory(),
		[]api.RuntimeSymbol{
			api.RuntimeIntegerDivide,
			api.RuntimeIntegerRemainder,
		},
		"GoPanic",
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(statements) != 2 {
		t.Fatalf("integer runtime statements = %d, want two", len(statements))
	}
	for index, expectedOperator := range []tsgo.SyntaxKind{
		tsgo.SyntaxKindSlashToken,
		tsgo.SyntaxKindPercentToken,
	} {
		function := statements[index].(tsgo.FunctionDeclaration)
		if len(function.Parameters()) != 2 ||
			function.Parameters()[0].Type().Kind() !=
				tsgo.SyntaxKindBigIntKeyword ||
			function.Parameters()[1].Type().Kind() !=
				tsgo.SyntaxKindBigIntKeyword ||
			function.Type().Kind() != tsgo.SyntaxKindBigIntKeyword {
			t.Fatalf("integer helper %d is not bigint -> bigint", index)
		}
		body := function.Body().(tsgo.Block).Statements()
		if len(body) != 2 {
			t.Fatalf("integer helper %d statements = %d, want guard and return", index, len(body))
		}
		guard := body[0].(tsgo.IfStatement)
		panicCall := guard.ThenStatement().(tsgo.Block).
			Statements()[0].(tsgo.ExpressionStatement).
			Expression().(tsgo.CallExpression)
		panicMember := panicCall.Expression().(tsgo.PropertyAccessExpression)
		if panicMember.Expression().(tsgo.Identifier).Text() != "GoPanic" ||
			panicMember.Name().(tsgo.Identifier).Text() !=
				panicruntime.RaiseRuntimeName {
			t.Fatalf("integer helper %d bypasses the shared panic ABI", index)
		}
		operation := body[1].(tsgo.ReturnStatement).
			Expression().(tsgo.BinaryExpression)
		if operation.OperatorToken().Kind() != expectedOperator {
			t.Fatalf(
				"integer helper %d operator = %d, want %d",
				index,
				operation.OperatorToken().Kind(),
				expectedOperator,
			)
		}
	}
}

func TestBuildCreatesExactNumberDivideAndRemainderFunctions(t *testing.T) {
	statements, err := Build(
		tsgo.NewFactory(),
		[]api.RuntimeSymbol{
			api.RuntimeNumberIntDivide,
			api.RuntimeNumberIntRemainder,
		},
		"GoPanic",
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(statements) != 2 {
		t.Fatalf("number runtime statements = %d, want two", len(statements))
	}
	for index := range statements {
		function := statements[index].(tsgo.FunctionDeclaration)
		if len(function.Parameters()) != 2 ||
			function.Parameters()[0].Type().Kind() !=
				tsgo.SyntaxKindNumberKeyword ||
			function.Parameters()[1].Type().Kind() !=
				tsgo.SyntaxKindNumberKeyword ||
			function.Type().Kind() != tsgo.SyntaxKindNumberKeyword {
			t.Fatalf("number integer helper %d is not number -> number", index)
		}
		body := function.Body().(tsgo.Block).Statements()
		if len(body) != 3 {
			t.Fatalf("number helper %d statements = %d, want guard, result, and return", index, len(body))
		}
		guard := body[0].(tsgo.IfStatement)
		panicCall := guard.ThenStatement().(tsgo.Block).
			Statements()[0].(tsgo.ExpressionStatement).
			Expression().(tsgo.CallExpression)
		panicMember := panicCall.Expression().(tsgo.PropertyAccessExpression)
		if panicMember.Expression().(tsgo.Identifier).Text() != "GoPanic" ||
			panicMember.Name().(tsgo.Identifier).Text() !=
				panicruntime.RaiseRuntimeName {
			t.Fatalf("number helper %d bypasses the shared panic ABI", index)
		}
		returned := body[2].(tsgo.ReturnStatement).
			Expression().(tsgo.ConditionalExpression)
		if returned.WhenTrue().(tsgo.NumericLiteral).Text() != "0" ||
			returned.WhenFalse().(tsgo.Identifier).Text() != "result" {
			t.Fatalf("number helper %d does not normalize signed zero", index)
		}
	}
	divide := statements[0].(tsgo.FunctionDeclaration).
		Body().(tsgo.Block).Statements()[1].(tsgo.VariableStatement).
		DeclarationList().Declarations()[0].Initializer().(tsgo.CallExpression)
	member := divide.Expression().(tsgo.PropertyAccessExpression)
	if member.Expression().(tsgo.Identifier).Text() != "Math" ||
		member.Name().(tsgo.Identifier).Text() != "trunc" {
		t.Fatal("number integer division does not truncate toward zero")
	}
}

func TestBuildCreatesExactFixedWidthFunctions(t *testing.T) {
	statements, err := Build(
		tsgo.NewFactory(),
		[]api.RuntimeSymbol{
			api.RuntimeIntegerNormalizeSigned64,
			api.RuntimeIntegerNormalizeUnsigned64,
		},
		"GoPanic",
	)
	if err != nil {
		t.Fatal(err)
	}
	for index, expected := range []string{"asIntN", "asUintN"} {
		function := statements[index].(tsgo.FunctionDeclaration)
		if len(function.Parameters()) != 1 ||
			function.Parameters()[0].Type().Kind() !=
				tsgo.SyntaxKindBigIntKeyword ||
			function.Type().Kind() != tsgo.SyntaxKindBigIntKeyword {
			t.Fatalf("fixed-width helper %d is not bigint -> bigint", index)
		}
		call := function.Body().(tsgo.Block).Statements()[0].(tsgo.ReturnStatement).Expression().(tsgo.CallExpression)
		member := call.Expression().(tsgo.PropertyAccessExpression)
		name := member.Name().(tsgo.Identifier)
		if name.Text() != expected ||
			len(call.Arguments()) != 2 ||
			call.Arguments()[0].(tsgo.NumericLiteral).Text() != "64" {
			t.Fatalf("fixed-width helper %d does not call BigInt.%s(64, value)", index, expected)
		}
	}
}

func TestBuildRejectsSymbolsOwnedByOtherRuntimeModules(t *testing.T) {
	for _, symbols := range [][]api.RuntimeSymbol{
		nil,
		{api.RuntimePointer},
		{api.RuntimeIntegerDivide, api.RuntimeIntegerDivide},
	} {
		if _, err := Build(
			tsgo.NewFactory(),
			symbols,
			"GoPanic",
		); err == nil {
			t.Fatalf("integer runtime accepted symbols %v", symbols)
		}
	}
}

func TestBuildCreatesTypedStableIntegerMaxAndMin(t *testing.T) {
	statements, err := Build(
		tsgo.NewFactory(),
		[]api.RuntimeSymbol{
			api.RuntimeIntegerMax,
			api.RuntimeIntegerMin,
		},
		"GoPanic",
	)
	if err != nil {
		t.Fatal(err)
	}
	for index, operator := range []tsgo.SyntaxKind{
		tsgo.SyntaxKindGreaterThanEqualsToken,
		tsgo.SyntaxKindLessThanEqualsToken,
	} {
		function := statements[index].(tsgo.FunctionDeclaration)
		body := function.Body().(tsgo.Block).Statements()
		selected := body[0].(tsgo.ReturnStatement).
			Expression().(tsgo.ConditionalExpression)
		condition := selected.Condition().(tsgo.BinaryExpression)
		if condition.OperatorToken().Kind() != operator ||
			selected.WhenTrue().(tsgo.Identifier).Text() != "left" ||
			selected.WhenFalse().(tsgo.Identifier).Text() != "right" {
			t.Fatalf("ordered integer helper %d is not stable", index)
		}
	}
}
