package functionliteral

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/callable"
	cooperativecall "github.com/tsoniclang/gotots/internal/emit/concurrency/cooperative"
	"github.com/tsoniclang/gotots/internal/emit/deferredregistry"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.FuncLit,
) (api.ExpressionEmission, error) {
	context, staticallySelected := context.TakeStaticallySelectedCallable()
	context, deferredSelected := context.TakeDeferredCallableSelection()
	if source == nil {
		return api.ExpressionEmission{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "function literal is nil",
		}
	}
	if source.Type == nil || source.Body == nil {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	signature, ok := types.Unalias(
		context.TypesInfo().TypeOf(source),
	).(*types.Signature)
	if !ok || signature.Recv() != nil {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	facet, err := context.FunctionLiteralCallableFacet(source)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	observation, err := context.ObserveCooperativeCallable(facet)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	context = context.WithCooperativeCallable(
		facet,
		observation.Cooperative(),
	)
	targetSignature, err := callable.Emit(
		context,
		children,
		source.Type,
		signature,
		api.RoleCallableParameter,
		api.RoleCallableResult,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	parameters := targetSignature.Parameters()
	requests := targetSignature.Requests()
	body, err := callable.EmitBody(
		context,
		children,
		source,
		source.Type,
		source.Body,
		signature,
		api.RoleFunctionLiteralBody,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	var modifiers []tsgo.ModifierLike
	resultType := targetSignature.Result()
	if observation.Cooperative() {
		modifiers = []tsgo.ModifierLike{context.Factory().AsyncKeyword()}
		resultType = callable.PromiseResult(context.Factory(), resultType)
	}
	ordinary := api.DirectExpression(
		context.Factory().ArrowFunction(
			modifiers,
			nil,
			parameters,
			resultType,
			context.Factory().EqualsGreaterThanToken(),
			body.Value(),
		),
		api.CombineRequests(
			requests,
			body.Requests(),
			observation.Requests(),
		)...,
	)
	if deferredSelected {
		if !context.CallableControlFor(source).Recovery() {
			return ordinary, nil
		}
		return emitDeferredLiteral(
			context,
			children,
			source,
			signature,
			targetSignature,
			observation.Cooperative(),
			false,
		)
	}
	if staticallySelected {
		return ordinary, nil
	}
	ordinary, err = cooperativecall.AdaptLiteralValue(
		context,
		children,
		source,
		ordinary,
	)
	if err != nil || !context.CallableControlFor(source).Recovery() {
		return ordinary, err
	}
	deferred, err := emitDeferredLiteral(
		context,
		children,
		source,
		signature,
		targetSignature,
		observation.Cooperative(),
		true,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	registry, err := deferredregistry.Reference(context, source, signature)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.NewExpressionEmission(
		append(ordinary.Before(), deferred.Before()...),
		context.Factory().CallExpression(
			context.Factory().PropertyAccessExpression(
				registry.Expression(context.Factory()),
				nil,
				context.Factory().Identifier(
					api.DeferredRegistryRegisterName,
				),
				tsgo.NodeFlagsNone,
			),
			nil,
			nil,
			[]tsgo.Expression{ordinary.Value(), deferred.Value()},
			tsgo.NodeFlagsNone,
		),
		api.CombineRequests(
			ordinary.Requests(),
			deferred.Requests(),
			registry.Requests(),
		),
	)
}

func emitDeferredLiteral(
	context api.Context,
	children api.ChildEmitter,
	source *ast.FuncLit,
	signature *types.Signature,
	targetSignature callable.SignatureEmission,
	providerCooperative bool,
	transported bool,
) (api.ExpressionEmission, error) {
	body, err := callable.EmitBody(
		context.WithRecoveryAuthority(callable.RecoveryAuthorityName),
		children,
		source,
		source.Type,
		source.Body,
		signature,
		api.RoleFunctionLiteralBody,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	recovery, recoveryRequests, err :=
		callable.RecoveryAuthorityParameter(context)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	abiCooperative := false
	var abiRequests []api.RootRequest
	if transported {
		abiCooperative, abiRequests, err =
			cooperativecall.ValueContract(context, signature)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
	}
	cooperative := providerCooperative || abiCooperative
	var modifiers []tsgo.ModifierLike
	resultType := targetSignature.Result()
	if cooperative {
		modifiers = []tsgo.ModifierLike{context.Factory().AsyncKeyword()}
		resultType = callable.PromiseResult(context.Factory(), resultType)
	}
	return api.DirectExpression(
		context.Factory().ArrowFunction(
			modifiers,
			nil,
			append(
				[]tsgo.ParameterDeclaration{recovery},
				targetSignature.Parameters()...,
			),
			resultType,
			context.Factory().EqualsGreaterThanToken(),
			body.Value(),
		),
		api.CombineRequests(
			targetSignature.Requests(),
			body.Requests(),
			recoveryRequests,
			abiRequests,
		)...,
	), nil
}
