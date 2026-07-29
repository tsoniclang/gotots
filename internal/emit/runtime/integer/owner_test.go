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
