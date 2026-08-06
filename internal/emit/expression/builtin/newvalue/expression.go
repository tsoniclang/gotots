package newvalue

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	pointerruntime "github.com/tsoniclang/gotots/internal/emit/runtime/pointer"
	pointertype "github.com/tsoniclang/gotots/internal/emit/type/pointer"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func emitExpression(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
	builtin *types.Builtin,
) (api.ExpressionEmission, error) {
	if source == nil ||
		builtin == nil ||
		types.Object(builtin) != types.Universe.Lookup("new") ||
		len(source.Args) != 1 {
		return api.ExpressionEmission{}, api.Unsupported(
			context,
			api.CategoryExpression,
			source,
		)
	}
	argument := source.Args[0]
	argumentType := context.TypesInfo().TypeOf(argument)
	resultType := context.TypesInfo().TypeOf(source)
	pointer, element, ok := pointertype.Resolve(resultType)
	if !ok || argumentType == nil || !types.Identical(argumentType, element) {
		return api.ExpressionEmission{}, api.Unsupported(
			context,
			api.CategoryExpression,
			source,
		)
	}
	if expected := context.ExpectedType(); expected == nil ||
		!types.AssignableTo(pointer, expected) {
		return api.ExpressionEmission{}, api.Unsupported(
			context,
			api.CategoryExpression,
			source,
		)
	}
	value, err := children.Expression(
		context.
			WithRole(api.RoleCallArgument).
			WithExpectedType(element),
		argument,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	value, err = context.Values().Transfer(
		context.WithRole(api.RoleCallArgument),
		argument,
		argumentType,
		element,
		api.ValueTransferCopy,
		value,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	representation, err := pointertype.Observe(
		context,
		pointer,
		api.PointerRepresentationDemandNone,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if representation.Representation().DirectClass() {
		return api.NewExpressionEmission(
			value.Before(),
			value.Value(),
			api.CombineRequests(
				value.Requests(),
				representation.Requests(),
			),
		)
	}
	represented, err := children.RepresentedType(
		context.WithRole(api.RoleCallArgument),
		source,
		element,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	storageType, err := context.ContainerStorage().PointerStorageType(
		context.WithRole(api.RoleStorageType),
		source,
		element,
		representation,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	value, err = context.ContainerStorage().ToPointerStorage(
		context.WithRole(api.RoleCallArgument),
		source,
		element,
		representation,
		value,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	reference, err := context.Names().Runtime(
		api.RuntimePointer,
		api.ImportPhaseValue,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.NewExpressionEmission(
		value.Before(),
		context.Factory().CallExpression(
			context.Factory().PropertyAccessExpression(
				reference.Expression(context.Factory()),
				nil,
				context.Factory().Identifier(pointerruntime.CellName),
				tsgo.NodeFlagsNone,
			),
			nil,
			[]tsgo.TypeNode{
				represented.Value(),
				storageType.Value(),
			},
			[]tsgo.Expression{value.Value()},
			tsgo.NodeFlagsNone,
		),
		api.CombineRequests(
			value.Requests(),
			represented.Requests(),
			storageType.Requests(),
			reference.Requests(),
			representation.Requests(),
		),
	)
}
