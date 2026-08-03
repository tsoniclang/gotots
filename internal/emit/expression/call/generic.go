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
	operationSet, ok, err := context.ResolveGenericCallable(owner)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	if !ok ||
		instance.TypeArgs.Len() != len(operationSet.Parameters()) {
		return api.ExpressionEmission{}, true,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	if err := validateResults(context, source, signature, discarded); err != nil {
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
	requiresConcretization, err :=
		context.GenericCallableRequiresConcretization(owner)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	var (
		reference     api.NameReference
		callableFacet api.CallableFacet
		typeArguments []tsgo.TypeNode
		typeRequests  []api.RootRequest
		mechanicArgs  []tsgo.Expression
		mechanicReqs  []api.RootRequest
	)
	openConcretization := requiresConcretization &&
		instance.TypeArgs.ContainsGenericTypeParameter()
	if requiresConcretization && !openConcretization {
		facet, selectionErr := cooperativecall.SelectGenericClassMethod(
			context,
			owner,
		)
		if selectionErr != nil {
			return api.ExpressionEmission{}, true, selectionErr
		}
		concrete, concreteErr := context.ResolveGenericConcretization(
			owner,
			instance.TypeArgs,
			signature,
		)
		if concreteErr != nil {
			return api.ExpressionEmission{}, true, concreteErr
		}
		reference, err = api.NewNameReference(
			concrete.Name(),
			concrete.Requests()...,
		)
		if err != nil {
			return api.ExpressionEmission{}, true, err
		}
		callableFacet = facet
	} else if openConcretization {
		facet, selectionErr := cooperativecall.SelectGenericClassMethod(
			context,
			owner,
		)
		if selectionErr != nil {
			return api.ExpressionEmission{}, true, selectionErr
		}
		kernelNames, available := context.Names().(api.GenericKernelNames)
		if !available {
			return api.ExpressionEmission{}, true, &api.ContextError{
				Reason: "generic kernel names are unavailable",
			}
		}
		reference, err = kernelNames.GenericKernel(owner)
		callableFacet = facet
		typeArguments, typeRequests, err =
			genericinstance.EmitFunctionTypeArguments(
				context,
				children,
				source,
				owner,
				instance.TypeArgs,
			)
		if err == nil {
			capabilities, capabilityRequests, capabilityErr :=
				genericinstance.EmitCapabilities(
					context,
					source,
					operationSet,
					instance.TypeArgs,
				)
			if capabilityErr != nil {
				return api.ExpressionEmission{}, true, capabilityErr
			}
			mechanicArgs, capabilityErr = genericabi.JoinCapabilities(
				owner,
				operationSet.Operations(),
				capabilities,
			)
			if capabilityErr != nil {
				return api.ExpressionEmission{}, true, capabilityErr
			}
			mechanicReqs = capabilityRequests
		}
	} else {
		if len(operationSet.Operations()) != 0 {
			return api.ExpressionEmission{}, true, &api.InvariantError{
				Role:   context.Role(),
				Reason: "generic mechanics reached a source-facing callable",
			}
		}
		selected, facet, selectionErr :=
			cooperativecall.SelectGenericCallable(context, owner)
		if selectionErr != nil {
			return api.ExpressionEmission{}, true, selectionErr
		}
		reference = selected
		callableFacet = facet
		typeArguments, typeRequests, err =
			genericinstance.EmitFunctionTypeArguments(
				context,
				children,
				source,
				owner,
				instance.TypeArgs,
			)
	}
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	arguments = append(mechanicArgs, arguments...)
	result, err := api.NewExpressionEmission(
		before,
		context.Factory().CallExpression(
			reference.Expression(context.Factory()),
			nil,
			typeArguments,
			arguments,
			tsgo.NodeFlagsNone,
		),
		api.CombineRequests(
			reference.Requests(),
			typeRequests,
			mechanicReqs,
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
) (*types.Func, api.TypeInstance, bool) {
	if source == nil {
		return nil, api.TypeInstance{}, false
	}
	return genericinstance.FunctionEvidence(context.TypesInfo(), source.Fun)
}
