package functionliteral

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/callable"
	cooperativecall "github.com/tsoniclang/gotots/internal/emit/concurrency/cooperative"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.FuncLit,
) (api.ExpressionEmission, error) {
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
	facet, err := api.NewFunctionLiteralCallableFacet(
		context.ArtifactOwner(),
		source,
	)
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
	if context.CallableControlFor(source).Recovery() {
		recovery, recoveryRequests, err :=
			callable.RecoveryAuthorityParameter(context)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		parameters = append(parameters, recovery)
		requests = append(requests, recoveryRequests...)
	}
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
	target := api.DirectExpression(
		context.Factory().FunctionExpression(
			modifiers,
			nil,
			nil,
			nil,
			parameters,
			resultType,
			body.Value(),
		),
		api.CombineRequests(
			requests,
			body.Requests(),
			observation.Requests(),
		)...,
	)
	return cooperativecall.AdaptLiteralValue(
		context,
		children,
		source,
		target,
	)
}
