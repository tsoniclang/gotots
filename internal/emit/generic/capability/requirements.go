package capability

import (
	"github.com/tsoniclang/gotots/internal/emit/api"
)

func ValidateRequirements(
	role api.Role,
	artifact *api.GeneratedArtifact,
	requirements []api.DeclarationRequirement,
) error {
	definitions := 0
	for _, requirement := range requirements {
		if selected, ok := requirement.GenericCapability(); ok {
			if selected != artifact {
				return requirementError(
					role,
					"generic capability received a foreign definition",
				)
			}
			definitions++
			continue
		}
		return requirementError(
			role,
			"generic capability received a foreign requirement",
		)
	}
	if definitions != 1 {
		return requirementError(
			role,
			"generic capability requires exactly one definition request",
		)
	}
	return nil
}

func requirementError(role api.Role, reason string) error {
	return &api.InvariantError{Role: role, Reason: reason}
}
