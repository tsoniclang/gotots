package resulttuple

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/value/interfacevalue"
)

func AdaptAssignment(
	context api.Context,
	source ast.Node,
	sourceType types.Type,
	targetType types.Type,
	value api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	if sourceType == nil ||
		targetType == nil ||
		!types.AssignableTo(sourceType, targetType) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	if types.Identical(
		types.Unalias(sourceType),
		types.Unalias(targetType),
	) {
		return value, nil
	}
	adapted, handled, err := interfacevalue.Assign(
		context,
		source,
		sourceType,
		targetType,
		value,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if handled {
		return adapted, nil
	}
	return value, nil
}
