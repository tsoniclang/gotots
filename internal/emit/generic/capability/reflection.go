package capability

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func emitReflectionType(
	context api.Context,
	operation api.GenericOperation,
	signature *types.Signature,
	arguments []tsgo.Expression,
) (api.ExpressionEmission, error) {
	if signature == nil || len(arguments) != 1 ||
		signature.Params().Len() != 1 || signature.Results().Len() != 1 {
		return api.ExpressionEmission{}, shapeError(context, operation)
	}
	witness, ok := types.Unalias(signature.Params().At(0).Type()).(*types.Pointer)
	reflectionType, okResult := types.Unalias(
		signature.Results().At(0).Type(),
	).(*types.Named)
	if !ok || !okResult || reflectionType.Obj() == nil {
		return api.ExpressionEmission{}, shapeError(context, operation)
	}
	names, ok := context.Names().(api.ReflectionNames)
	if !ok {
		return api.ExpressionEmission{}, &api.ContextError{
			Reason: "reflection names are unavailable",
		}
	}
	reference, err := names.ReflectionType(witness.Elem(), reflectionType.Obj())
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.DirectExpression(
		reference.Expression(context.Factory()),
		reference.Requests()...,
	), nil
}
