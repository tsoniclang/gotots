package slicevalue

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	expressionoperands "github.com/tsoniclang/gotots/internal/emit/expression/operands"
	pointertype "github.com/tsoniclang/gotots/internal/emit/type/pointer"
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
	logical, err := children.RepresentedType(
		context.WithRole(api.RoleSliceElement),
		source,
		element,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	representation, err := pointertype.Observe(
		context,
		types.NewPointer(element),
		api.PointerRepresentationDemandDynamicLocation,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	storage, err := context.ContainerStorage().PointerStorageType(
		context.WithRole(api.RoleStorageType),
		source,
		element,
		representation,
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
		[]tsgo.TypeNode{logical.Value(), storage.Value()},
		values,
		tsgo.NodeFlagsNone,
	)
	return api.NewExpressionEmission(
		ordered.Before(),
		target,
		api.CombineRequests(
			ordered.Requests(),
			logical.Requests(),
			storage.Requests(),
			representation.Requests(),
			runtime.Requests(),
		),
	)
}
