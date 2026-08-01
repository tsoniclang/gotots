package call

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

func emitDeferredGeneric(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
) (api.ExpressionEmission, bool, error) {
	owner, instance, ok := genericFunctionInstance(
		context,
		source,
	)
	if !ok {
		return api.ExpressionEmission{}, false, nil
	}
	signature, ok := instance.Type.(*types.Signature)
	if !ok || signature.Recv() != nil {
		return api.ExpressionEmission{}, true,
			api.Unsupported(context, api.CategoryStatement, source)
	}
	selected, ok, err := context.ResolveGenericCallable(owner)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	if !ok ||
		instance.TypeArgs == nil ||
		instance.TypeArgs.Len() != len(selected.Parameters()) {
		return api.ExpressionEmission{}, true,
			api.Unsupported(context, api.CategoryStatement, source)
	}
	capabilities, capabilityRequests, err := genericinstance.EmitCapabilities(
		context,
		source,
		selected,
		instance.TypeArgs,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	capabilityArguments, err := genericabi.JoinCapabilities(
		owner,
		selected.Operations(),
		capabilities,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	arguments, before, argumentRequests, err := emitArguments(
		context,
		children,
		source,
		signature,
		true,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	declarationSignature, ok := owner.Type().(*types.Signature)
	if !ok {
		return api.ExpressionEmission{}, true,
			api.Unsupported(context, api.CategoryStatement, source)
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
	typeArguments, typeRequests, err :=
		genericinstance.EmitFunctionTypeArguments(
			context,
			children,
			source,
			owner,
			instance.TypeArgs,
		)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	arguments = append(capabilityArguments, arguments...)
	arguments = append(
		arguments,
		context.Factory().Identifier(callable.RecoveryAuthorityName),
	)
	control, err := api.NewDirectCallableControlRequest(
		owner,
		api.CallableControlRecovery,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	cooperative, contractRequests, err :=
		cooperativecall.GenericContract(context, callableFacet)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	call := context.Factory().CallExpression(
		reference.Expression(context.Factory()),
		nil,
		typeArguments,
		arguments,
		tsgo.NodeFlagsNone,
	)
	target, err := deferredInvocation(
		context,
		before,
		nil,
		call,
		cooperative,
		api.CombineRequests(
			reference.Requests(),
			typeRequests,
			capabilityRequests,
			argumentRequests,
			contractRequests,
			[]api.RootRequest{control},
		),
	)
	return target, true, err
}
