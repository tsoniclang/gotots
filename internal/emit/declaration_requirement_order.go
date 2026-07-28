package emit

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func compareBasicKinds(left types.BasicKind, right types.BasicKind) int {
	switch {
	case left < right:
		return -1
	case left > right:
		return 1
	default:
		return 0
	}
}

func compareDeclarationRequirements(
	left api.DeclarationRequirement,
	right api.DeclarationRequirement,
) int {
	if order := compareArtifactOwners(left.Owner(), right.Owner()); order != 0 {
		return order
	}
	if left.Kind() < right.Kind() {
		return -1
	}
	if left.Kind() > right.Kind() {
		return 1
	}
	if left.Kind() == api.DeclarationRequirementAddressableStorage {
		_, leftVariable, leftOK := left.AddressableStorage()
		_, rightVariable, rightOK := right.AddressableStorage()
		switch {
		case !leftOK && rightOK:
			return -1
		case leftOK && !rightOK:
			return 1
		case !leftOK:
			return 0
		default:
			return compareObjects(leftVariable, rightVariable)
		}
	}
	if left.Kind() == api.DeclarationRequirementConstantProjection {
		_, leftProjection, _ := left.ConstantProjection()
		_, rightProjection, _ := right.ConstantProjection()
		return compareBasicKinds(leftProjection, rightProjection)
	}
	if left.Kind() == api.DeclarationRequirementLocalConstantProjection {
		_, leftConstant, leftProjection, _ := left.LocalConstantProjection()
		_, rightConstant, rightProjection, _ := right.LocalConstantProjection()
		if order := compareObjects(leftConstant, rightConstant); order != 0 {
			return order
		}
		return compareBasicKinds(leftProjection, rightProjection)
	}
	if left.Kind() == api.DeclarationRequirementAnonymousStruct {
		_, leftDemand, _ := left.AnonymousStruct()
		_, rightDemand, _ := right.AnonymousStruct()
		switch {
		case leftDemand < rightDemand:
			return -1
		case leftDemand > rightDemand:
			return 1
		default:
			return 0
		}
	}
	if left.Kind() == api.DeclarationRequirementMapSpecialization {
		_, leftDemand, _ := left.MapSpecialization()
		_, rightDemand, _ := right.MapSpecialization()
		switch {
		case leftDemand < rightDemand:
			return -1
		case leftDemand > rightDemand:
			return 1
		default:
			return 0
		}
	}
	_, leftOperation, _ := left.NamedStructOperation()
	_, rightOperation, _ := right.NamedStructOperation()
	switch {
	case leftOperation < rightOperation:
		return -1
	case leftOperation > rightOperation:
		return 1
	default:
		return 0
	}
}
