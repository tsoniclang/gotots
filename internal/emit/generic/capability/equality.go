package capability

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func emitEquality(
	context api.Context,
	operation api.GenericOperation,
	signature *types.Signature,
	arguments []tsgo.Expression,
	negated bool,
) (api.ExpressionEmission, error) {
	if len(arguments) != 2 ||
		!types.Identical(
			signature.Params().At(0).Type(),
			signature.Params().At(1).Type(),
		) {
		return api.ExpressionEmission{}, shapeError(context, operation)
	}
	result, err := context.Values().Equal(
		context,
		nil,
		signature.Params().At(0).Type(),
		arguments[0],
		arguments[1],
	)
	if err != nil || !negated {
		return result, err
	}
	return api.NewExpressionEmission(
		result.Before(),
		context.Factory().PrefixUnaryExpression(
			tsgo.PrefixUnaryExpressionOperatorKindExclamationToken,
			result.Value(),
		),
		result.Requests(),
	)
}

func shapeError(
	context api.Context,
	operation api.GenericOperation,
) error {
	return invariant(
		context,
		"generic capability signature has invalid operation shape: "+
			operation.String(),
	)
}

func invariant(context api.Context, reason string) error {
	return &api.InvariantError{
		Role:   context.Role(),
		Reason: reason,
	}
}
