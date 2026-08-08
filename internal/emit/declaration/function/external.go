package function

import (
	"go/types"

	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/callable"
	panicruntime "github.com/tsoniclang/gotots/internal/emit/runtime/panic"
	"github.com/tsoniclang/gotots/internal/emit/value/providerboundary"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func emitExternalBody(
	context api.Context,
	children api.ChildEmitter,
	function *types.Func,
	signature callable.SignatureEmission,
) (api.BlockEmission, error) {
	target, linked, err := context.ResolveExternalFunction(function)
	if err != nil {
		return api.BlockEmission{}, err
	}
	if linked {
		return EmitLinkedExternalBody(
			context,
			children,
			function,
			target,
			signature,
		)
	}
	contract, err := environmentcontract.Describe(function)
	if err != nil {
		return api.BlockEmission{}, err
	}
	panicReference, err := context.Names().Runtime(
		api.RuntimePanic,
		api.ImportPhaseValue,
	)
	if err != nil {
		return api.BlockEmission{}, err
	}
	return api.DirectBlock(
		context.Factory().Block(
			[]tsgo.Statement{
				context.Factory().ReturnStatement(
					panicruntime.Call(
						context.Factory(),
						panicReference.Name(),
						context.Factory().StringLiteral(
							"unresolved external Go function "+contract.Identity(),
							tsgo.TokenFlagsNone,
						),
					),
				),
			},
			true,
		),
		panicReference.Requests()...,
	), nil
}

func EmitLinkedExternalBody(
	context api.Context,
	children api.ChildEmitter,
	function *types.Func,
	target api.ExternalFunctionTarget,
	signature callable.SignatureEmission,
) (api.BlockEmission, error) {
	var reference api.NameReference
	var err error
	switch target.Kind() {
	case api.ExternalFunctionTargetModule:
		module, export, ok := target.Module()
		if !ok {
			return api.BlockEmission{}, &api.InvariantError{
				Role:   context.Role(),
				Reason: "external module target is invalid",
			}
		}
		reference, err = context.Names().ExternalProviderFunction(module, export)
	case api.ExternalFunctionTargetSource:
		implementation, ok := target.Source()
		if !ok {
			return api.BlockEmission{}, &api.InvariantError{
				Role:   context.Role(),
				Reason: "external source target is invalid",
			}
		}
		reference, err = context.Names().Reference(implementation)
	default:
		return api.BlockEmission{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "external target kind is invalid",
		}
	}
	if err != nil {
		return api.BlockEmission{}, err
	}
	arguments := signature.SourceParameterReferences(context.Factory())
	var before []tsgo.Statement
	var argumentRequests []api.RootRequest
	if target.Kind() == api.ExternalFunctionTargetModule {
		sourceSignature, ok := function.Type().(*types.Signature)
		if !ok {
			return api.BlockEmission{}, &api.InvariantError{
				Role:   context.Role(),
				Reason: "external function signature is invalid",
			}
		}
		arguments, before, argumentRequests, err =
			providerboundary.ToProviderArguments(
				context,
				children,
				sourceSignature.Params(),
				arguments,
			)
		if err != nil {
			return api.BlockEmission{}, err
		}
	}
	call := context.Factory().CallExpression(
		reference.Expression(context.Factory()),
		nil,
		nil,
		arguments,
		tsgo.NodeFlagsNone,
	)
	emission, err := api.NewExpressionEmission(
		before,
		call,
		api.CombineRequests(reference.Requests(), argumentRequests),
	)
	if err != nil {
		return api.BlockEmission{}, err
	}
	if target.Kind() == api.ExternalFunctionTargetModule {
		sourceSignature := function.Type().(*types.Signature)
		emission, err = providerboundary.FromProviderResults(
			context,
			children,
			nil,
			"",
			sourceSignature.Results(),
			emission,
		)
		if err != nil {
			return api.BlockEmission{}, err
		}
	}
	statements := append(
		emission.Before(),
		context.Factory().ReturnStatement(emission.Value()),
	)
	return api.DirectBlock(
		context.Factory().Block(statements, true),
		emission.Requests()...,
	), nil
}
