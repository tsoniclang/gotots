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
	return addressOperation(context, source, element, api.RuntimeSliceAddress, receiver, index)
}

func Data(context api.Context, source ast.Node, element types.Type, receiver api.ExpressionEmission) (api.ExpressionEmission, error) {
	zero, err := context.ContainerStorage().ContainerStorageZero(context.WithRole(api.RoleStorageType), source, element)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	storage, err := context.ContainerStorage().ContainerStorageType(context.WithRole(api.RoleStorageType), source, element)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	factory := api.DirectExpression(context.Factory().ArrowFunction(nil, nil, nil, storage.Value(),
		context.Factory().EqualsGreaterThanToken(), context.Factory().Block(
			append(zero.Before(), context.Factory().ReturnStatement(zero.Value())), true,
		)), api.CombineRequests(storage.Requests(), zero.Requests())...)
	return addressOperation(context, source, element, api.RuntimeSliceData, receiver, factory)
}

func addressOperation(context api.Context, source ast.Node, element types.Type, symbol api.RuntimeSymbol, receiver, operand api.ExpressionEmission) (api.ExpressionEmission, error) {
	ordered, err := expressionoperands.Preserve(
		context,
		api.TemporaryAddressOperand,
		expressionoperands.Present(receiver),
		expressionoperands.Present(operand),
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
		symbol,
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
