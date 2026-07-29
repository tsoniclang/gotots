package representation

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func panicNilRuntimeValue(
	context api.Context,
	sourceType types.Type,
) bool {
	return context.GoRuntimeType(sourceType) ==
		api.GoRuntimeTypePanicNilError
}

func panicNilZero(
	context api.Context,
) (api.ExpressionEmission, error) {
	reference, err := context.Names().Runtime(
		api.RuntimePanicNilError,
		api.ImportPhaseValue,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.DirectExpression(
		context.Factory().NewExpression(
			context.Factory().Identifier(reference.Name()),
			nil,
			nil,
		),
		reference.Requests()...,
	), nil
}

func panicNilCopy(
	context api.Context,
	value api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	target, err := panicNilZero(context)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.NewExpressionEmission(
		value.Before(),
		target.Value(),
		api.CombineRequests(
			value.Requests(),
			target.Requests(),
		),
	)
}

func panicNilEqual(
	context api.Context,
) api.ExpressionEmission {
	return api.DirectExpression(context.Factory().TrueLiteral())
}

func panicNilHash(
	context api.Context,
) api.ExpressionEmission {
	return api.DirectExpression(
		context.Factory().NumericLiteral("0", tsgo.TokenFlagsNone),
	)
}
