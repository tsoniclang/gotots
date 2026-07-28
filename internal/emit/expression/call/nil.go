package call

import (
	"go/ast"

	"github.com/tsoniclang/gotots/internal/emit/api"
	panicruntime "github.com/tsoniclang/gotots/internal/emit/runtime/panic"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func knownNonNil(source ast.Expr) bool {
	switch source := source.(type) {
	case *ast.FuncLit:
		return true
	case *ast.ParenExpr:
		return knownNonNil(source.X)
	default:
		return false
	}
}

func nilGuard(
	context api.Context,
	callee tsgo.Expression,
) (tsgo.Statement, []api.RootRequest, error) {
	reference, err := context.Names().Runtime(
		api.RuntimePanic,
		api.ImportPhaseValue,
	)
	if err != nil {
		return nil, nil, err
	}
	condition := context.Factory().BinaryExpression(
		nil,
		callee,
		nil,
		context.Factory().BinaryOperatorToken(
			tsgo.BinaryOperatorEqualsEqualsEqualsToken,
		),
		context.Factory().VoidExpression(
			context.Factory().NumericLiteral("0", tsgo.TokenFlagsNone),
		),
	)
	raise := panicruntime.Call(
		context.Factory(),
		reference.Name(),
		context.Factory().StringLiteral(
			"call of nil function",
			tsgo.TokenFlagsNone,
		),
	)
	return context.Factory().IfStatement(
		condition,
		context.Factory().Block(
			[]tsgo.Statement{
				context.Factory().ExpressionStatement(raise),
			},
			true,
		),
		nil,
	), reference.Requests(), nil
}
