package capability

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func emitAssignment(context api.Context, signature *types.Signature, arguments []tsgo.Expression) (api.ExpressionEmission, error) {
	if signature.Params().Len() != 2 || len(arguments) != 2 || signature.Results().Len() != 1 ||
		!types.Identical(signature.Params().At(0).Type(), signature.Params().At(1).Type()) ||
		!types.Identical(signature.Params().At(0).Type(), signature.Results().At(0).Type()) {
		return api.ExpressionEmission{}, shapeError(context, api.GenericOperationAssign)
	}
	sourceType := signature.Params().At(0).Type()
	switch sourceType.Underlying().(type) {
	case *types.Array, *types.Struct:
		assigned, err := context.StableAssignments().AssignStable(context, nil, sourceType, arguments[0], api.DirectExpression(arguments[1]))
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		return api.NewExpressionEmission(append(assigned.Before(), context.Factory().ExpressionStatement(assigned.Value())), arguments[0], assigned.Requests())
	default:
		if api.ContainsGenericTypeParameter(sourceType) {
			return api.ExpressionEmission{}, invariant(context, "generic assignment capability has no concrete representation")
		}
		return api.DirectExpression(arguments[1]), nil
	}
}
