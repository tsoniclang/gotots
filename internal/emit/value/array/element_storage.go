package array

import (
	"go/ast"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func (a RuntimeArray) loadElement(
	context api.Context,
	source ast.Node,
	value tsgo.Expression,
) (api.ExpressionEmission, error) {
	return context.ContainerStorage().FromContainerStorage(
		context.WithRole(api.RoleArrayElement),
		source,
		a.ElementType(),
		api.DirectExpression(value),
	)
}

func (a RuntimeArray) zeroElement(
	context api.Context,
	source ast.Node,
) (api.ExpressionEmission, error) {
	return context.ContainerStorage().ContainerStorageZero(
		context,
		source,
		a.ElementType(),
	)
}

func (a RuntimeArray) storeElement(
	context api.Context,
	source ast.Node,
	value api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	return context.ContainerStorage().ToContainerStorage(
		context.WithRole(api.RoleArrayElement),
		source,
		a.ElementType(),
		value,
	)
}
