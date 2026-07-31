package slicevalue

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func loadElement(
	context api.Context,
	source ast.Node,
	elementType types.Type,
	value tsgo.Expression,
) (api.ExpressionEmission, error) {
	return context.ContainerStorage().FromContainerStorage(
		context.WithRole(api.RoleSliceElement),
		source,
		elementType,
		api.DirectExpression(value),
	)
}

func storeElement(
	context api.Context,
	source ast.Node,
	elementType types.Type,
	value api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	return context.ContainerStorage().ToContainerStorage(
		context.WithRole(api.RoleSliceElement),
		source,
		elementType,
		value,
	)
}
