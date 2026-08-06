package providerboundary

import (
	"go/types"
	"strconv"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/callable"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func ToProviderGenericArguments(
	context api.Context,
	children api.ChildEmitter,
	contract *types.Tuple,
	concrete *types.Tuple,
	arguments []tsgo.Expression,
) ([]tsgo.Expression, []tsgo.Statement, []api.RootRequest, error) {
	target, before, requests, _, err := convertGenericArguments(
		context,
		children,
		contract,
		concrete,
		arguments,
		toProviderGenericValue,
	)
	return target, before, requests, err
}

func convertGenericArguments(
	context api.Context,
	children api.ChildEmitter,
	contract *types.Tuple,
	concrete *types.Tuple,
	arguments []tsgo.Expression,
	convert genericValueConverter,
) ([]tsgo.Expression, []tsgo.Statement, []api.RootRequest, bool, error) {
	if contract == nil && concrete == nil && len(arguments) == 0 {
		return nil, nil, nil, false, nil
	}
	if contract == nil || concrete == nil ||
		contract.Len() != concrete.Len() || concrete.Len() != len(arguments) {
		return nil, nil, nil, false, boundaryInvariant(
			context,
			"provider generic argument contract is inconsistent",
		)
	}
	target := make([]tsgo.Expression, 0, len(arguments))
	var before []tsgo.Statement
	var requests []api.RootRequest
	changed := false
	for index, argument := range arguments {
		converted, selected, err := convert(
			context,
			children,
			contract.At(index).Type(),
			concrete.At(index).Type(),
			api.DirectExpression(argument),
		)
		if err != nil {
			return nil, nil, nil, false, err
		}
		changed = changed || selected
		before = append(before, converted.Before()...)
		target = append(target, converted.Value())
		requests = append(requests, converted.Requests()...)
	}
	return target, before, api.CombineRequests(requests), changed, nil
}

func FromProviderGenericResults(
	context api.Context,
	children api.ChildEmitter,
	contract *types.Tuple,
	concrete *types.Tuple,
	emission api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	return convertGenericResults(
		context,
		children,
		contract,
		concrete,
		emission,
		fromProviderGenericValue,
		false,
	)
}

func toProviderGenericResults(
	context api.Context,
	children api.ChildEmitter,
	contract *types.Tuple,
	concrete *types.Tuple,
	emission api.ExpressionEmission,
) (api.ExpressionEmission, bool, error) {
	return convertGenericResultsSelected(
		context,
		children,
		contract,
		concrete,
		emission,
		toProviderGenericValue,
		true,
	)
}

type genericValueConverter func(
	api.Context,
	api.ChildEmitter,
	types.Type,
	types.Type,
	api.ExpressionEmission,
) (api.ExpressionEmission, bool, error)

func convertGenericResults(
	context api.Context,
	children api.ChildEmitter,
	contract *types.Tuple,
	concrete *types.Tuple,
	emission api.ExpressionEmission,
	convert genericValueConverter,
	providerTarget bool,
) (api.ExpressionEmission, error) {
	target, _, err := convertGenericResultsSelected(
		context,
		children,
		contract,
		concrete,
		emission,
		convert,
		providerTarget,
	)
	return target, err
}

func convertGenericResultsSelected(
	context api.Context,
	children api.ChildEmitter,
	contract *types.Tuple,
	concrete *types.Tuple,
	emission api.ExpressionEmission,
	convert genericValueConverter,
	providerTarget bool,
) (api.ExpressionEmission, bool, error) {
	if contract == nil && concrete == nil {
		return emission, false, nil
	}
	if contract == nil || concrete == nil || contract.Len() != concrete.Len() {
		return api.ExpressionEmission{}, false, boundaryInvariant(
			context,
			"provider generic result contract is inconsistent",
		)
	}
	if concrete.Len() == 0 {
		return emission, false, nil
	}
	if concrete.Len() == 1 {
		return convert(
			context,
			children,
			contract.At(0).Type(),
			concrete.At(0).Type(),
			emission,
		)
	}
	temporary, err := context.Names().Temporary(api.TemporaryMultipleResults)
	if err != nil {
		return api.ExpressionEmission{}, false, err
	}
	before := append(emission.Before(), context.Factory().VariableStatement(
		nil,
		context.Factory().VariableDeclarationList(
			[]tsgo.VariableDeclaration{context.Factory().VariableDeclaration(
				context.Factory().Identifier(temporary),
				nil,
				nil,
				emission.Value(),
			)},
			tsgo.NodeFlagsConst,
		),
	))
	elements := make([]tsgo.Expression, 0, concrete.Len())
	requests := emission.Requests()
	changed := false
	for index := range concrete.Len() {
		value := context.Factory().ElementAccessExpression(
			context.Factory().Identifier(temporary),
			nil,
			context.Factory().NumericLiteral(strconv.Itoa(index), tsgo.TokenFlagsNone),
			tsgo.NodeFlagsNone,
		)
		converted, selected, convertErr := convert(
			context,
			children,
			contract.At(index).Type(),
			concrete.At(index).Type(),
			api.DirectExpression(value),
		)
		if convertErr != nil {
			return api.ExpressionEmission{}, false, convertErr
		}
		changed = changed || selected
		before = append(before, converted.Before()...)
		elements = append(elements, converted.Value())
		requests = append(requests, converted.Requests()...)
	}
	if !changed {
		return emission, false, nil
	}
	targetContext := context
	if providerTarget {
		targetContext, err = providerRepresentationContext(context, nil)
		if err != nil {
			return api.ExpressionEmission{}, false, err
		}
	}
	targetType, err := children.RepresentedType(
		targetContext.WithRole(api.RoleResultType),
		nil,
		concrete,
	)
	if err != nil {
		return api.ExpressionEmission{}, false, err
	}
	target, err := api.NewExpressionEmission(
		before,
		context.Factory().SatisfiesExpression(
			context.Factory().ArrayLiteralExpression(elements, false),
			targetType.Value(),
		),
		api.CombineRequests(requests, targetType.Requests()),
	)
	return target, true, err
}

