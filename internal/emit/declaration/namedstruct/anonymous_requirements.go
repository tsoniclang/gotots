package namedstruct

import "github.com/tsoniclang/gotots/internal/emit/api"

func SelectAnonymousRequirements(
	role api.Role,
	artifact *api.GeneratedArtifact,
	requirements []api.DeclarationRequirement,
) (
	[]api.NamedStructOperation,
	[]api.TypeRepresentationFacet,
	error,
) {
	if artifact == nil ||
		artifact.Kind() != api.GeneratedArtifactAnonymousStruct {
		return nil, nil, &api.InvariantError{
			Role:   role,
			Reason: "anonymous struct requirement owner is invalid",
		}
	}
	var operations []api.NamedStructOperation
	var facets []api.TypeRepresentationFacet
	for _, requirement := range requirements {
		if selected, demand, ok := requirement.AnonymousStruct(); ok {
			if selected != artifact {
				return nil, nil, foreignAnonymousRequirement(role)
			}
			operation, materialized, err := anonymousOperation(demand)
			if err != nil {
				return nil, nil, err
			}
			if materialized {
				operations = append(operations, operation)
			}
			continue
		}
		typeName, selected, facet, ok := requirement.TypeRepresentation()
		if !ok || typeName != nil || selected != artifact {
			return nil, nil, foreignAnonymousRequirement(role)
		}
		facets = append(facets, facet)
	}
	return operations, facets, nil
}

func anonymousOperation(
	demand api.AnonymousStructDemand,
) (api.NamedStructOperation, bool, error) {
	switch demand {
	case api.AnonymousStructDemandDefinition:
		return api.NamedStructOperationInvalid, false, nil
	case api.AnonymousStructDemandZero:
		return api.NamedStructOperationZero, true, nil
	case api.AnonymousStructDemandCopy:
		return api.NamedStructOperationCopy, true, nil
	case api.AnonymousStructDemandEqual:
		return api.NamedStructOperationEqual, true, nil
	case api.AnonymousStructDemandHash:
		return api.NamedStructOperationHash, true, nil
	case api.AnonymousStructDemandConvert:
		return api.NamedStructOperationConvert, true, nil
	case api.AnonymousStructDemandStorage:
		return api.NamedStructOperationStorage, true, nil
	default:
		return api.NamedStructOperationInvalid, false, &api.InvariantError{
			Reason: "anonymous struct demand is invalid",
		}
	}
}

func foreignAnonymousRequirement(role api.Role) error {
	return &api.InvariantError{
		Role:   role,
		Reason: "anonymous struct received a foreign requirement",
	}
}
