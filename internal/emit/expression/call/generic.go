package call

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	cooperativecall "github.com/tsoniclang/gotots/internal/emit/concurrency/cooperative"
	genericabi "github.com/tsoniclang/gotots/internal/emit/generic/abi"
	genericinstance "github.com/tsoniclang/gotots/internal/emit/generic/instance"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func emitGeneric(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
	discarded bool,
	detached bool,
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
			api.Unsupported(context, api.CategoryExpression, source)
	}
	callable, ok, err := context.ResolveGenericCallable(owner)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	if !ok ||
		instance.TypeArgs == nil ||
		instance.TypeArgs.Len() != len(callable.Parameters()) {
		return api.ExpressionEmission{}, true,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	if err := validateResults(context, source, signature, discarded); err != nil {
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
		callable,
		instance.TypeArgs,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	capabilityArguments, err := genericabi.JoinCapabilities(
		owner,
		callable.Operations(),
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
		detached,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
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
	arguments = append(capabilityArguments, arguments...)
	result, err := api.NewExpressionEmission(
		before,
		context.Factory().CallExpression(
			context.Factory().Identifier(reference.Name()),
			nil,
			typeArguments,
			arguments,
			tsgo.NodeFlagsNone,
		),
		api.CombineRequests(
			reference.Requests(),
			typeRequests,
			capabilityRequests,
			argumentRequests,
		),
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	if detached {
		result, err = cooperativecall.DetachedGenericCall(
			context,
			source,
			callableFacet,
			result,
		)
	} else {
		result, err = cooperativecall.GenericCall(
			context,
			source,
			callableFacet,
			result,
		)
	}
	return result, true, err
}

func genericFunctionInstance(
	context api.Context,
	source *ast.CallExpr,
) (*types.Func, types.Instance, bool) {
	if source == nil {
		return nil, types.Instance{}, false
	}
	return genericinstance.FunctionEvidence(context.TypesInfo(), source.Fun)
}
