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
	if source.Ellipsis != token.NoPos {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	if selector, method, selection, ok := selectedMethod(
		context.TypesInfo(),
		source.Fun,
	); ok {
		return emitMethod(
			context,
			children,
			source,
			selector,
			method,
			selection,
			discarded,
		)
	}
	object, ok := calleeObject(context.TypesInfo(), source.Fun)
	if !ok {
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
	if err := validateResults(context, source, signature, discarded); err != nil {
		return api.ExpressionEmission{}, err
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

func validateResults(
	context api.Context,
	source *ast.CallExpr,
	signature *types.Signature,
	discarded bool,
) error {
	resultCount := 0
	if signature.Results() != nil {
		resultCount = signature.Results().Len()
	}
	if discarded {
		return nil
	}
	switch {
	case resultCount == 0:
		return api.Unsupported(context, api.CategoryExpression, source)
	case resultCount == 1 && context.ExpectedResults() != nil:
		return api.Unsupported(context, api.CategoryExpression, source)
	case resultCount > 1 &&
		(context.ExpectedResults() == nil ||
			!types.Identical(signature.Results(), context.ExpectedResults())):
		return api.Unsupported(context, api.CategoryExpression, source)
	default:
		return nil
	}
}

func calleeObject(info *types.Info, source ast.Expr) (*types.Func, bool) {
	if info == nil {
		return nil, false
	}
	switch source := source.(type) {
	case *ast.Ident:
		object, ok := info.Uses[source].(*types.Func)
		return object, ok
	case *ast.SelectorExpr:
		if info.Selections[source] != nil {
			return nil, false
		}
		qualifier, ok := source.X.(*ast.Ident)
		if !ok {
			return nil, false
		}
		packageName, ok := info.Uses[qualifier].(*types.PkgName)
		if !ok {
			return nil, false
		}
		object, ok := info.Uses[source.Sel].(*types.Func)
		if !ok || object.Pkg() != packageName.Imported() {
			return nil, false
		}
		return object, true
	default:
		return nil, false
	}
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
	emissions := make([]api.ExpressionEmission, 0, len(source.Args))
	requiresCapture := false
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
		target, err = context.Values().Copy(
			context.WithRole(api.RoleCallArgument),
			argument,
			signature.Params().At(index).Type(),
			target,
		)
		if err != nil {
			return nil, nil, nil, err
		}
		if len(target.Before()) != 0 {
			requiresCapture = true
		}
		emissions = append(emissions, target)
	}
	if requiresCapture {
		return captureArguments(context, children, source, signature, emissions)
	}
	arguments := make([]tsgo.Expression, 0, len(emissions))
	var requests []api.PlacementRequest
	for _, target := range emissions {
		arguments = append(arguments, target.Value())
		requests = append(requests, target.Requests()...)
	}
	return arguments, nil, requests, nil
}

func captureArguments(
	context api.Context,
	_ api.ChildEmitter,
	_ *ast.CallExpr,
	_ *types.Signature,
	emissions []api.ExpressionEmission,
) ([]tsgo.Expression, []tsgo.Statement, []api.PlacementRequest, error) {
	arguments := make([]tsgo.Expression, 0, len(emissions))
	var before []tsgo.Statement
	var requests []api.PlacementRequest
	for _, emission := range emissions {
		temporaryName, err := context.Names().Temporary(api.TemporaryCallArgument)
		if err != nil {
			return nil, nil, nil, err
		}
		declaration := context.Factory().VariableDeclaration(
			context.Factory().Identifier(temporaryName),
			nil,
			nil,
			emission.Value(),
		)
		before = append(before, emission.Before()...)
		before = append(
			before,
			context.Factory().VariableStatement(
				nil,
				context.Factory().VariableDeclarationList(
					[]tsgo.VariableDeclaration{declaration},
					tsgo.NodeFlagsConst,
				),
			),
		)
		arguments = append(
			arguments,
			context.Factory().Identifier(temporaryName),
		)
		requests = append(requests, emission.Requests()...)
	}
	return arguments, before, requests, nil
}
