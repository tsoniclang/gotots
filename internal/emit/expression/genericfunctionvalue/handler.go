package genericfunctionvalue

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/callable"
	"github.com/tsoniclang/gotots/internal/emit/deferredregistry"
	genericabi "github.com/tsoniclang/gotots/internal/emit/generic/abi"
	genericparameters "github.com/tsoniclang/gotots/internal/emit/generic/callableparameters"
	genericinstance "github.com/tsoniclang/gotots/internal/emit/generic/instance"
	providerboundary "github.com/tsoniclang/gotots/internal/emit/value/providerboundary"
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
	parameterSelection, err := genericparameters.ForCallable(context, owner)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	providerParameters := parameterSelection.Parameters()
	providerKernel := parameterSelection.ProviderKernel()
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
		facet, selectionErr := callable.SelectGenericMethod(
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
		facet, selectionErr := callable.SelectGenericMethod(
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
					children,
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
		if providerKernel {
			kernelNames, available := context.Names().(api.GenericKernelNames)
			if !available {
				return api.ExpressionEmission{}, true, &api.ContextError{
					Reason: "generic kernel names are unavailable",
				}
			}
			reference, err = kernelNames.GenericKernel(owner)
			if err == nil {
				callableFacet, err = api.NewSourceCallableFacet(owner)
			}
			if err == nil {
				deferredTarget, err = kernelNames.DeferredGenericKernel(owner)
			}
			if err == nil {
				deferredReference = deferredTarget.Reference()
				deferredTargetSelected = true
			}
		} else {
			reference, callableFacet, err = callable.SelectGeneric(context, owner)
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
	providerBoundary := context.ProviderCallableBoundary()
	recoverySelected := false
	var recoveryObservationRequests []api.RootRequest
	if !providerBoundary {
		recoveryObservation, recoveryErr :=
			context.ObserveRecoveryCallable(callableFacet)
		if recoveryErr != nil {
			return api.ExpressionEmission{}, true, recoveryErr
		}
		recoverySelected = recoveryObservation.Recovery()
		recoveryObservationRequests = recoveryObservation.Requests()
	}
	var target callable.SignatureEmission
	if providerKernel {
		target, err = callable.EmitAdapterWithProviderCallableParameters(
			context,
			children,
			source,
			signature,
			providerParameters,
		)
	} else {
		target, err = callable.EmitABIAdapter(
			context,
			children,
			source,
			signature,
		)
	}
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	sourceArguments := target.SourceParameterReferences(context.Factory())
	contract, ok := owner.Type().(*types.Signature)
	if !ok {
		return api.ExpressionEmission{}, true, &api.InvariantError{
			Role:   context.Role(),
			Reason: "generic function-value owner has no signature",
		}
	}
	ordinaryCall, err := kernelValueInvocation(
		context,
		children,
		reference,
		typeArguments,
		mechanicArgs,
		sourceArguments,
		contract,
		signature,
		providerParameters,
		nil,
		api.DeferredGenericRecoveryInvalid,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	ordinary := context.Factory().ArrowFunction(
		nil,
		nil,
		target.Parameters(),
		target.Result(),
		context.Factory().EqualsGreaterThanToken(),
		callableBody(context.Factory(), signature.Results(), ordinaryCall),
	)
	if !recoverySelected {
		return api.DirectExpression(
			ordinary,
			api.CombineRequests(
				target.Requests(),
				typeRequests,
				mechanicReqs,
				ordinaryCall.Requests(),
				recoveryObservationRequests,
			)...,
		), true, nil
	}
	recovery, recoveryRequests, err :=
		callable.RecoveryAuthorityParameter(context)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	recoveryValue := context.Factory().Identifier(callable.RecoveryAuthorityName)
	recoveryPlacement := api.DeferredGenericRecoveryFirst
	if deferredTargetSelected {
		recoveryPlacement = deferredTarget.RecoveryPlacement()
	}
	if deferredReference.ProviderBoundary() != reference.ProviderBoundary() {
		return api.ExpressionEmission{}, true, &api.InvariantError{
			Role:   context.Role(),
			Reason: "generic function-value variants disagree on provider ownership",
		}
	}
	deferredCall, err := kernelValueInvocation(
		context,
		children,
		deferredReference,
		typeArguments,
		mechanicArgs,
		sourceArguments,
		contract,
		signature,
		providerParameters,
		recoveryValue,
		recoveryPlacement,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	deferred := context.Factory().ArrowFunction(
		nil,
		nil,
		append(
			[]tsgo.ParameterDeclaration{recovery},
			target.Parameters()...,
		),
		target.Result(),
		context.Factory().EqualsGreaterThanToken(),
		callableBody(context.Factory(), signature.Results(), deferredCall),
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
			ordinaryCall.Requests(),
			deferredCall.Requests(),
			recoveryRequests,
			registry.Requests(),
			recoveryObservationRequests,
		)...,
	), true, nil
}

func kernelValueInvocation(
	context api.Context,
	children api.ChildEmitter,
	reference api.NameReference,
	typeArguments []tsgo.TypeNode,
	mechanics []tsgo.Expression,
	sourceArguments []tsgo.Expression,
	contract *types.Signature,
	signature *types.Signature,
	providerCallableParameters []int,
	recovery tsgo.Expression,
	recoveryPlacement api.DeferredGenericRecoveryPlacement,
) (api.ExpressionEmission, error) {
	arguments := sourceArguments
	var before []tsgo.Statement
	var requests []api.RootRequest
	var err error
	if reference.ProviderBoundary() {
		if len(providerCallableParameters) != 0 {
			arguments, before, requests, err =
				providerboundary.ToProviderGenericArgumentsWithCallableParameters(
					context,
					children,
					contract.Params(),
					signature.Params(),
					sourceArguments,
					providerCallableParameters,
				)
		} else {
			arguments, before, requests, err =
				providerboundary.ToProviderGenericArguments(
					context,
					children,
					contract.Params(),
					signature.Params(),
					sourceArguments,
				)
		}
		if err != nil {
			return api.ExpressionEmission{}, err
		}
	}
	arguments = append(append([]tsgo.Expression{}, mechanics...), arguments...)
	if recoveryPlacement != api.DeferredGenericRecoveryInvalid {
		deferred, err := api.NewDeferredGenericCallableReference(
			reference,
			recoveryPlacement,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		arguments, err = deferred.CallArguments(recovery, arguments)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
	}
	target, err := api.NewExpressionEmission(
		before,
		context.Factory().CallExpression(
			reference.Expression(context.Factory()),
			nil,
			typeArguments,
			arguments,
			tsgo.NodeFlagsNone,
		),
		api.CombineRequests(reference.Requests(), requests),
	)
	if err != nil || !reference.ProviderBoundary() {
		return target, err
	}
	return providerboundary.FromProviderGenericResults(
		context,
		children,
		contract.Results(),
		signature.Results(),
		target,
	)
}

func callableBody(
	factory tsgo.Factory,
	results *types.Tuple,
	emission api.ExpressionEmission,
) tsgo.ConciseBody {
	statements := emission.Before()
	if results == nil || results.Len() == 0 {
		statements = append(statements, factory.ExpressionStatement(emission.Value()))
	} else {
		statements = append(statements, factory.ReturnStatement(emission.Value()))
	}
	return factory.Block(statements, true)
}
