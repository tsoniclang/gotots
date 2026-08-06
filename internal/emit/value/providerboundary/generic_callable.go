package providerboundary

import (
	"go/types"
	"strconv"

	"github.com/tsoniclang/gotots/internal/emit/api"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func toProviderGenericCallable(
	context api.Context,
	children api.ChildEmitter,
	contract *types.Signature,
	concrete *types.Signature,
	model definedtype.Model,
	source api.ExpressionEmission,
) (api.ExpressionEmission, bool, error) {
	parameters := make([]tsgo.ParameterDeclaration, 0, concrete.Params().Len())
	parameterValues := make([]tsgo.Expression, 0, concrete.Params().Len())
	for index := range concrete.Params().Len() {
		name := "$providerArgument" + strconv.Itoa(index)
		parameters = append(parameters, context.Factory().ParameterDeclaration(
			nil,
			nil,
			context.Factory().Identifier(name),
			nil,
			nil,
			nil,
		))
		parameterValues = append(
			parameterValues,
			context.Factory().Identifier(name),
		)
	}
	arguments, before, requests, argumentsChanged, err := convertGenericArguments(
		context,
		children,
		contract.Params(),
		concrete.Params(),
		parameterValues,
		fromProviderGenericValue,
	)
	if err != nil {
		return api.ExpressionEmission{}, false, err
	}
	captured, err := context.Names().Temporary(api.TemporaryCallCallee)
	if err != nil {
		return api.ExpressionEmission{}, false, err
	}
	cooperative, contractRequests, err := providerCallableContract(
		context,
		concrete,
	)
	if err != nil {
		return api.ExpressionEmission{}, false, err
	}
	callValue := tsgo.Expression(context.Factory().CallExpression(
		context.Factory().Identifier(captured),
		nil,
		nil,
		arguments,
		tsgo.NodeFlagsNone,
	))
	if cooperative {
		callValue = context.Factory().AwaitExpression(callValue)
	}
	call, err := api.NewExpressionEmission(
		before,
		callValue,
		api.CombineRequests(requests, contractRequests),
	)
	if err != nil {
		return api.ExpressionEmission{}, false, err
	}
	result, resultChanged, err := toProviderGenericResults(
		context,
		children,
		contract.Results(),
		concrete.Results(),
		call,
	)
	if err != nil {
		return api.ExpressionEmission{}, false, err
	}
	if !argumentsChanged && !resultChanged {
		return source, false, nil
	}
	projected, err := projectCallable(context, model, source)
	if err != nil {
		return api.ExpressionEmission{}, false, err
	}
	var modifiers []tsgo.ModifierLike
	if cooperative {
		modifiers = []tsgo.ModifierLike{context.Factory().AsyncKeyword()}
	}
	adapter := context.Factory().ConditionalExpression(
		isUndefined(context.Factory(), captured),
		context.Factory().QuestionToken(),
		context.Factory().Identifier("undefined"),
		context.Factory().ColonToken(),
		context.Factory().ArrowFunction(
			modifiers,
			nil,
			parameters,
			nil,
			context.Factory().EqualsGreaterThanToken(),
			adapterBody(context.Factory(), concrete.Results(), result),
		),
	)
	wrapped, err := wrapCallable(
		context,
		model,
		api.DirectExpression(
			adapter,
			api.CombineRequests(
				projected.Requests(),
				result.Requests(),
			)...,
		),
	)
	if err != nil {
		return api.ExpressionEmission{}, false, err
	}
	target, err := api.NewExpressionEmission(
		append(projected.Before(), captureStatement(
			context.Factory(),
			captured,
			projected.Value(),
		)),
		wrapped.Value(),
		wrapped.Requests(),
	)
	return target, true, err
}
