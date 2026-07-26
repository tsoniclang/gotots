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
	return emit(context, children, source, false)
}

func EmitDiscarded(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
) (api.ExpressionEmission, error) {
	return emit(context, children, source, true)
}

func emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
	discarded bool,
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
		signature.TypeParams().Len() != 0 {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	resultCount := 0
	if signature.Results() != nil {
		resultCount = signature.Results().Len()
	}
	if !discarded {
		if resultCount == 0 {
			return api.ExpressionEmission{},
				api.Unsupported(context, api.CategoryExpression, source)
		} else if resultCount == 1 {
			if context.ExpectedResults() != nil {
				return api.ExpressionEmission{},
					api.Unsupported(context, api.CategoryExpression, source)
			}
		} else if expected := context.ExpectedResults(); expected == nil ||
			!types.Identical(signature.Results(), expected) {
			return api.ExpressionEmission{},
				api.Unsupported(context, api.CategoryExpression, source)
		}
	}
	reference, err := context.Names().Reference(object)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	arguments, before, argumentRequests, err := emitArguments(
		context,
		children,
		source,
		signature,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.NewExpressionEmission(
		before,
		context.Factory().CallExpression(
			context.Factory().Identifier(reference.Name()),
			nil,
			nil,
			arguments,
			tsgo.NodeFlagsNone,
		),
		api.CombineRequests(reference.Requests(), argumentRequests),
	)
}

func emitArguments(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
	signature *types.Signature,
) ([]tsgo.Expression, []tsgo.Statement, []api.PlacementRequest, error) {
	if len(source.Args) == 1 {
		if results, ok := context.TypesInfo().TypeOf(source.Args[0]).(*types.Tuple); ok {
			return emitMultipleArgument(context, children, source, signature, results)
		}
	}
	if signature.Params().Len() != len(source.Args) {
		return nil, nil, nil,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	arguments := make([]tsgo.Expression, 0, len(source.Args))
	var requests []api.PlacementRequest
	for index, argument := range source.Args {
		argumentType := context.TypesInfo().TypeOf(argument)
		if argumentType == nil ||
			!types.AssignableTo(argumentType, signature.Params().At(index).Type()) {
			return nil, nil, nil,
				api.Unsupported(context, api.CategoryExpression, source)
		}
		target, err := children.Expression(
			context.
				WithRole(api.RoleCallArgument).
				WithExpectedType(signature.Params().At(index).Type()),
			argument,
		)
		if err != nil {
			return nil, nil, nil, err
		}
		if len(target.Before()) != 0 {
			return nil, nil, nil,
				api.Unsupported(context, api.CategoryExpression, source)
		}
		arguments = append(arguments, target.Value())
		requests = append(requests, target.Requests()...)
	}
	return arguments, nil, requests, nil
}
