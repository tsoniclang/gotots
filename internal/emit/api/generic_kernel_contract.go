package api

import "go/types"

func GenericKernelRequired(
	owner *types.Func,
	requirements []DeclarationRequirement,
) (bool, error) {
	if owner == nil || owner.Origin() != owner ||
		len(GenericDeclarationParameters(owner)) == 0 {
		return false, &InvariantError{
			Reason: "generic kernel owner is invalid",
		}
	}
	required := false
	for _, requirement := range requirements {
		switch requirement.Kind() {
		case DeclarationRequirementGenericOperation:
			requirementOwner, _, ok := requirement.GenericOperation()
			if !ok || requirementOwner != owner {
				return false, &InvariantError{
					Reason: "generic operation has foreign kernel ownership",
				}
			}
			required = true
		case DeclarationRequirementGenericRepresentation:
			requirementOwner, _, _, ok :=
				requirement.GenericRepresentation()
			if !ok || requirementOwner != owner {
				return false, &InvariantError{
					Reason: "generic representation has foreign kernel ownership",
				}
			}
		}
	}
	return required, nil
}
