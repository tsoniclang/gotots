package instance

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	genericabi "github.com/tsoniclang/gotots/internal/emit/generic/abi"
	pointertype "github.com/tsoniclang/gotots/internal/emit/type/pointer"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func EmitTypeArguments(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	declaration types.Object,
	arguments *types.TypeList,
) ([]tsgo.TypeNode, []api.RootRequest, error) {
	profile, resolved, err :=
		context.ResolveGenericRepresentationProfile(declaration)
	if err != nil {
		return nil, nil, err
	}
	parameters := profile.Parameters()
	if arguments == nil ||
		!resolved ||
		arguments.Len() != len(parameters) {
		return nil, nil, &api.InvariantError{
			Role:   context.Role(),
			Reason: "generic type arguments do not match their representation profile",
		}
	}
	targets := make([]tsgo.TypeNode, 0, arguments.Len()*3)
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
		if profile.Requires(
			parameters[index],
			api.GenericRepresentationStorage,
		) {
			storage, storageErr := context.Values().StorageType(
				context.WithRole(api.RoleStorageType),
				source,
				argument,
			)
			if storageErr != nil {
				return nil, nil, storageErr
			}
			targets = append(targets, storage.Value())
			requests = append(requests, storage.Requests()...)
		}
		if profile.Requires(
			parameters[index],
			api.GenericRepresentationPointer,
		) {
			pointer, pointerErr := pointertype.EmitNonNilRepresented(
				context.WithRole(api.RoleCallArgumentType),
				children,
				source,
				types.NewPointer(argument),
			)
			if pointerErr != nil {
				return nil, nil, pointerErr
			}
			targets = append(targets, pointer.Value())
			requests = append(requests, pointer.Requests()...)
		}
	}
	return targets, api.CombineRequests(requests), nil
}

func EmitCapabilities(
	context api.Context,
	source ast.Node,
	operationSet api.GenericOperationSet,
	arguments *types.TypeList,
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
		if api.ContainsGenericTypeParameter(signature) {
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
			if providerObservation.Cooperative() &&
				!consumerObservation.Cooperative() {
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
