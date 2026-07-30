package cooperative

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

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
	if owner == nil {
		return api.NameReference{},
			api.CallableFacet{},
			api.GenericCallableProfileSelection{},
			&api.InvariantError{
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
		return api.NameReference{},
			api.CallableFacet{},
			api.GenericCallableProfileSelection{},
			err
	}
	var (
		reference api.NameReference
		facet     api.CallableFacet
	)
	if profileSelection.Cooperative() {
		profile, resolveErr := context.ResolveGenericCallableProfile(
			owner,
			profileSelection,
		)
		if resolveErr != nil {
			return api.NameReference{},
				api.CallableFacet{},
				api.GenericCallableProfileSelection{},
				resolveErr
		}
		reference, err = context.Names().GenericCallableProfile(profile)
		if err == nil {
			facet, err = api.NewGenericCallableProfileFacet(profile)
		}
	} else {
		reference, err = context.Names().Reference(owner)
		if err == nil {
			facet, err = api.NewSourceCallableFacet(owner)
		}
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
			requests,
		)...,
	)
	return reference, facet, profileSelection, err
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
