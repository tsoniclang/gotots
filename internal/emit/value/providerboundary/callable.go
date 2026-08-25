package providerboundary

import (
	"go/ast"
	"go/types"
	"strconv"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/callable"
	cooperativecall "github.com/tsoniclang/gotots/internal/emit/concurrency/cooperative"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func FromProviderSourceCallable(
	context api.Context,
	children api.ChildEmitter,
	source ast.Expr,
	function *types.Func,
	target api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	if err := RequireSynchronousCallable(context, function); err != nil {
		return api.ExpressionEmission{}, err
	}
	if context.SynchronousCallableBoundary() {
		signature, supported, model := callableType(function.Type())
		if !supported {
			return api.ExpressionEmission{}, boundaryInvariant(
				context,
				"synchronous provider callable is unsupported",
			)
		}
		converted, _, err := fromProviderCallableSelectedWithEffect(
			context,
			children,
			nil,
			"",
			nil,
			signature,
			model,
			target,
			true,
		)
		return converted, err
	}
	converted, changed, err := FromProviderValue(
		context,
		children,
		nil,
		"",
		function.Type(),
		target,
	)
	if err != nil || changed {
		return converted, err
	}
	return cooperativecall.TransportSourceValue(
		context,
		source,
		function,
		target,
	)
}

func callableType(
	sourceType types.Type,
) (*types.Signature, bool, definedtype.Model) {
	if model, ok := definedtype.ResolveCallable(sourceType); ok {
		signature, valid := model.Callable()
		return signature, valid, model
	}
	signature, ok := types.Unalias(sourceType).(*types.Signature)
	return signature, ok && callable.Supports(signature), definedtype.Model{}
}

func fromProviderCallable(
	context api.Context,
	children api.ChildEmitter,
	owner *types.Named,
	ownerBridge string,
	signature *types.Signature,
	model definedtype.Model,
	source api.ExpressionEmission,
) (api.ExpressionEmission, bool, error) {
	return fromProviderCallableSelected(
		context,
		children,
		owner,
		ownerBridge,
		nil,
		signature,
		model,
		source,
	)
}

func fromProviderCallableSelected(
	context api.Context,
	children api.ChildEmitter,
	owner *types.Named,
	ownerBridge string,
	profile []gostdlib.ProviderCallableProfileInterface,
	signature *types.Signature,
	model definedtype.Model,
	source api.ExpressionEmission,
) (api.ExpressionEmission, bool, error) {
	return fromProviderCallableSelectedWithEffect(
		context,
		children,
		owner,
		ownerBridge,
		profile,
		signature,
		model,
		source,
		false,
	)
}

func fromProviderCallableSelectedWithEffect(
	context api.Context,
	children api.ChildEmitter,
	owner *types.Named,
	ownerBridge string,
	profile []gostdlib.ProviderCallableProfileInterface,
	signature *types.Signature,
	model definedtype.Model,
	source api.ExpressionEmission,
	synchronous bool,
) (api.ExpressionEmission, bool, error) {
	if len(profile) == 0 {
		if err := RequireProviderDefinedCallableOutput(context, model); err != nil {
			return api.ExpressionEmission{}, false, err
		}
	}
	target, err := callable.EmitABIAdapter(context, children, nil, signature)
	if err != nil {
		return api.ExpressionEmission{}, false, err
	}
	parameters := target.SourceParameterReferences(context.Factory())
	if len(parameters) != signature.Params().Len() {
		return api.ExpressionEmission{}, false, boundaryInvariant(
			context,
			"canonical callable parameter count drifted",
		)
	}
	arguments := make([]tsgo.Expression, 0, len(parameters))
	var argumentBefore []tsgo.Statement
	var argumentRequests []api.RootRequest
	changed := false
	for index, parameter := range parameters {
		converted, selected, convertErr := toProviderValueSelected(
			context,
			children,
			owner,
			ownerBridge,
			profile,
			signature.Params().At(index).Type(),
			api.DirectExpression(parameter),
		)
		if convertErr != nil {
			return api.ExpressionEmission{}, false, convertErr
		}
		changed = changed || selected
		argumentBefore = append(argumentBefore, converted.Before()...)
		arguments = append(arguments, converted.Value())
		argumentRequests = append(argumentRequests, converted.Requests()...)
	}
	captured, err := context.Names().Temporary(api.TemporaryCallCallee)
	if err != nil {
		return api.ExpressionEmission{}, false, err
	}
	call, err := api.NewExpressionEmission(
		argumentBefore,
		context.Factory().CallExpression(
			context.Factory().Identifier(captured),
			nil,
			nil,
			arguments,
			tsgo.NodeFlagsNone,
		),
		api.CombineRequests(argumentRequests),
	)
	if err != nil {
		return api.ExpressionEmission{}, false, err
	}
	result, resultChanged, err := fromProviderResultsSelected(
		context,
		children,
		owner,
		ownerBridge,
		nil,
		profile,
		signature.Results(),
		call,
	)
	if err != nil {
		return api.ExpressionEmission{}, false, err
	}
	changed = changed || resultChanged
	if !changed {
		return source, false, nil
	}
	projected, err := projectCallable(context, model, source)
	if err != nil {
		return api.ExpressionEmission{}, false, err
	}
	cooperative := false
	var contractRequests []api.RootRequest
	if !synchronous {
		var contractErr error
		cooperative, contractRequests, contractErr = providerCallableContract(
			context,
			signature,
		)
		if contractErr != nil {
			return api.ExpressionEmission{}, false, contractErr
		}
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
			adapterBody(context.Factory(), signature.Results(), result),
		),
	)
	requests := api.CombineRequests(
		projected.Requests(),
		target.Requests(),
		result.Requests(),
		contractRequests,
	)
	wrapped, err := wrapCallable(
		context,
		model,
		api.DirectExpression(adapter, requests...),
	)
	if err != nil {
		return api.ExpressionEmission{}, false, err
	}
	before := append(projected.Before(), captureStatement(
		context.Factory(),
		captured,
		projected.Value(),
	))
	targetEmission, err := api.NewExpressionEmission(
		before,
		wrapped.Value(),
		wrapped.Requests(),
	)
	return targetEmission, true, err
}

