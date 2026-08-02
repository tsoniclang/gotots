package recovercall

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	panicruntime "github.com/tsoniclang/gotots/internal/emit/runtime/panic"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Emit(
	context api.Context,
	source *ast.CallExpr,
	builtin *types.Builtin,
) (api.ExpressionEmission, bool, error) {
	if types.Object(builtin) != types.Universe.Lookup("recover") {
		return api.ExpressionEmission{}, false, nil
	}
	if source == nil || len(source.Args) != 0 || source.Ellipsis.IsValid() {
		return api.ExpressionEmission{}, true,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	if !context.CallableControl().Recovery() {
		request, err := context.CallableControlRequest(
			api.CallableControlRecovery,
		)
		if err != nil {
			return api.ExpressionEmission{}, true, err
		}
		return api.DirectExpression(
			context.Factory().VoidExpression(
				context.Factory().NumericLiteral("0", tsgo.TokenFlagsNone),
			),
			request,
		), true, nil
	}
	authority, available := context.RecoveryAuthority()
	if !available {
		return api.DirectExpression(
			context.Factory().Identifier("undefined"),
		), true, nil
	}
	return api.DirectExpression(
		context.Factory().ConditionalExpression(
			context.Factory().BinaryExpression(
				nil,
				context.Factory().Identifier(authority),
				nil,
				context.Factory().BinaryOperatorToken(
					tsgo.BinaryOperatorEqualsEqualsEqualsToken,
				),
				context.Factory().Identifier("undefined"),
			),
			context.Factory().QuestionToken(),
			context.Factory().Identifier("undefined"),
			context.Factory().ColonToken(),
			context.Factory().CallExpression(
				context.Factory().PropertyAccessExpression(
					context.Factory().Identifier(
						authority,
					),
					nil,
					context.Factory().Identifier(panicruntime.TakeName),
					tsgo.NodeFlagsNone,
				),
				nil,
				nil,
				nil,
				tsgo.NodeFlagsNone,
			),
		),
	), true, nil
}