func fromProviderGenericValue(
	context api.Context,
	children api.ChildEmitter,
	contractType types.Type,
	concreteType types.Type,
	value api.ExpressionEmission,
) (api.ExpressionEmission, bool, error) {
	contract, concrete, model, callablePair := genericCallablePair(
		contractType,
		concreteType,
	)
	if callablePair {
		return fromProviderGenericCallable(
			context,
			children,
			contract,
			concrete,
			model,
			value,
		)
	}
	if genericOpaque(contractType) {
		return value, false, nil
	}
	return fromProviderValueSelected(
		context,
		children,
		nil,
		"",
		nil,
		concreteType,
		value,
	)
}

func toProviderGenericValue(
	context api.Context,
	children api.ChildEmitter,
	contractType types.Type,
	concreteType types.Type,
	value api.ExpressionEmission,
) (api.ExpressionEmission, bool, error) {
	contract, concrete, model, callablePair := genericCallablePair(
		contractType,
		concreteType,
	)
	if callablePair {
		return toProviderGenericCallable(
			context,
			children,
			contract,
			concrete,
			model,
			value,
		)
	}
	if genericOpaque(contractType) {
		return value, false, nil
	}
	return toProviderValueSelected(
		context,
		children,
		nil,
		"",
		nil,
		concreteType,
		value,
	)
}

func genericOpaque(source types.Type) bool {
	return api.ContainsGenericTypeParameter(source)
}

func genericCallablePair(
	contractType types.Type,
	concreteType types.Type,
) (*types.Signature, *types.Signature, definedtype.Model, bool) {
	contract, contractOK, _ := callableType(contractType)
	concrete, concreteOK, model := callableType(concreteType)
	return contract, concrete, model, contractOK && concreteOK
}

func fromProviderGenericCallable(
	context api.Context,
	children api.ChildEmitter,
	contract *types.Signature,
	concrete *types.Signature,
	model definedtype.Model,
	source api.ExpressionEmission,
) (api.ExpressionEmission, bool, error) {
	target, err := callable.EmitABIAdapter(context, children, nil, concrete)
	if err != nil {
		return api.ExpressionEmission{}, false, err
	}
	parameters := target.SourceParameterReferences(context.Factory())
	arguments, before, requests, argumentsChanged, err := convertGenericArguments(
		context,
		children,
		contract.Params(),
		concrete.Params(),
		parameters,
		toProviderGenericValue,
	)
	if err != nil {
		return api.ExpressionEmission{}, false, err
	}
	captured, err := context.Names().Temporary(api.TemporaryCallCallee)
	if err != nil {
		return api.ExpressionEmission{}, false, err
	}
	call, err := api.NewExpressionEmission(
		before,
		context.Factory().CallExpression(
			context.Factory().Identifier(captured),
			nil,
			nil,
			arguments,
			tsgo.NodeFlagsNone,
		),
		requests,
	)
	if err != nil {
		return api.ExpressionEmission{}, false, err
	}
	result, resultChanged, err := convertGenericResultsSelected(
		context,
		children,
		contract.Results(),
		concrete.Results(),
		call,
		fromProviderGenericValue,
		false,
	)
	changed := argumentsChanged || resultChanged
	if err != nil || !changed {
		return source, false, err
	}
	projected, err := projectCallable(context, model, source)
	if err != nil {
		return api.ExpressionEmission{}, false, err
	}
	cooperative, contractRequests, err := providerCallableContract(context, concrete)
	if err != nil {
		return api.ExpressionEmission{}, false, err
	}
	resultType := target.Result()
	var modifiers []tsgo.ModifierLike
	if cooperative {
		modifiers = []tsgo.ModifierLike{context.Factory().AsyncKeyword()}
		resultType = callable.PromiseResult(context.Factory(), resultType)
	}
	adapter := context.Factory().ConditionalExpression(
		isUndefined(context.Factory(), captured),
		context.Factory().QuestionToken(),
		context.Factory().Identifier("undefined"),
		context.Factory().ColonToken(),
		context.Factory().ArrowFunction(
			modifiers,
			nil,
			target.Parameters(),
			resultType,
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
				target.Requests(),
				result.Requests(),
				contractRequests,
			)...,
		),
	)
	if err != nil {
		return api.ExpressionEmission{}, false, err
	}
	targetEmission, err := api.NewExpressionEmission(
		append(projected.Before(), captureStatement(
			context.Factory(),
			captured,
			projected.Value(),
		)),
		wrapped.Value(),
		wrapped.Requests(),
	)
	return targetEmission, err == nil, err
}
