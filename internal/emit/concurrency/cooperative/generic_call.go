package cooperative

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func SelectGenericCallable(
	context api.Context,
	owner *types.Func,
) (api.NameReference, api.CallableFacet, error) {
	if owner == nil {
		return api.NameReference{}, api.CallableFacet{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "generic callable owner is nil",
		}
	}
	owner = owner.Origin()
	reference, err := context.Names().Reference(owner)
	if err != nil {
		return api.NameReference{}, api.CallableFacet{}, err
	}
	facet, err := api.NewSourceCallableFacet(owner)
	return reference, facet, err
}

func SelectGenericClassMethod(
	context api.Context,
	owner *types.Func,
) (api.CallableFacet, error) {
	if owner == nil {
		return api.CallableFacet{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "generic method owner is nil",
		}
	}
	return api.NewSourceCallableFacet(owner.Origin())
}

func GenericCall(
	context api.Context,
	source ast.Node,
	facet api.CallableFacet,
	target api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	if _, sourceFacet := facet.SourceFunction(); !sourceFacet {
		return api.ExpressionEmission{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "generic call facet is invalid",
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
		return api.ExpressionEmission{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "detached generic call facet is invalid",
		}
	}
	return facetCall(context, source, facet, target, false)
}

func GenericContract(
	context api.Context,
	facet api.CallableFacet,
) (bool, []api.RootRequest, error) {
	if _, sourceFacet := facet.SourceFunction(); !sourceFacet {
		return false, nil, &api.InvariantError{
			Role:   context.Role(),
			Reason: "generic callable contract facet is invalid",
		}
	}
	observation, err := context.ObserveCooperativeCallable(facet)
	if err != nil {
		return false, nil, err
	}
	return observation.Cooperative(), observation.Requests(), nil
}
