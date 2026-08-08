package slicevalue

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	expressionoperands "github.com/tsoniclang/gotots/internal/emit/expression/operands"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Address(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	element types.Type,
	receiver api.ExpressionEmission,
	index api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	ordered, err := expressionoperands.Preserve(
		context,
		api.TemporaryAddressOperand,
		expressionoperands.Present(receiver),
		expressionoperands.Present(index),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	values := ordered.Values()
	storage, err := context.ContainerStorage().ContainerStorageType(
		context.WithRole(api.RoleStorageType),
		source,
		element,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	runtime, err := context.Names().Runtime(
		api.RuntimeSliceAddress,
		api.ImportPhaseValue,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	target := context.Factory().CallExpression(
		context.Factory().Identifier(runtime.Name()),
		nil,
		[]tsgo.TypeNode{storage.Value()},
		values,
		tsgo.NodeFlagsNone,
	)
	storagePointer, err := api.NewExpressionEmission(
		ordered.Before(),
		target,
		api.CombineRequests(
			ordered.Requests(),
			storage.Requests(),
			runtime.Requests(),
		),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return context.Values().ProjectStoragePointer(
		context,
		source,
		element,
		storagePointer,
	)
}
