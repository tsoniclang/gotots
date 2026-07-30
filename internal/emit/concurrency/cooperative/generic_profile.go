package cooperative

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

type genericCallableSelection struct {
	facet     api.CallableFacet
	profile   *api.GenericCallableProfile
	selection api.GenericCallableProfileSelection
	requests  []api.RootRequest
}

func SelectGenericCallable(
	context api.Context,
	owner *types.Func,
	declaration *types.Signature,
	instantiated *types.Signature,
) (
	api.NameReference,
	api.CallableFacet,
	api.GenericCallableProfileSelection,
	error,
) {
	selected, err := selectGenericCallable(
		context,
		owner,
		declaration,
		instantiated,
	)
	if err != nil {
		return api.NameReference{},
			api.CallableFacet{},
			api.GenericCallableProfileSelection{},
			err
	}
	owner = owner.Origin()
	var reference api.NameReference
	if selected.profile != nil {
		reference, err = context.Names().GenericCallableProfile(
			selected.profile,
		)
	} else {
		reference, err = context.Names().Reference(owner)
	}
	if err != nil {
		return api.NameReference{},
			api.CallableFacet{},
			api.GenericCallableProfileSelection{},
			err
	}
	reference, err = api.NewNameReference(
		reference.Name(),
		api.CombineRequests(
			reference.Requests(),
			selected.requests,
		)...,
	)
	return reference, selected.facet, selected.selection, err
}

func SelectGenericClassMethod(
	context api.Context,
	owner *types.Func,
	declaration *types.Signature,
	instantiated *types.Signature,
) (
	string,
	api.CallableFacet,
	api.GenericCallableProfileSelection,
	[]api.RootRequest,
	error,
) {
	selected, err := selectGenericCallable(
		context,
		owner,
		declaration,
		instantiated,
	)
	if err != nil {
		return "",
			api.CallableFacet{},
			api.GenericCallableProfileSelection{},
			nil,
			err
	}
	requests := selected.requests
	suffix := ""
	if selected.profile != nil {
		request, requestErr :=
			api.NewGenericCallableProfileRequest(selected.profile)
		if requestErr != nil {
			return "",
				api.CallableFacet{},
				api.GenericCallableProfileSelection{},
				nil,
				requestErr
		}
		requests = api.CombineRequests(
			requests,
			[]api.RootRequest{request},
		)
		suffix = selected.profile.Suffix()
	}
	return suffix,
		selected.facet,
		selected.selection,
		requests,
		nil
}

func selectGenericCallable(
	context api.Context,
	owner *types.Func,
	declaration *types.Signature,
	instantiated *types.Signature,
) (genericCallableSelection, error) {
	if owner == nil {
		return genericCallableSelection{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "generic callable owner is nil",
		}
	}
	owner = owner.Origin()
	profileSelection, requests, err :=
		CorrespondGenericCallableABIs(
			context,
			owner,
			declaration,
			instantiated,
		)
	if err != nil {
		return genericCallableSelection{}, err
	}
	if profileSelection.Cooperative() {
		profile, resolveErr := context.ResolveGenericCallableProfile(
			owner,
			profileSelection,
		)
		if resolveErr != nil {
			return genericCallableSelection{}, resolveErr
		}
		propagationRequests, propagationErr :=
			PropagateGenericCallableProfile(
				context,
				owner,
				profile,
				declaration,
				instantiated,
			)
		if propagationErr != nil {
			return genericCallableSelection{}, propagationErr
		}
		facet, facetErr := api.NewGenericCallableProfileFacet(profile)
		return genericCallableSelection{
			facet:     facet,
			profile:   profile,
			selection: profileSelection,
			requests: api.CombineRequests(
				requests,
				propagationRequests,
			),
		}, facetErr
	}
	facet, err := api.NewSourceCallableFacet(owner)
	return genericCallableSelection{
		facet:     facet,
		selection: profileSelection,
		requests:  requests,
	}, err
}

func GenericCall(
	context api.Context,
	source ast.Node,
	facet api.CallableFacet,
	target api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	if _, sourceFacet := facet.SourceFunction(); !sourceFacet {
		if _, profileFacet := facet.GenericProfile(); !profileFacet {
			return api.ExpressionEmission{}, &api.InvariantError{
				Role:   context.Role(),
				Reason: "generic call facet is invalid",
			}
		}
	}
	return facetCall(context, source, facet, target, true)
}

func DetachedGenericCall(
	context api.Context,
	source ast.Node,
	facet api.CallableFacet,
	target api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	if _, sourceFacet := facet.SourceFunction(); !sourceFacet {
		if _, profileFacet := facet.GenericProfile(); !profileFacet {
			return api.ExpressionEmission{}, &api.InvariantError{
				Role:   context.Role(),
				Reason: "detached generic call facet is invalid",
			}
		}
	}
	return facetCall(context, source, facet, target, false)
}

func GenericContract(
	context api.Context,
	facet api.CallableFacet,
) (bool, []api.RootRequest, error) {
	if _, sourceFacet := facet.SourceFunction(); !sourceFacet {
		if _, profileFacet := facet.GenericProfile(); !profileFacet {
			return false, nil, &api.InvariantError{
				Role:   context.Role(),
				Reason: "generic callable contract facet is invalid",
			}
		}
	}
	observation, err := context.ObserveCooperativeCallable(facet)
	if err != nil {
		return false, nil, err
	}
	return observation.Cooperative(), observation.Requests(), nil
}
