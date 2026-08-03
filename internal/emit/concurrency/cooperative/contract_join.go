package cooperative

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/callable"
	"github.com/tsoniclang/gotots/internal/emit/type/fieldidentity"
)

func JoinInterfaceMethodCallableABIs(
	context api.Context,
	selected api.InterfaceMethodCallableCorrespondence,
) ([]api.RootRequest, error) {
	owner, declaration, instantiated := selected.Parts()
	if owner == nil || declaration == nil || instantiated == nil {
		return nil, &api.InvariantError{
			Role:   context.Role(),
			Reason: "interface-method callable correspondence is invalid",
		}
	}
	var requests []api.RootRequest
	correspondence := callableCorrespondence{
		seen:                make(map[genericTypePair]struct{}),
		stopAtNamedBoundary: true,
		leaf: contractJoinLeaf(
			context,
			owner,
			&requests,
		),
	}
	if err := correspondence.signatureMembers(
		declaration,
		instantiated,
	); err != nil {
		return nil, err
	}
	return api.CombineRequests(requests), nil
}

func JoinNominalFieldCallableABIs(
	context api.Context,
	container types.Type,
	field *types.Var,
) ([]api.RootRequest, error) {
	selected, applicable, err := fieldidentity.Resolve(container, field)
	if err != nil {
		return nil, &api.InvariantError{
			Role:   context.Role(),
			Reason: err.Error(),
		}
	}
	if !applicable {
		return nil, nil
	}
	owner, declaration, instantiated := selected.Parts()
	var requests []api.RootRequest
	correspondence := callableCorrespondence{
		seen:                make(map[genericTypePair]struct{}),
		traverseIdentical:   true,
		stopAtNamedBoundary: true,
		leaf: contractJoinLeaf(
			context,
			owner,
			&requests,
		),
	}
	if err := correspondence.pair(
		declaration,
		instantiated,
	); err != nil {
		return nil, err
	}
	return api.CombineRequests(requests), nil
}

func contractJoinLeaf(
	context api.Context,
	owner types.Object,
	requests *[]api.RootRequest,
) func(*types.Signature, *types.Signature) error {
	return func(
		declaration *types.Signature,
		instantiated *types.Signature,
	) error {
		declarationReference, err :=
			context.Names().SourceCallableABI(owner, declaration)
		if err != nil {
			return err
		}
		instantiatedReference, err :=
			callable.ABIReference(context, instantiated)
		if err != nil {
			return err
		}
		declarationFacet, err := api.NewCallableABIFacet(
			declarationReference.Artifact(),
		)
		if err != nil {
			return err
		}
		instantiatedFacet, err :=
			context.CallableABIFacet(instantiatedReference)
		if err != nil {
			return err
		}
		declarationObservation, err :=
			context.ObserveCooperativeCallable(declarationFacet)
		if err != nil {
			return err
		}
		instantiatedObservation, err :=
			context.ObserveCooperativeCallable(instantiatedFacet)
		if err != nil {
			return err
		}
		*requests = append(
			*requests,
			declarationReference.Requests()...,
		)
		*requests = append(
			*requests,
			instantiatedReference.Requests()...,
		)
		*requests = append(
			*requests,
			declarationObservation.Requests()...,
		)
		*requests = append(
			*requests,
			instantiatedObservation.Requests()...,
		)
		cooperative := declarationObservation.Cooperative() ||
			instantiatedObservation.Cooperative()
		if cooperative {
			for _, candidate := range []api.CallableFacet{
				declarationFacet,
				instantiatedFacet,
			} {
				request, requestErr :=
					api.NewCooperativeCallableRequest(candidate)
				if requestErr != nil {
					return requestErr
				}
				*requests = append(*requests, request)
			}
		}
		return nil
	}
}
