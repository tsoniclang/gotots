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
	operationSet, ok, err := context.ResolveGenericCallable(owner)
	if err != nil {
		return api.ExpressionEmission{}, true, err
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
	declarationSignature, ok := owner.Type().(*types.Signature)
	if !ok {
		return api.ExpressionEmission{}, true,
			api.Unsupported(context, api.CategoryStatement, source)
	}
	_, callableFacet, _, selectionRequests, err :=
		cooperativecall.SelectGenericClassMethod(
			context,
			owner,
			declarationSignature,
			signature,
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
	var (
		ordinaryReference      api.NameReference
		deferredReference      api.NameReference
		deferredTarget         api.DeferredGenericCallableReference
		typeArguments          []tsgo.TypeNode
		typeRequests           []api.RootRequest
		capabilityRequests     []api.RootRequest
		concreteRequests       []api.RootRequest
		deferredTargetSelected bool
	)
	switch {
	case requiresConcretization && !openConcretization:
		profile, _ := callableFacet.GenericProfile()
		concretization, concreteErr := context.ResolveGenericConcretization(
			owner,
			instance.TypeArgs,
			signature,
			profile,
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
			api.CombineRequests(
				concretization.Requests(),
				selectionRequests,
			)...,
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
		profile, _ := callableFacet.GenericProfile()
		ordinaryReference, err = kernelNames.GenericKernel(owner, profile)
		if err == nil {
			deferredTarget, err =
				kernelNames.DeferredGenericKernel(owner, profile)
		}
		if err == nil {
			deferredReference = deferredTarget.Reference()
			deferredTargetSelected = true
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
					source,
					operationSet,
					instance.TypeArgs,
				)
			if err == nil {
				var mechanics []tsgo.Expression
				mechanics, err = genericabi.JoinCapabilities(
					owner,
					operationSet.Operations(),
					capabilities,
				)
				arguments = append(mechanics, arguments...)
			}
		}
	default:
		if len(operationSet.Operations()) != 0 {
			return api.ExpressionEmission{}, true, &api.InvariantError{
				Role:   context.Role(),
				Reason: "generic mechanics reached a source-facing deferred call",
			}
		}
		profile, profiled := callableFacet.GenericProfile()
		if profiled {
			ordinaryReference, err =
				context.Names().GenericCallableProfile(profile)
		} else {
			ordinaryReference, err = context.Names().Reference(owner)
		}
		if err == nil {
			kernelNames, available := context.Names().(api.GenericKernelNames)
			if !available {
				return api.ExpressionEmission{}, true, &api.ContextError{
					Reason: "generic callable variant names are unavailable",
				}
			}
			deferredTarget, err =
				kernelNames.DeferredGenericCallable(owner, profile)
		}
		if err == nil {
			deferredReference = deferredTarget.Reference()
			deferredTargetSelected = true
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
			selectionRequests,
			concreteRequests,
			typeRequests,
			capabilityRequests,
			argumentRequests,
			contractRequests,
			recoveryObservation.Requests(),
		),
	)
	return target, true, err
}
