package call

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/callable"
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
	reference, err := context.Names().Reference(owner)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	typeArguments, typeRequests, err := genericinstance.EmitTypeArguments(
		context,
		children,
		source,
		instance.TypeArgs,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
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
	call := context.Factory().CallExpression(
		context.Factory().Identifier(reference.Name()),
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
		api.CombineRequests(
			reference.Requests(),
			typeRequests,
			capabilityRequests,
			argumentRequests,
			[]api.RootRequest{control},
		),
	)
	return target, true, err
}