func toProviderCallable(
	context api.Context,
	children api.ChildEmitter,
	owner *types.Named,
	ownerBridge string,
	signature *types.Signature,
	model definedtype.Model,
	source api.ExpressionEmission,
) (api.ExpressionEmission, bool, error) {
	return toProviderCallableSelected(
		context,
		children,
		owner,
		ownerBridge,
		nil,
		signature,
		model,
		source,
	)
}

func toProviderCallableSelected(
	context api.Context,
	children api.ChildEmitter,
	owner *types.Named,
	ownerBridge string,
	profile []gostdlib.ProviderCallableProfileInterface,
	signature *types.Signature,
	model definedtype.Model,
	source api.ExpressionEmission,
) (api.ExpressionEmission, bool, error) {
	if len(profile) == 0 {
		if err := RequireProviderDefinedCallableInput(
			context,
			model,
			context.SynchronousCallableBoundary(),
		); err != nil {
			return api.ExpressionEmission{}, false, err
		}
	}
	parameters := make([]tsgo.ParameterDeclaration, 0, signature.Params().Len())
	arguments := make([]tsgo.Expression, 0, signature.Params().Len())
	var argumentBefore []tsgo.Statement
	var argumentRequests []api.RootRequest
	changed := false
	for index := range signature.Params().Len() {
		name := "$providerArgument" + strconv.Itoa(index)
		parameters = append(parameters, context.Factory().ParameterDeclaration(
			nil,
			nil,
			context.Factory().Identifier(name),
			nil,
			nil,
			nil,
		))
		converted, selected, err := fromProviderValueSelected(
			context,
			children,
			owner,
			ownerBridge,
			profile,
			signature.Params().At(index).Type(),
			api.DirectExpression(context.Factory().Identifier(name)),
		)
		if err != nil {
			return api.ExpressionEmission{}, false, err
		}
		changed = changed || selected
		argumentBefore = append(argumentBefore, converted.Before()...)
		arguments = append(arguments, converted.Value())
		argumentRequests = append(argumentRequests, converted.Requests()...)
	}
	captured, err := context.Names().Temporary(api.TemporaryCallCallee)
	if err != nil {
		return api.ExpressionEmission{}, false, err
	}
	cooperative, contractRequests, err := providerCallableContract(
		context,
		signature,
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
		argumentBefore,
		callValue,
		api.CombineRequests(argumentRequests, contractRequests),
	)
	if err != nil {
		return api.ExpressionEmission{}, false, err
	}
	result, resultChanged, err := toProviderResults(
		context,
		children,
		owner,
		ownerBridge,
		signature.Results(),
		call,
	)
	if err != nil {
		return api.ExpressionEmission{}, false, err
	}
	changed = changed || resultChanged
	if !changed {
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
			adapterBody(context.Factory(), signature.Results(), result),
		),
	)
	requests := api.CombineRequests(projected.Requests(), result.Requests())
	wrapped, err := wrapCallable(
		context,
		model,
		api.DirectExpression(adapter, requests...),
	)
	if err != nil {
		return api.ExpressionEmission{}, false, err
	}
	before := append(projected.Before(), captureStatement(
		context.Factory(),
		captured,
		projected.Value(),
	))
	target, err := api.NewExpressionEmission(
		before,
		wrapped.Value(),
		wrapped.Requests(),
	)
	return target, true, err
}

func projectCallable(
	context api.Context,
	model definedtype.Model,
	source api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	if model.Type() == nil {
		return source, nil
	}
	return model.Project(context, source)
}

func wrapCallable(
	context api.Context,
	model definedtype.Model,
	source api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	if model.Type() == nil {
		return source, nil
	}
	return model.Wrap(context, source)
}

func adapterBody(
	factory tsgo.Factory,
	results *types.Tuple,
	emission api.ExpressionEmission,
) tsgo.ConciseBody {
	statements := emission.Before()
	if results == nil || results.Len() == 0 {
		statements = append(
			statements,
			factory.ExpressionStatement(emission.Value()),
		)
	} else {
		statements = append(statements, factory.ReturnStatement(emission.Value()))
	}
	return factory.Block(statements, true)
}

func captureStatement(
	factory tsgo.Factory,
	name string,
	value tsgo.Expression,
) tsgo.Statement {
	return factory.VariableStatement(
		nil,
		factory.VariableDeclarationList(
			[]tsgo.VariableDeclaration{
				factory.VariableDeclaration(
					factory.Identifier(name),
					nil,
					nil,
					value,
				),
			},
			tsgo.NodeFlagsConst,
		),
	)
}

func isUndefined(factory tsgo.Factory, name string) tsgo.Expression {
	return factory.BinaryExpression(
		nil,
		factory.Identifier(name),
		nil,
		factory.BinaryOperatorToken(
			tsgo.BinaryOperatorEqualsEqualsEqualsToken,
		),
		factory.Identifier("undefined"),
	)
}

func boundaryInvariant(context api.Context, reason string) error {
	return &api.InvariantError{Role: context.Role(), Reason: reason}
}

func providerCallableContract(
	context api.Context,
	signature *types.Signature,
) (bool, []api.RootRequest, error) {
	_, requests, err := cooperativecall.ValueContract(context, signature)
	return context.ConcurrencySemantics() == api.ConcurrencySemanticsCooperative,
		requests, err
}
