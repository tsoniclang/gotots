package cooperative

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

type interfaceMethodObservations struct {
	facets      []api.CallableFacet
	cooperative []bool
	any         bool
	requests    []api.RootRequest
}

func InterfaceMethodContract(
	context api.Context,
	reference api.InterfaceMethodCallableReference,
) (bool, []api.RootRequest, error) {
	observations, err := observeInterfaceMethods(
		context,
		[]api.InterfaceMethodCallableReference{reference},
	)
	if err != nil {
		return false, nil, err
	}
	return observations.any, observations.requests, nil
}

func ProviderInterfaceMethodContracts(
	context api.Context,
	provider api.CallableFacet,
	references []api.InterfaceMethodCallableReference,
) (bool, bool, []api.RootRequest, error) {
	if _, source := provider.SourceFunction(); !source {
		if _, profile := provider.GenericProfile(); !profile {
			return false, false, nil, &api.InvariantError{
				Role:   context.Role(),
				Reason: "interface-method provider facet is invalid",
			}
		}
	}
	observation, err := context.ObserveCooperativeCallable(provider)
	if err != nil {
		return false, false, nil, err
	}
	providers := interfaceMethodObservations{
		facets:      []api.CallableFacet{provider},
		cooperative: []bool{observation.Cooperative()},
		any:         observation.Cooperative(),
		requests:    observation.Requests(),
	}
	return joinInterfaceMethodContracts(context, providers, references)
}

func InterfaceProviderMethodContracts(
	context api.Context,
	provider api.InterfaceMethodCallableReference,
	references []api.InterfaceMethodCallableReference,
) (bool, bool, []api.RootRequest, error) {
	providers, err := observeInterfaceMethods(
		context,
		[]api.InterfaceMethodCallableReference{provider},
	)
	if err != nil {
		return false, false, nil, err
	}
	return joinInterfaceMethodContracts(context, providers, references)
}

func InterfaceMethodValueContract(
	context api.Context,
	reference api.InterfaceMethodCallableReference,
	signature *types.Signature,
) (bool, bool, []api.RootRequest, error) {
	providers, err := observeInterfaceMethods(
		context,
		[]api.InterfaceMethodCallableReference{reference},
	)
	if err != nil {
		return false, false, nil, err
	}
	abiReference, abiObservation, err := observeABI(context, signature)
	if err != nil {
		return false, false, nil, err
	}
	requests := api.CombineRequests(
		providers.requests,
		abiReference.Requests(),
		abiObservation.Requests(),
	)
	if providers.any && !abiObservation.Cooperative() {
		abiFacet, facetErr := context.CallableABIFacet(abiReference)
		if facetErr != nil {
			return false, false, nil, facetErr
		}
		selection, selectionErr :=
			api.NewCooperativeCallableRequest(abiFacet)
		if selectionErr != nil {
			return false, false, nil, selectionErr
		}
		requests = append(requests, selection)
	}
	return providers.any,
		providers.any || abiObservation.Cooperative(),
		requests,
		nil
}

func InterfaceMethodCall(
	context api.Context,
	source ast.Node,
	reference api.InterfaceMethodCallableReference,
	target api.ExpressionEmission,
	detached bool,
) (api.ExpressionEmission, error) {
	observations, err := observeInterfaceMethods(
		context,
		[]api.InterfaceMethodCallableReference{reference},
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	target, err = api.NewExpressionEmission(
		target.Before(),
		target.Value(),
		api.CombineRequests(
			target.Requests(),
			observations.requests,
		),
	)
	if err != nil || !observations.any {
		return target, err
	}
	return await(context, source, target, !detached, false)
}

func GeneratedInterfaceMethodCall(
	context api.Context,
	reference api.InterfaceMethodCallableReference,
	target api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	observations, err := observeInterfaceMethods(
		context,
		[]api.InterfaceMethodCallableReference{reference},
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	target, err = api.NewExpressionEmission(
		target.Before(),
		target.Value(),
		api.CombineRequests(
			target.Requests(),
			observations.requests,
		),
	)
	if err != nil || !observations.any {
		return target, err
	}
	return await(context, nil, target, true, true)
}

func GeneratedInterfaceProviderCall(
	context api.Context,
	target api.ExpressionEmission,
	providerCooperative bool,
) (api.ExpressionEmission, error) {
	if !providerCooperative {
		return target, nil
	}
	return await(context, nil, target, false, true)
}

func joinInterfaceMethodContracts(
	context api.Context,
	providers interfaceMethodObservations,
	references []api.InterfaceMethodCallableReference,
) (bool, bool, []api.RootRequest, error) {
	targets, err := observeInterfaceMethods(context, references)
	if err != nil {
		return false, false, nil, err
	}
	cooperative := providers.any || targets.any
	requests := api.CombineRequests(
		providers.requests,
		targets.requests,
	)
	if cooperative {
		for index, facet := range targets.facets {
			if targets.cooperative[index] {
				continue
			}
			selection, selectionErr :=
				api.NewCooperativeCallableRequest(facet)
			if selectionErr != nil {
				return false, false, nil, selectionErr
			}
			requests = append(requests, selection)
		}
	}
	return providers.any, cooperative, requests, nil
}

func observeInterfaceMethods(
	context api.Context,
	references []api.InterfaceMethodCallableReference,
) (interfaceMethodObservations, error) {
	if len(references) == 0 {
		return interfaceMethodObservations{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "interface-method references are absent",
		}
	}
	result := interfaceMethodObservations{}
	seen := make(map[*api.GeneratedArtifact]struct{})
	for _, reference := range references {
		result.requests = append(
			result.requests,
			reference.Requests()...,
		)
		for _, correspondence := range reference.Correspondences() {
			requests, err := JoinInterfaceMethodCallableABIs(
				context,
				correspondence,
			)
			if err != nil {
				return interfaceMethodObservations{}, err
			}
			result.requests = append(result.requests, requests...)
		}
		for _, artifact := range reference.Artifacts() {
			if _, duplicate := seen[artifact]; duplicate {
				continue
			}
			seen[artifact] = struct{}{}
			facet, err := api.NewInterfaceMethodCallableFacet(artifact)
			if err != nil {
				return interfaceMethodObservations{}, err
			}
			observation, err :=
				context.ObserveCooperativeCallable(facet)
			if err != nil {
				return interfaceMethodObservations{}, err
			}
			result.facets = append(result.facets, facet)
			result.cooperative = append(
				result.cooperative,
				observation.Cooperative(),
			)
			result.any = result.any || observation.Cooperative()
			result.requests = append(
				result.requests,
				observation.Requests()...,
			)
		}
	}
	if len(result.facets) == 0 {
		return interfaceMethodObservations{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "interface-method callable facets are absent",
		}
	}
	return result, nil
}
