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
) (api.ExpressionEmission, error) {
	callee, ok := source.Fun.(*ast.Ident)
	if !ok || source.Ellipsis != token.NoPos {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	object, ok := context.TypesInfo().Uses[callee].(*types.Func)
	if !ok || object.Pkg() != context.TypesPackage() {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	signature, ok := object.Type().(*types.Signature)
	if !ok ||
		signature.Recv() != nil ||
		signature.Variadic() ||
		signature.TypeParams().Len() != 0 ||
		signature.Params().Len() != len(source.Args) ||
		signature.Results().Len() != 1 {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	reference, err := context.Names().Reference(object)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	arguments := make([]tsgo.Expression, 0, len(source.Args))
	requests := reference.Requests()
	for index, argument := range source.Args {
		argumentType := context.TypesInfo().TypeOf(argument)
		if argumentType == nil ||
			!types.AssignableTo(argumentType, signature.Params().At(index).Type()) {
			return api.ExpressionEmission{},
				api.Unsupported(context, api.CategoryExpression, source)
		}
		target, err := children.Expression(
			context.
				WithRole(api.RoleCallArgument).
				WithExpectedType(signature.Params().At(index).Type()),
			argument,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		if len(target.Before()) != 0 {
			return api.ExpressionEmission{},
				api.Unsupported(context, api.CategoryExpression, source)
		}
		arguments = append(arguments, target.Value())
		requests = append(requests, target.Requests()...)
	}
	return api.NewExpressionEmission(
		nil,
		context.Factory().CallExpression(
			context.Factory().Identifier(reference.Name()),
			nil,
			nil,
			arguments,
			tsgo.NodeFlagsNone,
		),
		requests,
	)
}
