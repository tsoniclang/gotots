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
	if left.Kind() == api.DeclarationRequirementGenericOperation {
		_, leftOperation, leftOK := left.GenericOperation()
		_, rightOperation, rightOK := right.GenericOperation()
		switch {
		case !leftOK && rightOK:
			return -1
		case leftOK && !rightOK:
			return 1
		case !leftOK:
			return 0
		}
		switch {
		case leftOperation.Key() < rightOperation.Key():
			return -1
		case leftOperation.Key() > rightOperation.Key():
			return 1
		default:
			return 0
		}
	}
	if left.Kind() == api.DeclarationRequirementCooperativeCallable {
		leftFacet, leftOK := left.CooperativeCallable()
		rightFacet, rightOK := right.CooperativeCallable()
		switch {
		case !leftOK && rightOK:
			return -1
		case leftOK && !rightOK:
			return 1
		case !leftOK:
			return 0
		case leftFacet.Kind() < rightFacet.Kind():
			return -1
		case leftFacet.Kind() > rightFacet.Kind():
			return 1
		}
		if leftLiteral, ok := leftFacet.FunctionLiteral(); ok {
			rightLiteral, _ := rightFacet.FunctionLiteral()
			switch {
			case leftLiteral.Pos() < rightLiteral.Pos():
				return -1
			case leftLiteral.Pos() > rightLiteral.Pos():
				return 1
			}
		}
		if leftOperation, ok := leftFacet.GenericOperation(); ok {
			rightOperation, _ := rightFacet.GenericOperation()
			switch {
			case leftOperation.Key() < rightOperation.Key():
				return -1
			case leftOperation.Key() > rightOperation.Key():
				return 1
			}
		}
		return 0
	}
	if left.Kind() == api.DeclarationRequirementAnonymousStruct {
		leftArtifact, leftDemand, _ := left.AnonymousStruct()
		rightArtifact, rightDemand, _ := right.AnonymousStruct()
		if order := compareGeneratedArtifacts(
			leftArtifact,
			rightArtifact,
		); order != 0 {
			return order
		}
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
		leftArtifact, leftDemand, _ := left.MapSpecialization()
		rightArtifact, rightDemand, _ := right.MapSpecialization()
		if order := compareGeneratedArtifacts(
			leftArtifact,
			rightArtifact,
		); order != 0 {
			return order
		}
		switch {
		case leftDemand < rightDemand:
			return -1
		case leftDemand > rightDemand:
			return 1
		default:
			return 0
		}
	}
	if artifactKinds(left.Kind()) {
		leftArtifact, _ := left.GeneratedArtifact()
		rightArtifact, _ := right.GeneratedArtifact()
		return compareGeneratedArtifacts(leftArtifact, rightArtifact)
	}
	if left.Kind() == api.DeclarationRequirementCallableControl {
		_, _, leftCallable, leftControl, leftOK := left.CallableControl()
		_, _, rightCallable, rightControl, rightOK := right.CallableControl()
		switch {
		case !leftOK && rightOK:
			return -1
		case leftOK && !rightOK:
			return 1
		case !leftOK:
			return 0
		case leftCallable == nil && rightCallable != nil:
			return -1
		case leftCallable != nil && rightCallable == nil:
			return 1
		case leftCallable == nil:
			return compareCallableControlRequirements(
				left,
				right,
				leftControl,
				rightControl,
			)
		case leftCallable.Pos() < rightCallable.Pos():
			return -1
		case leftCallable.Pos() > rightCallable.Pos():
			return 1
		default:
			return compareCallableControlRequirements(
				left,
				right,
				leftControl,
				rightControl,
			)
		}
	}
	leftType, leftOperation, _ := left.NamedStructOperation()
	rightType, rightOperation, _ := right.NamedStructOperation()
	if order := compareObjects(leftType, rightType); order != 0 {
		return order
	}
	switch {
	case leftOperation < rightOperation:
		return -1
	case leftOperation > rightOperation:
		return 1
	default:
		return 0
	}
}

func compareCallableControlRequirements(
	left api.DeclarationRequirement,
	right api.DeclarationRequirement,
	leftControl api.CallableControlFacet,
	rightControl api.CallableControlFacet,
) int {
	switch {
	case leftControl < rightControl:
		return -1
	case leftControl > rightControl:
		return 1
	case leftControl != api.CallableControlGoto:
		return 0
	}
	leftLabel, leftPosition, leftOK := left.GotoControl()
	rightLabel, rightPosition, rightOK := right.GotoControl()
	switch {
	case !leftOK && rightOK:
		return -1
	case leftOK && !rightOK:
		return 1
	case !leftOK:
		return 0
	}
	if order := compareObjects(leftLabel, rightLabel); order != 0 {
		return order
	}
	switch {
	case leftPosition < rightPosition:
		return -1
	case leftPosition > rightPosition:
		return 1
	default:
		return 0
	}
}

func artifactKinds(kind api.DeclarationRequirementKind) bool {
	return kind == api.DeclarationRequirementInterfaceAdapter ||
		kind == api.DeclarationRequirementAnonymousInterface ||
		kind == api.DeclarationRequirementInterfaceMethodToken ||
		kind == api.DeclarationRequirementInterfaceDynamicTypeToken ||
		kind == api.DeclarationRequirementGenericCapability ||
		kind == api.DeclarationRequirementCallableABI
}

func compareGeneratedArtifacts(
	left *api.GeneratedArtifact,
	right *api.GeneratedArtifact,
) int {
	switch {
	case left == nil && right != nil:
		return -1
	case left != nil && right == nil:
		return 1
	case left == nil:
		return 0
	case left.ArtifactKey() < right.ArtifactKey():
		return -1
	case left.ArtifactKey() > right.ArtifactKey():
		return 1
	default:
		return 0
	}
}
