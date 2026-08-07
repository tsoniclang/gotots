package newvalue

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	pointertype "github.com/tsoniclang/gotots/internal/emit/type/pointer"
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
	return allocate(context, children, source, element, value)
}
