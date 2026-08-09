package newvalue

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	pointertype "github.com/tsoniclang/gotots/internal/emit/type/pointer"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
	builtin *types.Builtin,
) (api.ExpressionEmission, error) {
	if source == nil ||
		types.Object(builtin) != types.Universe.Lookup("new") {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	if source.Ellipsis != token.NoPos ||
		len(source.Args) != 1 {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	argumentFacts, ok := context.TypesInfo().TypeAndValue(source.Args[0])
	if !ok {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	if !argumentFacts.IsType() {
		return emitExpression(context, children, source, builtin)
	}
	resultType := context.TypesInfo().TypeOf(source)
	pointer, element, represented := pointertype.Resolve(resultType)
	if !represented ||
		!types.Identical(context.TypesInfo().TypeOf(source.Args[0]), element) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	if expected := context.ExpectedType(); expected == nil ||
		!types.AssignableTo(pointer, expected) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	zero, err := context.Values().Zero(
		context.WithRole(api.RoleCallArgument),
		source.Args[0],
		element,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if len(zero.Before()) != 0 {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	return allocate(context, children, source, element, zero)
}
