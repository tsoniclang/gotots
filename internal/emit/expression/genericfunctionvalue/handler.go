package genericfunctionvalue

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/callable"
	cooperativecall "github.com/tsoniclang/gotots/internal/emit/concurrency/cooperative"
	genericabi "github.com/tsoniclang/gotots/internal/emit/generic/abi"
	genericinstance "github.com/tsoniclang/gotots/internal/emit/generic/instance"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source ast.Expr,
) (api.ExpressionEmission, bool, error) {
	switch source.(type) {
	case *ast.IndexExpr, *ast.IndexListExpr:
	default:
		return api.ExpressionEmission{}, false, nil
	}
	owner, instance, ok := genericinstance.FunctionEvidence(
		context.TypesInfo(),
		source,
	)
	if !ok {
		return api.ExpressionEmission{}, false, nil
	}
	signature, ok := instance.Type.(*types.Signature)
	if !ok ||
		signature.Recv() != nil ||
		!callable.Supports(signature) ||
		!types.Identical(context.TypesInfo().TypeOf(source), signature) {
		return api.ExpressionEmission{}, true,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	operationSet, resolved, err := context.ResolveGenericCallable(owner)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	if !resolved ||
		instance.TypeArgs == nil ||
		instance.TypeArgs.Len() != len(operationSet.Parameters()) {
		return api.ExpressionEmission{}, true,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	declarationSignature, ok := owner.Type().(*types.Signature)
	if !ok {
		return api.ExpressionEmission{}, true,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	reference, callableFacet, _, err :=
		cooperativecall.SelectGenericCallable(
			context,
			owner,
			declarationSignature,
			signature,
		)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	_, abiCooperative, contractRequests, err :=
		cooperativecall.GenericValueContract(
			context,
			callableFacet,
			signature,
		)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	target, err := callable.EmitABIAdapter(
		context,
		children,
		source,
		signature,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	typeArguments, typeRequests, err := genericinstance.EmitTypeArguments(
		context,
		children,
		source,
		owner,
		instance.TypeArgs,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	capabilities, capabilityRequests, err := genericinstance.EmitCapabilities(
		context,
		source,
		operationSet,
		instance.TypeArgs,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	capabilityArguments, err := genericabi.JoinCapabilities(
		owner,
		operationSet.Operations(),
		capabilities,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	recoveryAuthority, recoveryOK :=
		target.RecoveryAuthorityReference(context.Factory())
	if !recoveryOK {
		return api.ExpressionEmission{}, true, &api.InvariantError{
			Role:   context.Role(),
			Reason: "generic function value lacks recovery authority",
		}
	}
	arguments := append(
		capabilityArguments,
		target.SourceParameterReferences(context.Factory())...,
	)
	arguments = append(arguments, recoveryAuthority)
	controlRequest, err := api.NewDirectCallableControlRequest(
		owner,
		api.CallableControlRecovery,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	var modifiers []tsgo.ModifierLike
	resultType := target.Result()
	if abiCooperative {
		modifiers = []tsgo.ModifierLike{
			context.Factory().AsyncKeyword(),
		}
		resultType = callable.PromiseResult(
			context.Factory(),
			resultType,
		)
	}
	return api.DirectExpression(
		context.Factory().ArrowFunction(
			modifiers,
			nil,
			target.Parameters(),
			resultType,
			context.Factory().EqualsGreaterThanToken(),
			context.Factory().CallExpression(
				context.Factory().Identifier(reference.Name()),
				nil,
				typeArguments,
				arguments,
				tsgo.NodeFlagsNone,
			),
		),
		api.CombineRequests(
			target.Requests(),
			typeRequests,
			capabilityRequests,
			reference.Requests(),
			contractRequests,
			[]api.RootRequest{controlRequest},
		)...,
	), true, nil
}
