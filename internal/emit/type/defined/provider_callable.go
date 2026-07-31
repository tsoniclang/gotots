package defined

import (
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/callable"
)

func providerCallableRequests(
	context api.Context,
	model Model,
) ([]api.RootRequest, error) {
	representation, err := context.Names().DefinedValueRepresentation(
		model.TypeName(),
	)
	if err != nil {
		return nil, err
	}
	cooperative, providerIdentity := representation.ProviderCallableEffect()
	if !providerIdentity || !cooperative {
		return nil, nil
	}
	signature, callableValue := model.Callable()
	if !callableValue {
		return nil, &api.InvariantError{
			Role:   context.Role(),
			Reason: "provider callable effect does not own a callable value",
		}
	}
	reference, err := callable.ABIReference(context, signature)
	if err != nil {
		return nil, err
	}
	facet, err := context.CallableABIFacet(reference)
	if err != nil {
		return nil, err
	}
	request, err := api.NewCooperativeCallableRequest(facet)
	if err != nil {
		return nil, err
	}
	return api.CombineRequests(
		reference.Requests(),
		[]api.RootRequest{request},
	), nil
}
