package genericfunctionvalue

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/callable"
	cooperativecall "github.com/tsoniclang/gotots/internal/emit/concurrency/cooperative"
	"github.com/tsoniclang/gotots/internal/emit/deferredregistry"
	genericabi "github.com/tsoniclang/gotots/internal/emit/generic/abi"
	genericeffect "github.com/tsoniclang/gotots/internal/emit/generic/effect"
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
	effectSelection, err := genericeffect.ForExecutionProfile(context, owner)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	synchronousParameters := effectSelection.SynchronousParameters()
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
			effectSelection.Effect(),
		)
		if concreteErr != nil {
			return api.ExpressionEmission{}, true, concreteErr
		}
		reference, err = api.NewNameReference(
			concrete.Name(),
			concrete.Requests()...,
		)
		if err == nil && !effectSelection.Effect().Synchronous() {
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
		if effectSelection.Effect().Synchronous() {
			reference, err = kernelNames.SynchronousGenericKernel(owner)
		} else {
			reference, err = kernelNames.GenericKernel(owner)
		}
		if err == nil && !effectSelection.Effect().Synchronous() {
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
		if effectSelection.Effect().Synchronous() {
			kernelNames, available := context.Names().(api.GenericKernelNames)
			if !available {
				return api.ExpressionEmission{}, true, &api.ContextError{
					Reason: "generic kernel names are unavailable",
				}
			}
			reference, err = kernelNames.SynchronousGenericKernel(owner)
			if err == nil {
				callableFacet, err = api.NewSourceCallableFacet(owner)
			}
		} else {
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
	synchronousBoundary := context.SynchronousCallableBoundary()
	recoverySelected := false
	var recoveryObservationRequests []api.RootRequest
	if !synchronousBoundary {
		recoveryObservation, recoveryErr :=
			context.ObserveRecoveryCallable(callableFacet)
		if recoveryErr != nil {
			return api.ExpressionEmission{}, true, recoveryErr
		}
		recoverySelected = recoveryObservation.Recovery()
		recoveryObservationRequests = recoveryObservation.Requests()
	}
	providerCooperative := false
	var contractRequests []api.RootRequest
	if effectSelection.Effect().Synchronous() {
		providerCooperative = false
	} else if synchronousBoundary {
		sourceFacet, facetErr := api.NewSourceCallableFacet(owner)
		if facetErr != nil {
			return api.ExpressionEmission{}, true, facetErr
		}
		observation, observationErr :=
			context.ObserveCooperativeCallable(sourceFacet)
		if observationErr != nil {
			return api.ExpressionEmission{}, true, observationErr
		}
		if observation.Cooperative() {
			return api.ExpressionEmission{}, true, &api.InvariantError{
				Role:   context.Role(),
				Reason: "synchronous generic function value may suspend",
			}
		}
		contractRequests = observation.Requests()
	} else {
		providerCooperative, _, contractRequests, err =
			cooperativecall.GenericValueContract(
				context,
				callableFacet,
				signature,
			)
		if err != nil {
			return api.ExpressionEmission{}, true, err
		}
	}
	var target callable.SignatureEmission
	if effectSelection.Effect().Synchronous() {
		target, err = callable.EmitAdapterWithSynchronousParameters(
			context,
			children,
			source,
			signature,
			synchronousParameters,
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
	var modifiers []tsgo.ModifierLike
	resultType := target.Result()
	if providerCooperative {
		modifiers = []tsgo.ModifierLike{
			context.Factory().AsyncKeyword(),
		}
		resultType = callable.PromiseResult(
			context.Factory(),
			resultType,
		)
	}
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
		synchronousParameters,
		providerCooperative,
		nil,
		api.DeferredGenericRecoveryInvalid,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	ordinary := context.Factory().ArrowFunction(
		modifiers,
		nil,
		target.Parameters(),
		resultType,
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
				contractRequests,
				recoveryObservationRequests,
			)...,
		), true, nil
	}
	if effectSelection.Effect().Synchronous() {
		return api.ExpressionEmission{}, true, &api.InvariantError{
			Role:   context.Role(),
			Reason: "synchronous provider generic function value requires recovery transport",
		}
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
		nil,
		providerCooperative,
		recoveryValue,
		recoveryPlacement,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	deferred := context.Factory().ArrowFunction(
		modifiers,
		nil,
		append(
			[]tsgo.ParameterDeclaration{recovery},
			target.Parameters()...,
		),
		resultType,
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
			contractRequests,
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
	synchronousParameters []int,
	cooperative bool,
	recovery tsgo.Expression,
	recoveryPlacement api.DeferredGenericRecoveryPlacement,
) (api.ExpressionEmission, error) {
	arguments := sourceArguments
	var before []tsgo.Statement
	var requests []api.RootRequest
	var err error
	if reference.ProviderBoundary() {
		if len(synchronousParameters) != 0 {
			arguments, before, requests, err =
				providerboundary.ToProviderGenericArgumentsWithSynchronousParameters(
					context,
					children,
					contract.Params(),
					signature.Params(),
					sourceArguments,
					synchronousParameters,
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
	if cooperative {
		target, err = api.NewExpressionEmission(
			target.Before(),
			context.Factory().AwaitExpression(target.Value()),
			target.Requests(),
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
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
