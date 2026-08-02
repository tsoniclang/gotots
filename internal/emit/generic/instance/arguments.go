package instance

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/deferredregistry"
	genericabi "github.com/tsoniclang/gotots/internal/emit/generic/abi"
	pointertype "github.com/tsoniclang/gotots/internal/emit/type/pointer"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func EmitTypeArguments(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	declaration types.Object,
	arguments api.TypeArgumentList,
) ([]tsgo.TypeNode, []api.RootRequest, error) {
	profile, resolved, err :=
		context.ResolveGenericRepresentationProfile(declaration)
	if err != nil {
		return nil, nil, err
	}
	parameters := profile.Parameters()
	if !resolved ||
		arguments.Len() != len(parameters) {
		return nil, nil, &api.InvariantError{
			Role:   context.Role(),
			Reason: "generic type arguments do not match their representation profile",
		}
	}
	targets := make([]tsgo.TypeNode, 0, arguments.Len()*4)
	var requests []api.RootRequest
	for index := range arguments.Len() {
		argument := arguments.At(index)
		target, err := children.RepresentedType(
			context.WithRole(api.RoleCallArgumentType),
			source,
			argument,
		)
		if err != nil {
			return nil, nil, err
		}
		targets = append(targets, target.Value())
		requests = append(requests, target.Requests()...)
		for _, facet := range api.GenericRepresentationFacetOrder() {
			if !profile.Requires(parameters[index], facet) {
				continue
			}
			representation, representationErr := emitRepresentationArgument(
				context,
				children,
				source,
				argument,
				facet,
			)
			if representationErr != nil {
				return nil, nil, representationErr
			}
			requests = append(requests, representation.Requests()...)
		}
	}
	return targets, api.CombineRequests(requests), nil
}

func EmitFunctionTypeArguments(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	declaration *types.Func,
	arguments api.TypeArgumentList,
) ([]tsgo.TypeNode, []api.RootRequest, error) {
	projection, providerOwned, err :=
		context.Names().ProviderGenericTypeArguments(declaration)
	if err != nil {
		return nil, nil, err
	}
	if !providerOwned {
		return EmitTypeArguments(
			context,
			children,
			source,
			declaration,
			arguments,
		)
	}
	if arguments.Len() == 0 {
		return nil, nil, &api.InvariantError{
			Role:   context.Role(),
			Reason: "provider generic callable has no source type arguments",
		}
	}
	targets := make([]tsgo.TypeNode, 0, len(projection))
	var requests []api.RootRequest
	for _, projected := range projection {
		index := projected.Parameter()
		if index < 0 || index >= arguments.Len() {
			return nil, nil, &api.InvariantError{
				Role:   context.Role(),
				Reason: "provider generic type-argument projection is outside the source instance",
			}
		}
		target, err := emitProviderTypeArgument(
			context,
			children,
			source,
			arguments.At(index),
			projected.Facet(),
		)
		if err != nil {
			return nil, nil, err
		}
		targets = append(targets, target.Value())
		requests = append(requests, target.Requests()...)
	}
	return targets, api.CombineRequests(requests), nil
}

func emitProviderTypeArgument(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	argument types.Type,
	facet api.GenericTypeArgumentFacet,
) (api.TypeEmission, error) {
	if facet == api.GenericTypeArgumentLogical {
		return children.RepresentedType(
			context.WithRole(api.RoleCallArgumentType),
			source,
			argument,
		)
	}
	var representation api.GenericRepresentationFacet
	switch facet {
	case api.GenericTypeArgumentStorage:
		representation = api.GenericRepresentationStorage
	case api.GenericTypeArgumentContainerStorage:
		representation = api.GenericRepresentationContainerStorage
	case api.GenericTypeArgumentPointer:
		representation = api.GenericRepresentationPointer
	default:
		return api.TypeEmission{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "provider generic type-argument facet is invalid",
		}
	}
	return emitRepresentationArgument(
		context,
		children,
		source,
		argument,
		representation,
	)
}

