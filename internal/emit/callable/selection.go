package callable

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func SelectGeneric(
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

func SelectGenericMethod(
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
