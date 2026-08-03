package genericfunctionvalue

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/callable"
	cooperativecall "github.com/tsoniclang/gotots/internal/emit/concurrency/cooperative"
	"github.com/tsoniclang/gotots/internal/emit/deferredregistry"
	genericabi "github.com/tsoniclang/gotots/internal/emit/generic/abi"
	genericinstance "github.com/tsoniclang/gotots/internal/emit/generic/instance"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source ast.Expr,
) (api.ExpressionEmission, bool, error) {
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
		instance.TypeArgs.Len() != len(operationSet.Parameters()) {
		return api.ExpressionEmission{}, true,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	requiresConcretization, err :=
		context.GenericCallableRequiresConcretization(owner)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	var (
		reference              api.NameReference
		deferredReference      api.NameReference
		deferredTarget         api.DeferredGenericCallableReference
		callableFacet          api.CallableFacet
		typeArguments          []tsgo.TypeNode
		typeRequests           []api.RootRequest
		mechanicArgs           []tsgo.Expression
		mechanicReqs           []api.RootRequest
		deferredTargetSelected bool
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
		if err == nil {
			concretizationNames, available :=
				context.Names().(api.GenericConcretizationNames)
			if !available {
				return api.ExpressionEmission{}, true, &api.ContextError{
					Reason: "generic concretization names are unavailable",
				}
			}
			deferredReference, err =
				concretizationNames.DeferredGenericConcretization(
					concrete.Concretization(),
				)
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
		if err == nil {
			deferredTarget, err =
				kernelNames.DeferredGenericKernel(owner)
			if err == nil {
				deferredReference = deferredTarget.Reference()
				deferredTargetSelected = true
			}
		}
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
				Reason: "generic mechanics reached a source-facing function value",
			}
		}
		reference, callableFacet, err =
			cooperativecall.SelectGenericCallable(context, owner)
		if err == nil {
			kernelNames, available := context.Names().(api.GenericKernelNames)
			if !available {
				return api.ExpressionEmission{}, true, &api.ContextError{
					Reason: "generic callable variant names are unavailable",
				}
			}
			deferredTarget, err =
				kernelNames.DeferredGenericCallable(owner)
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
	recoveryObservation, err := context.ObserveRecoveryCallable(callableFacet)
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
	sourceArguments := target.SourceParameterReferences(context.Factory())
	arguments := make(
		[]tsgo.Expression,
		0,
		len(mechanicArgs)+len(sourceArguments),
	)
	arguments = append(arguments, mechanicArgs...)
	arguments = append(arguments, sourceArguments...)
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
	ordinary := context.Factory().ArrowFunction(
		modifiers,
		nil,
		target.Parameters(),
		resultType,
		context.Factory().EqualsGreaterThanToken(),
		context.Factory().CallExpression(
			reference.Expression(context.Factory()),
			nil,
			typeArguments,
			arguments,
			tsgo.NodeFlagsNone,
		),
	)
	if !recoveryObservation.Recovery() {
		return api.DirectExpression(
			ordinary,
			api.CombineRequests(
				target.Requests(),
				typeRequests,
				mechanicReqs,
				reference.Requests(),
				contractRequests,
				recoveryObservation.Requests(),
			)...,
		), true, nil
	}
	recovery, recoveryRequests, err :=
		callable.RecoveryAuthorityParameter(context)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	deferredArguments := append(
		[]tsgo.Expression{
			context.Factory().Identifier(callable.RecoveryAuthorityName),
		},
		arguments...,
	)
	if deferredTargetSelected {
		deferredArguments, err = deferredTarget.CallArguments(
			context.Factory().Identifier(callable.RecoveryAuthorityName),
			arguments,
		)
		if err != nil {
			return api.ExpressionEmission{}, true, err
		}
	}
	deferredCallee := deferredReference.Expression(context.Factory())
	deferredRequests := deferredReference.Requests()
	deferred := context.Factory().ArrowFunction(
		modifiers,
		nil,
		append(
			[]tsgo.ParameterDeclaration{recovery},
			target.Parameters()...,
		),
		resultType,
		context.Factory().EqualsGreaterThanToken(),
		context.Factory().CallExpression(
			deferredCallee,
			nil,
			typeArguments,
			deferredArguments,
			tsgo.NodeFlagsNone,
		),
	)
	registry, err := deferredregistry.Reference(context, source, signature)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	return api.DirectExpression(
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
			[]tsgo.Expression{ordinary, deferred},
			tsgo.NodeFlagsNone,
		),
		api.CombineRequests(
			target.Requests(),
			typeRequests,
			mechanicReqs,
			reference.Requests(),
			deferredRequests,
			recoveryRequests,
			registry.Requests(),
			contractRequests,
			recoveryObservation.Requests(),
		)...,
	), true, nil
}
