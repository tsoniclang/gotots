package call

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/contracts/callableabi"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func emitArguments(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
	signature *types.Signature,
	captureAll bool,
) ([]tsgo.Expression, []tsgo.Statement, []api.RootRequest, error) {
	return emitArgumentsWithABI(
		context,
		children,
		source,
		signature,
		callableabi.Callable{},
		captureAll,
	)
}

func emitArgumentsWithABI(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
	signature *types.Signature,
	callableABI callableabi.Callable,
	captureAll bool,
) ([]tsgo.Expression, []tsgo.Statement, []api.RootRequest, error) {
	projected := callableABI.Valid()
	if len(source.Args) == 1 {
		if results, ok := context.TypesInfo().TypeOf(source.Args[0]).(*types.Tuple); ok {
			if projected {
				return nil, nil, nil,
					api.Unsupported(context, api.CategoryExpression, source)
			}
			if signature.Variadic() {
				return emitVariadicMultipleArgument(
					context,
					children,
					source,
					signature,
					results,
					captureAll,
				)
			}
			arguments, before, requests, err := emitMultipleArgument(
				context,
				children,
				source,
				signature,
				results,
			)
			if err != nil || !captureAll {
				return arguments, before, requests, err
			}
			return captureArgumentExpressions(
				context,
				arguments,
				before,
				requests,
			)
		}
	}
	if signature.Variadic() {
		if projected {
			return nil, nil, nil,
				api.Unsupported(context, api.CategoryExpression, source)
		}
		return emitVariadicArguments(
			context,
			children,
			source,
			signature,
			captureAll,
		)
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
		if selected, ok := callableABI.Parameter(index); ok &&
			selected.Projection() == callableabi.ProjectionPointeeValue {
			target, err := emitPointeeValueArgument(
				context,
				children,
				argument,
				signature.Params().At(index).Type(),
				selected,
			)
			if err != nil {
				return nil, nil, nil, err
			}
			if len(target.Before()) != 0 {
				requiresCapture = true
			}
			emissions = append(emissions, target)
			continue
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
		target, err = context.Values().Transfer(
			context.WithRole(api.RoleCallArgument),
			argument,
			argumentType,
			signature.Params().At(index).Type(),
			api.ValueTransferCopy,
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
	if requiresCapture || captureAll {
		return captureArguments(context, children, source, signature, emissions)
	}
	arguments := make([]tsgo.Expression, 0, len(emissions))
	var requests []api.RootRequest
	for _, target := range emissions {
		arguments = append(arguments, target.Value())
		requests = append(requests, target.Requests()...)
	}
	return arguments, nil, requests, nil
}

func captureArgumentExpressions(
	context api.Context,
	expressions []tsgo.Expression,
	before []tsgo.Statement,
	requests []api.RootRequest,
) ([]tsgo.Expression, []tsgo.Statement, []api.RootRequest, error) {
	arguments := make([]tsgo.Expression, 0, len(expressions))
	for _, expression := range expressions {
		temporaryName, err := context.Names().Temporary(
			api.TemporaryCallArgument,
		)
		if err != nil {
			return nil, nil, nil, err
		}
		before = append(
			before,
			context.Factory().VariableStatement(
				nil,
				context.Factory().VariableDeclarationList(
					[]tsgo.VariableDeclaration{
						context.Factory().VariableDeclaration(
							context.Factory().Identifier(temporaryName),
							nil,
							nil,
							expression,
						),
					},
					tsgo.NodeFlagsConst,
				),
			),
		)
		arguments = append(
			arguments,
			context.Factory().Identifier(temporaryName),
		)
	}
	return arguments, before, requests, nil
}

func captureArguments(
	context api.Context,
	_ api.ChildEmitter,
	_ *ast.CallExpr,
	_ *types.Signature,
	emissions []api.ExpressionEmission,
) ([]tsgo.Expression, []tsgo.Statement, []api.RootRequest, error) {
	arguments := make([]tsgo.Expression, 0, len(emissions))
	var before []tsgo.Statement
	var requests []api.RootRequest
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
