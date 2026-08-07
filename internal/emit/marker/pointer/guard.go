package pointer

import (
	"github.com/tsoniclang/gotots/internal/emit/api"
	panicruntime "github.com/tsoniclang/gotots/internal/emit/runtime/panic"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Guard(
	context api.Context,
	pointer api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	panicReference, err := context.Names().Runtime(
		api.RuntimePanic,
		api.ImportPhaseValue,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	raise := panicruntime.Call(
		context.Factory(),
		panicReference.Name(),
		context.Factory().StringLiteral(
			"invalid memory address or nil pointer dereference",
			tsgo.TokenFlagsNone,
		),
	)
	return api.NewExpressionEmission(
		pointer.Before(),
		context.Factory().ParenthesizedExpression(
			context.Factory().BinaryExpression(
				nil,
				pointer.Value(),
				nil,
				context.Factory().BinaryOperatorToken(
					tsgo.BinaryOperatorQuestionQuestionToken,
				),
				raise,
			),
		),
		api.CombineRequests(
			pointer.Requests(),
			panicReference.Requests(),
		),
	)
}
