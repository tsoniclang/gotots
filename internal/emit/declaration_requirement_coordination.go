package emit

import (
	"github.com/tsoniclang/gotots/internal/emit/api"
	emitordering "github.com/tsoniclang/gotots/internal/emit/ordering"
)

func compareDeclarationRequirements(
	left api.DeclarationRequirement,
	right api.DeclarationRequirement,
) int {
	if order := emitordering.CompareArtifactOwners(
		left.Owner(),
		right.Owner(),
	); order != 0 {
		return order
	}
	if left.Kind() < right.Kind() {
		return -1
	}
	if left.Kind() > right.Kind() {
		return 1
	}
	if left.Kind() == api.DeclarationRequirementClassMethod {
		_, leftMethod, leftOK := left.ClassMethod()
		_, rightMethod, rightOK := right.ClassMethod()
		switch {
		case !leftOK && rightOK:
			return -1
		case leftOK && !rightOK:
			return 1
		case !leftOK:
			return 0
		default:
			return emitordering.CompareObjects(leftMethod, rightMethod)
		}
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
			return emitordering.CompareObjects(leftVariable, rightVariable)
		}
	}
	if left.Kind() == api.DeclarationRequirementConstantProjection {
		_, leftProjection, _ := left.ConstantProjection()
		_, rightProjection, _ := right.ConstantProjection()
		return emitordering.CompareBasicKinds(leftProjection, rightProjection)
	}
	if left.Kind() == api.DeclarationRequirementLocalConstantProjection {
		_, leftConstant, leftProjection, _ := left.LocalConstantProjection()
		_, rightConstant, rightProjection, _ := right.LocalConstantProjection()
		if order := emitordering.CompareObjects(
			leftConstant,
			rightConstant,
		); order != 0 {
			return order
		}
		return emitordering.CompareBasicKinds(leftProjection, rightProjection)
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
	if left.Kind() == api.DeclarationRequirementGenericConcretization {
		leftConcretization, leftOK := left.GenericConcretization()
		rightConcretization, rightOK := right.GenericConcretization()
		switch {
		case !leftOK && rightOK:
			return -1
		case leftOK && !rightOK:
			return 1
		case !leftOK:
			return 0
		case leftConcretization.Key() < rightConcretization.Key():
			return -1
		case leftConcretization.Key() > rightConcretization.Key():
			return 1
		case !left.DeferredGenericConcretization() &&
			right.DeferredGenericConcretization():
			return -1
		case left.DeferredGenericConcretization() &&
			!right.DeferredGenericConcretization():
			return 1
		default:
			return 0
		}
	}
	if left.Kind() == api.DeclarationRequirementGenericRepresentation {
		return compareGenericRepresentationRequirements(left, right)
	}
	if left.Kind() == api.DeclarationRequirementTypeRepresentation {
		leftType, leftArtifact, leftFacet, leftOK :=
			left.TypeRepresentation()
		rightType, rightArtifact, rightFacet, rightOK :=
			right.TypeRepresentation()
		switch {
		case !leftOK && rightOK:
			return -1
		case leftOK && !rightOK:
			return 1
		}
		if order := emitordering.CompareObjects(leftType, rightType); order != 0 {
			return order
		}
		if order := compareGeneratedArtifacts(
			leftArtifact,
			rightArtifact,
		); order != 0 {
			return order
		}
		switch {
		case leftFacet < rightFacet:
			return -1
		case leftFacet > rightFacet:
			return 1
		default:
			return 0
		}
	}
	if left.Kind() == api.DeclarationRequirementGenericCallableProfile {
		leftProfile, leftOK := left.GenericCallableProfile()
		rightProfile, rightOK := right.GenericCallableProfile()
		switch {
		case !leftOK && rightOK:
			return -1
		case leftOK && !rightOK:
			return 1
		case !leftOK:
			return 0
		case leftProfile.Key() < rightProfile.Key():
			return -1
		case leftProfile.Key() > rightProfile.Key():
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
			leftProfile, leftProfiled :=
				leftFacet.FunctionLiteralProfile()
			rightProfile, rightProfiled :=
				rightFacet.FunctionLiteralProfile()
			switch {
			case !leftProfiled && rightProfiled:
				return -1
			case leftProfiled && !rightProfiled:
				return 1
			case leftProfiled &&
				leftProfile.Key() < rightProfile.Key():
				return -1
			case leftProfiled &&
				leftProfile.Key() > rightProfile.Key():
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
		if order, profiled := compareGenericProfileABIs(
			leftFacet,
			rightFacet,
		); profiled && order != 0 {
			return order
		}
		if leftProfile, ok := leftFacet.GenericProfile(); ok {
			rightProfile, _ := rightFacet.GenericProfile()
			switch {
			case leftProfile.Key() < rightProfile.Key():
				return -1
			case leftProfile.Key() > rightProfile.Key():
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
	if left.Kind() == api.DeclarationRequirementInterfaceAdapter {
		leftArtifact, _, leftKey, leftDemand :=
			left.InterfaceAdapterContract()
		rightArtifact, _, rightKey, rightDemand :=
			right.InterfaceAdapterContract()
		if !leftDemand {
			leftArtifact, _ = left.InterfaceAdapter()
		}
		if !rightDemand {
			rightArtifact, _ = right.InterfaceAdapter()
		}
		if order := compareGeneratedArtifacts(
			leftArtifact,
			rightArtifact,
		); order != 0 {
			return order
		}
		switch {
		case !leftDemand && rightDemand:
			return -1
		case leftDemand && !rightDemand:
			return 1
		case leftKey < rightKey:
			return -1
		case leftKey > rightKey:
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
	if order := emitordering.CompareObjects(leftType, rightType); order != 0 {
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
	case leftControl == api.CallableControlIteratorReturn:
		leftRange, leftOK := left.IteratorReturnControl()
		rightRange, rightOK := right.IteratorReturnControl()
		switch {
		case !leftOK && rightOK:
			return -1
		case leftOK && !rightOK:
			return 1
		case !leftOK:
			return 0
		case leftRange.Pos() < rightRange.Pos():
			return -1
		case leftRange.Pos() > rightRange.Pos():
			return 1
		default:
			return 0
		}
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
	if order := emitordering.CompareObjects(leftLabel, rightLabel); order != 0 {
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
	return kind == api.DeclarationRequirementAnonymousInterface ||
		kind == api.DeclarationRequirementInterfaceMethodToken ||
		kind == api.DeclarationRequirementInterfaceMethodCallable ||
		kind == api.DeclarationRequirementInterfaceDynamicTypeToken ||
		kind == api.DeclarationRequirementGenericCapability ||
		kind == api.DeclarationRequirementCallableABI ||
		kind == api.DeclarationRequirementPointerRepresentation
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
