package call

import (
	"go/ast"
	"go/types"

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
	return emitArgumentsWithProviderCallableParameters(
		context,
		children,
		source,
		signature,
		captureAll,
		nil,
	)
}

func emitArgumentsWithProviderCallableParameters(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
	signature *types.Signature,
	captureAll bool,
	providerCallableParameters []int,
) ([]tsgo.Expression, []tsgo.Statement, []api.RootRequest, error) {
	selectedProviderCallables := make(map[int]struct{}, len(providerCallableParameters))
	for _, index := range providerCallableParameters {
		if index < 0 || index >= signature.Params().Len() {
			return nil, nil, nil, &api.InvariantError{
				Role:   context.Role(),
				Reason: "provider callable parameter index is invalid",
			}
		}
		selectedProviderCallables[index] = struct{}{}
	}
	if len(source.Args) == 1 {
		if results, ok := context.TypesInfo().TypeOf(source.Args[0]).(*types.Tuple); ok {
			if len(selectedProviderCallables) != 0 {
				return nil, nil, nil, &api.InvariantError{
					Role:   context.Role(),
					Reason: "provider callable parameter came from multiple results",
				}
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
		if len(selectedProviderCallables) != 0 {
			return nil, nil, nil, &api.InvariantError{
				Role:   context.Role(),
				Reason: "provider callable parameter belongs to a variadic call",
			}
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
		argumentContext := context.
			WithRole(api.RoleCallArgument).
			WithExpectedType(signature.Params().At(index).Type())
		if _, selected := selectedProviderCallables[index]; selected {
			argumentContext = argumentContext.WithProviderCallableBoundary()
		}
		target, err := children.Expression(argumentContext, argument)
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
