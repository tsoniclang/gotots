package instance

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func typeRepresentationRequests(
	context api.Context,
	argument types.Type,
	facet api.GenericRepresentationFacet,
) ([]api.RootRequest, error) {
	selected, ok := materializedRepresentationFacet(facet)
	if !ok {
		return nil, nil
	}
	names, ok := context.Names().(api.TypeRepresentationNames)
	if !ok {
		return nil, &api.InvariantError{
			Role:   context.Role(),
			Reason: "type-representation name owner is unavailable",
		}
	}
	argument = types.Unalias(argument)
	switch source := argument.(type) {
	case *types.Named:
		origin := source.Origin()
		if origin == nil || origin.Obj() == nil {
			return nil, &api.InvariantError{
				Role:   context.Role(),
				Reason: "generic type representation has no named origin",
			}
		}
		if api.SupportsTypeRepresentation(origin.Obj()) {
			return names.TypeRepresentation(
				origin.Obj(),
				selected,
			)
		}
	case *types.Struct:
		if source.NumFields() != 0 {
			return names.AnonymousStructTypeRepresentation(
				source,
				selected,
			)
		}
	}
	return nil, nil
}

func materializedRepresentationFacet(
	facet api.GenericRepresentationFacet,
) (api.TypeRepresentationFacet, bool) {
	switch facet {
	case api.GenericRepresentationStorage:
		return api.TypeRepresentationStorage, true
	case api.GenericRepresentationContainerStorage:
		return api.TypeRepresentationContainerStorage, true
	case api.GenericRepresentationPointer:
		return api.TypeRepresentationPointer, true
	default:
		return api.TypeRepresentationInvalid, false
	}
}
