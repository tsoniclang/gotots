package call

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/callable"
	genericabi "github.com/tsoniclang/gotots/internal/emit/generic/abi"
	genericparameters "github.com/tsoniclang/gotots/internal/emit/generic/callableparameters"
	genericinstance "github.com/tsoniclang/gotots/internal/emit/generic/instance"
	providerboundary "github.com/tsoniclang/gotots/internal/emit/value/providerboundary"
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
	operationSet, ok, err := context.ResolveGenericCallable(owner)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	contract, ok := owner.Type().(*types.Signature)
	if !ok {
		return api.ExpressionEmission{}, true, &api.InvariantError{
			Role:   context.Role(),
			Reason: "generic deferred owner has no signature",
		}
	}
	if !ok ||
		instance.TypeArgs.Len() != len(operationSet.Parameters()) {
		return api.ExpressionEmission{}, true,
			api.Unsupported(context, api.CategoryStatement, source)
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
	_, callableFacet, err := callable.SelectGeneric(
		context,
		owner,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	requiresConcretization, err :=
		context.GenericCallableRequiresConcretization(owner)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	openConcretization := requiresConcretization &&
		instance.TypeArgs.ContainsGenericTypeParameter()
	parameterSelection, err := genericparameters.ForCallable(context, owner)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	providerParameters := parameterSelection.Parameters()
	providerKernel := parameterSelection.ProviderKernel()
	var (
		ordinaryReference      api.NameReference
		deferredReference      api.NameReference
		deferredTarget         api.DeferredGenericCallableReference
		typeArguments          []tsgo.TypeNode
		typeRequests           []api.RootRequest
		capabilityRequests     []api.RootRequest
		concreteRequests       []api.RootRequest
		mechanicArgs           []tsgo.Expression
		deferredTargetSelected bool
	)
	switch {
	case requiresConcretization && !openConcretization:
		concretization, concreteErr := context.ResolveGenericConcretization(
			owner,
			instance.TypeArgs,
			signature,
		)
		if concreteErr != nil {
			return api.ExpressionEmission{}, true, concreteErr
		}
		concreteRequests = concretization.Requests()
		concretizationNames, available :=
			context.Names().(api.GenericConcretizationNames)
		if !available {
			return api.ExpressionEmission{}, true, &api.ContextError{
				Reason: "generic concretization names are unavailable",
			}
		}
		ordinaryReference, err = api.NewNameReference(
			concretization.Name(),
			concretization.Requests()...,
		)
		if err == nil {
			deferredReference, err =
				concretizationNames.DeferredGenericConcretization(
					concretization.Concretization(),
				)
		}
	case openConcretization:
		kernelNames, available := context.Names().(api.GenericKernelNames)
		if !available {
			return api.ExpressionEmission{}, true, &api.ContextError{
				Reason: "generic kernel names are unavailable",
			}
		}
		ordinaryReference, err = kernelNames.GenericKernel(owner)
		if err == nil {
			deferredTarget, err =
				kernelNames.DeferredGenericKernel(owner)
			if err == nil {
				deferredReference = deferredTarget.Reference()
				deferredTargetSelected = true
			}
		}
		if err == nil {
			typeArguments, typeRequests, err =
				genericinstance.EmitFunctionTypeArguments(
					context,
					children,
					source,
					owner,
					instance.TypeArgs,
				)
		}
		if err == nil {
			var capabilities []genericabi.Binding[tsgo.Expression]
			capabilities, capabilityRequests, err =
				genericinstance.EmitCapabilities(
					context,
					children,
					source,
					operationSet,
					instance.TypeArgs,
				)
			if err == nil {
				mechanicArgs, err = genericabi.JoinCapabilities(
					owner,
					operationSet.Operations(),
					capabilities,
				)
			}
		}
	default:
		if len(operationSet.Operations()) != 0 {
			return api.ExpressionEmission{}, true, &api.InvariantError{
				Role:   context.Role(),
				Reason: "generic mechanics reached a source-facing deferred call",
			}
		}
		if providerKernel {
			kernelNames, available := context.Names().(api.GenericKernelNames)
			if !available {
				return api.ExpressionEmission{}, true, &api.ContextError{
					Reason: "generic kernel names are unavailable",
				}
			}
			ordinaryReference, err = kernelNames.GenericKernel(owner)
			if err == nil {
				deferredTarget, err = kernelNames.DeferredGenericKernel(owner)
			}
			if err == nil {
				deferredReference = deferredTarget.Reference()
				deferredTargetSelected = true
			}
		} else {
			ordinaryReference, err = context.Names().Reference(owner)
			if err == nil {
				kernelNames, available := context.Names().(api.GenericKernelNames)
				if !available {
					return api.ExpressionEmission{}, true, &api.ContextError{
						Reason: "generic callable variant names are unavailable",
					}
				}
				deferredTarget, err =
					kernelNames.DeferredGenericCallable(owner)
			}
			if err == nil {
				deferredReference = deferredTarget.Reference()
				deferredTargetSelected = true
			}
		}
		if err == nil {
			typeArguments, typeRequests, err =
				genericinstance.EmitFunctionTypeArguments(
					context,
					children,
					source,
					owner,
					instance.TypeArgs,
				)
		}
	}
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	if deferredTargetSelected &&
		deferredReference.ProviderBoundary() != ordinaryReference.ProviderBoundary() {
		return api.ExpressionEmission{}, true, &api.InvariantError{
			Role:   context.Role(),
			Reason: "generic callable variants disagree on provider ownership",
		}
	}
	if ordinaryReference.ProviderBoundary() {
		var providerBefore []tsgo.Statement
		var providerRequests []api.RootRequest
		arguments, providerBefore, providerRequests, err =
			providerboundary.ToProviderGenericArgumentsWithCallableParameters(
				context,
				children,
				contract.Params(),
				signature.Params(),
				arguments,
				providerParameters,
			)
		if err != nil {
			return api.ExpressionEmission{}, true, err
		}
		before = append(before, providerBefore...)
		argumentRequests = api.CombineRequests(
			argumentRequests,
			providerRequests,
		)
	}
	arguments = append(mechanicArgs, arguments...)
	recoveryObservation, err :=
		context.ObserveRecoveryCallable(callableFacet)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	reference := ordinaryReference
	if recoveryObservation.Recovery() {
		reference = deferredReference
		recovery := context.Factory().Identifier(callable.RecoveryAuthorityName)
		if deferredTargetSelected {
			arguments, err = deferredTarget.CallArguments(recovery, arguments)
		} else {
			arguments = append([]tsgo.Expression{recovery}, arguments...)
		}
		if err != nil {
			return api.ExpressionEmission{}, true, err
		}
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
		api.CombineRequests(
			reference.Requests(),
			concreteRequests,
			typeRequests,
			capabilityRequests,
			argumentRequests,
			recoveryObservation.Requests(),
		),
	)
	return target, true, err
}