func emitRepresentationArgument(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	argument types.Type,
	facet api.GenericRepresentationFacet,
) (api.TypeEmission, error) {
	var target api.TypeEmission
	var err error
	switch facet {
	case api.GenericRepresentationStorage:
		target, err = context.Values().StorageType(
			context.WithRole(api.RoleStorageType),
			source,
			argument,
		)
	case api.GenericRepresentationContainerStorage:
		target, err = context.ContainerStorage().ContainerStorageType(
			context.WithRole(api.RoleStorageType),
			source,
			argument,
		)
	case api.GenericRepresentationPointer:
		target, err = pointertype.EmitNonNilRepresented(
			context.WithRole(api.RoleCallArgumentType),
			children,
			source,
			types.NewPointer(argument),
		)
	default:
		return api.TypeEmission{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "generic representation argument facet is invalid",
		}
	}
	if err != nil {
		return target, err
	}
	facetRequests, err := typeRepresentationRequests(
		context,
		argument,
		facet,
	)
	if err != nil {
		return api.TypeEmission{}, err
	}
	return api.DirectType(
		target.Value(),
		api.CombineRequests(target.Requests(), facetRequests)...,
	), nil
}

func EmitCapabilities(
	context api.Context,
	source ast.Node,
	operationSet api.GenericOperationSet,
	arguments api.TypeArgumentList,
) ([]genericabi.Binding[tsgo.Expression], []api.RootRequest, error) {
	targets := make(
		[]genericabi.Binding[tsgo.Expression],
		0,
		len(operationSet.Operations()),
	)
	var requests []api.RootRequest
	for _, operation := range operationSet.Operations() {
		signature, err := InstantiateOperation(
			operationSet,
			operation.Signature(),
			arguments,
		)
		if err != nil {
			return nil, nil, err
		}
		var (
			referenceName     string
			referenceRequests []api.RootRequest
			providerFacet     api.CallableFacet
		)
		cooperativeCapability :=
			operation.Consumer() ==
				api.GenericFunctionOperationConsumer() &&
				operation.Operation() ==
					api.GenericOperationConstraintMethod
		if operation.Operation() ==
			api.GenericOperationDeferredCallableRegistry &&
			!api.ContainsGenericTypeParameter(signature) {
			registry, registryErr := deferredregistry.Reference(
				context,
				source,
				signature,
			)
			err = registryErr
			if err == nil {
				referenceName = registry.Name()
				referenceRequests = registry.Requests()
			}
		} else if api.ContainsGenericTypeParameter(signature) {
			reference, referenceErr := context.ProjectGenericOperation(
				source,
				operation,
				signature,
			)
			err = referenceErr
			if err == nil {
				referenceName = reference.Name()
				referenceRequests = reference.Requests()
				if cooperativeCapability {
					providerFacet, err =
						api.NewGenericOperationCallableFacet(
							reference.Contract(),
						)
				}
			}
		} else {
			reference, referenceErr :=
				context.Names().GenericCapability(
					operation.Selection(),
					signature,
				)
			err = referenceErr
			if err == nil {
				referenceName = reference.Name()
				referenceRequests = reference.Requests()
				if cooperativeCapability {
					providerFacet, err =
						api.NewGenericCapabilityCallableFacet(
							reference.Artifact(),
						)
				}
			}
		}
		if err != nil {
			return nil, nil, err
		}
		contractRequests := referenceRequests
		if cooperativeCapability {
			consumerFacet, facetErr :=
				api.NewGenericOperationCallableFacet(operation)
			if facetErr != nil {
				return nil, nil, facetErr
			}
			providerObservation, observationErr :=
				context.ObserveCooperativeCallable(providerFacet)
			if observationErr != nil {
				return nil, nil, observationErr
			}
			consumerObservation, observationErr :=
				context.ObserveCooperativeCallable(consumerFacet)
			if observationErr != nil {
				return nil, nil, observationErr
			}
			contractRequests = api.CombineRequests(
				referenceRequests,
				providerObservation.Requests(),
				consumerObservation.Requests(),
			)
			if providerObservation.Cooperative() {
				request, requestErr :=
					api.NewCooperativeCallableRequest(consumerFacet)
				if requestErr != nil {
					return nil, nil, requestErr
				}
				contractRequests = append(contractRequests, request)
			}
		}
		binding, err := genericabi.Capability[tsgo.Expression](
			operation,
			context.Factory().Identifier(referenceName),
		)
		if err != nil {
			return nil, nil, err
		}
		targets = append(targets, binding)
		requests = append(requests, contractRequests...)
	}
	return targets, requests, nil
}
