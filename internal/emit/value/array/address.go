package array

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	expressionoperands "github.com/tsoniclang/gotots/internal/emit/expression/operands"
	pointerruntime "github.com/tsoniclang/gotots/internal/emit/runtime/pointer"
	pointertype "github.com/tsoniclang/gotots/internal/emit/type/pointer"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func (a RuntimeArray) Address(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	parent api.ExpressionEmission,
	index api.ExpressionEmission,
	checkNil bool,
) (api.ExpressionEmission, error) {
	ordered, err := expressionoperands.Preserve(
		context,
		api.TemporaryAddressOperand,
		expressionoperands.Present(parent),
		expressionoperands.Present(index),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	values := ordered.Values()
	elementLogical, err := children.RepresentedType(
		context.WithRole(api.RoleArrayElement),
		source,
		a.ElementType(),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	elementRepresentation, err := pointertype.Observe(
		context,
		types.NewPointer(a.ElementType()),
		api.PointerRepresentationDemandDynamicLocation,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	elementStorage, err := context.ContainerStorage().PointerStorageType(
		context.WithRole(api.RoleStorageType),
		source,
		a.ElementType(),
		elementRepresentation,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	arrayLogical, err := a.EmitType(
		context.WithRole(api.RoleArrayReceiver),
		children,
		source,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	arrayRepresentation, err := pointertype.Observe(
		context,
		types.NewPointer(a.SourceType()),
		api.PointerRepresentationDemandStableLocation,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	arrayStorage, err := context.ContainerStorage().PointerStorageType(
		context.WithRole(api.RoleStorageType),
		source,
		a.SourceType(),
		arrayRepresentation,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	runtime, err := context.Names().Runtime(
		api.RuntimePointer,
		api.ImportPhaseValue,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	parentValue := values[0]
	if checkNil {
		parentValue = pointerruntime.Dereference(
			context.Factory(),
			runtime.Name(),
			arrayLogical.Value(),
			arrayStorage.Value(),
			parentValue,
		)
	}
	target := context.Factory().CallExpression(
		context.Factory().PropertyAccessExpression(
			context.Factory().Identifier(runtime.Name()),
			nil,
			context.Factory().Identifier(pointerruntime.IndexName),
			tsgo.NodeFlagsNone,
		),
		nil,
		[]tsgo.TypeNode{
			elementLogical.Value(),
			elementStorage.Value(),
			arrayLogical.Value(),
			arrayStorage.Value(),
		},
		[]tsgo.Expression{parentValue, values[1]},
		tsgo.NodeFlagsNone,
	)
	return api.NewExpressionEmission(
		ordered.Before(),
		target,
		api.CombineRequests(
			ordered.Requests(),
			elementLogical.Requests(),
			elementStorage.Requests(),
			elementRepresentation.Requests(),
			arrayLogical.Requests(),
			arrayStorage.Requests(),
			arrayRepresentation.Requests(),
			runtime.Requests(),
		),
	)
}
