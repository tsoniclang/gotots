package call

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
) (tsgo.Expression, error) {
	callee, ok := source.Fun.(*ast.Ident)
	if !ok || source.Ellipsis != token.NoPos {
		return nil, api.Unsupported(context, api.CategoryExpression, source)
	}
	object, ok := context.TypesInfo().Uses[callee].(*types.Func)
	if !ok || object.Pkg() != context.TypesPackage() {
		return nil, api.Unsupported(context, api.CategoryExpression, source)
	}
	signature, ok := object.Type().(*types.Signature)
	if !ok ||
		signature.Recv() != nil ||
		signature.Variadic() ||
		signature.TypeParams().Len() != 0 ||
		signature.Params().Len() != len(source.Args) ||
		signature.Results().Len() != 1 {
		return nil, api.Unsupported(context, api.CategoryExpression, source)
	}
	targetName, err := context.Names().Reference(object)
	if err != nil {
		return nil, err
	}
	arguments := make([]tsgo.Expression, 0, len(source.Args))
	for index, argument := range source.Args {
		argumentType := context.TypesInfo().TypeOf(argument)
		if argumentType == nil ||
			!types.AssignableTo(argumentType, signature.Params().At(index).Type()) {
			return nil, api.Unsupported(context, api.CategoryExpression, source)
		}
		target, err := children.Expression(
			context.WithRole(api.RoleCallArgument),
			argument,
		)
		if err != nil {
			return nil, err
		}
		arguments = append(arguments, target)
	}
	return context.Factory().CallExpression(
		context.Factory().Identifier(targetName),
		nil,
		nil,
		arguments,
		tsgo.NodeFlagsNone,
	), nil
}
