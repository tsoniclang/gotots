package emit

import (
	"go/types"
	"slices"

	"github.com/tsoniclang/gotots/internal/emit/api"
	declarationindex "github.com/tsoniclang/gotots/internal/emit/declaration/index"
	emitordering "github.com/tsoniclang/gotots/internal/emit/ordering"
)

func sourceArtifactOwnerOrder(
	sites map[types.Object]declarationSite,
) func(api.ArtifactOwner, api.ArtifactOwner) int {
	return func(left api.ArtifactOwner, right api.ArtifactOwner) int {
		leftObject, leftSource := left.Source()
		rightObject, rightSource := right.Source()
		if leftSource && rightSource {
			leftSite, leftIndexed := sites[leftObject]
			rightSite, rightIndexed := sites[rightObject]
			if leftIndexed && rightIndexed {
				if order := declarationindex.CompareSites(
					leftSite,
					rightSite,
				); order != 0 {
					return order
				}
			}
		}
		return emitordering.CompareArtifactOwners(left, right)
	}
}

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
		leftArtifact, _, _, leftKey, leftDemand :=
			left.InterfaceAdapterContract()
		rightArtifact, _, _, rightKey, rightDemand :=
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
	if left.Kind() == api.DeclarationRequirementProviderInterfaceCapability {
		leftArtifact, _, leftKey, leftOK :=
			left.ProviderInterfaceCapability()
		rightArtifact, _, rightKey, rightOK :=
			right.ProviderInterfaceCapability()
		if order := compareGeneratedArtifacts(
			leftArtifact,
			rightArtifact,
		); order != 0 {
			return order
		}
		switch {
		case !leftOK && rightOK:
			return -1
		case leftOK && !rightOK:
			return 1
		case leftKey < rightKey:
			return -1
		case leftKey > rightKey:
			return 1
		default:
			return 0
		}
	}
	if left.Kind() == api.DeclarationRequirementProviderProfileInterfaceCapability {
		leftArtifact, leftTarget, leftOK :=
			left.ProviderProfileInterfaceCapability()
		rightArtifact, rightTarget, rightOK :=
			right.ProviderProfileInterfaceCapability()
		if order := compareGeneratedArtifacts(
			leftArtifact,
			rightArtifact,
		); order != 0 {
			return order
		}
		switch {
		case !leftOK && rightOK:
			return -1
		case leftOK && !rightOK:
			return 1
		default:
			return compareGeneratedArtifacts(leftTarget, rightTarget)
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

func (s *programSession) settle() error {
	for {
		if object, ok := s.scheduler.next(); ok {
			if err := s.emit(object); err != nil {
				return err
			}
			continue
		}
		if owner, requirements, removed, ok := s.requirements.NextBatch(); ok {
			if err := s.applyDeclarationRequirements(
				owner,
				requirements,
				removed,
			); err != nil {
				return err
			}
			continue
		}
		if dirty := s.artifacts.DirtyBatch(); len(dirty) != 0 {
			for _, object := range dirty {
				if err := s.reconstructScheduledArtifact(object); err != nil {
					return err
				}
			}
			continue
		}
		if sourcePackage, ok := s.packageInitializations.next(); ok {
			if err := s.emitPackageInitialization(sourcePackage); err != nil {
				return err
			}
			continue
		}
		if s.requirements.FinalizeRemovals() {
			continue
		}
		if builders := s.packageExports.nextBatch(); len(builders) != 0 {
			for _, builder := range builders {
				if err := s.publishPackageExports(builder); err != nil {
					return err
				}
			}
			continue
		}
		return nil
	}
}

func (s *programSession) removeTargetDeclaration(
	owner api.ArtifactOwner,
) (bool, error) {
	removed := false
	for outputPath, builder := range s.builders {
		index, exists := builder.indexByOwner[owner]
		if !exists {
			continue
		}
		if removed || index < 0 || index >= len(builder.declarations) ||
			builder.declarations[index].owner != owner {
			return false, &ScheduleError{
				Object: owner.Name(),
				Reason: "target declaration ownership is inconsistent",
			}
		}
		removed = true
		builder.declarations = append(
			builder.declarations[:index],
			builder.declarations[index+1:]...,
		)
		delete(builder.byOwner, owner)
		delete(builder.indexByOwner, owner)
		for declarationIndex := index; declarationIndex < len(builder.declarations); declarationIndex++ {
			builder.indexByOwner[builder.declarations[declarationIndex].owner] = declarationIndex
		}
		if len(builder.declarations) == 0 {
			delete(s.builders, outputPath)
		}
	}
	return removed, nil
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
	case leftControl == api.CallableControlDefer:
		leftDefer, leftOK := left.DeferControl()
		rightDefer, rightOK := right.DeferControl()
		switch {
		case !leftOK && rightOK:
			return -1
		case leftOK && !rightOK:
			return 1
		case !leftOK:
			return 0
		case leftDefer.Pos() < rightDefer.Pos():
			return -1
		case leftDefer.Pos() > rightDefer.Pos():
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
		kind == api.DeclarationRequirementReflectionType ||
		kind == api.DeclarationRequirementReflectionValueOperations
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

func (s *programSession) installSourceImplementationRequirements() error {
	if s.sourceImplementationContracts == nil {
		return nil
	}
	if !s.requirements.CertifiedEmpty() {
		return &ScheduleError{
			Reason: "source-implementation accepted requirements were installed more than once",
		}
	}
	var selected []api.DeclarationRequirement
	for owner, contract := range s.sourceImplementationContracts {
		for _, requirement := range contract.acceptedRequirements {
			if !requirement.Valid() || requirement.Owner() != owner {
				return &ScheduleError{
					Object: owner.Name(),
					Reason: "source-implementation accepted requirement has invalid ownership",
				}
			}
			selected = append(selected, requirement)
		}
	}
	slices.SortFunc(selected, compareDeclarationRequirements)
	seen := make(map[api.DeclarationRequirement]struct{}, len(selected))
	for _, requirement := range selected {
		if _, duplicate := seen[requirement]; duplicate {
			return &ScheduleError{
				Object: requirement.Owner().Name(),
				Reason: "source-implementation accepted requirement is duplicated",
			}
		}
		seen[requirement] = struct{}{}
	}
	if !s.requirements.InstallCertified(selected) {
		return &ScheduleError{
			Reason: "source-implementation accepted requirements were installed more than once",
		}
	}
	return nil
}
