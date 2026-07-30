package emit

import (
	"go/types"
	"sort"

	"github.com/tsoniclang/gotots/internal/emit/api"
	emitordering "github.com/tsoniclang/gotots/internal/emit/ordering"
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
		return compareBasicKinds(leftProjection, rightProjection)
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
	if left.Kind() == api.DeclarationRequirementGenericRepresentation {
		return compareGenericRepresentationRequirements(left, right)
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
	if left.Kind() == api.DeclarationRequirementEnvironmentBuiltin {
		_, leftSignature, leftOK := left.EnvironmentBuiltin()
		_, rightSignature, rightOK := right.EnvironmentBuiltin()
		switch {
		case !leftOK && rightOK:
			return -1
		case leftOK && !rightOK:
			return 1
		case !leftOK:
			return 0
		}
		leftType := stableTypeString(leftSignature)
		rightType := stableTypeString(rightSignature)
		switch {
		case leftType < rightType:
			return -1
		case leftType > rightType:
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

func stableTypeString(source types.Type) string {
	return types.TypeString(source, func(sourcePackage *types.Package) string {
		if sourcePackage == nil {
			return ""
		}
		return sourcePackage.Path()
	})
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

type declarationRequirementLedger struct {
	byOwner map[api.ArtifactOwner]map[api.DeclarationRequirement]struct{}
}

func newDeclarationRequirementLedger() declarationRequirementLedger {
	return declarationRequirementLedger{
		byOwner: make(
			map[api.ArtifactOwner]map[api.DeclarationRequirement]struct{},
		),
	}
}

func (l declarationRequirementLedger) contains(
	requirement api.DeclarationRequirement,
) bool {
	requirements := l.byOwner[requirement.Owner()]
	if requirements == nil {
		return false
	}
	_, ok := requirements[requirement]
	return ok
}

func (l declarationRequirementLedger) containsOwner(
	owner api.ArtifactOwner,
) bool {
	return len(l.byOwner[owner]) != 0
}

func (l declarationRequirementLedger) add(
	requirement api.DeclarationRequirement,
) {
	owner := requirement.Owner()
	requirements := l.byOwner[owner]
	if requirements == nil {
		requirements = make(map[api.DeclarationRequirement]struct{})
		l.byOwner[owner] = requirements
	}
	requirements[requirement] = struct{}{}
}

func (l declarationRequirementLedger) forOwner(
	owner api.ArtifactOwner,
) ([]api.DeclarationRequirement, int) {
	selected := l.byOwner[owner]
	requirements := make([]api.DeclarationRequirement, 0, len(selected))
	for requirement := range selected {
		requirements = append(requirements, requirement)
	}
	sortDeclarationRequirements(requirements)
	return requirements, len(selected)
}

func (l declarationRequirementLedger) takeOwner(
	owner api.ArtifactOwner,
) []api.DeclarationRequirement {
	requirements, _ := l.forOwner(owner)
	delete(l.byOwner, owner)
	return requirements
}

func (l declarationRequirementLedger) empty() bool {
	return len(l.byOwner) == 0
}

func sortDeclarationRequirements(requirements []api.DeclarationRequirement) {
	sort.Slice(requirements, func(left, right int) bool {
		return compareDeclarationRequirements(
			requirements[left],
			requirements[right],
		) < 0
	})
}

type artifactOwnerPriorityQueue struct {
	owners []api.ArtifactOwner
}

func (q *artifactOwnerPriorityQueue) push(owner api.ArtifactOwner) {
	q.owners = append(q.owners, owner)
	index := len(q.owners) - 1
	for index > 0 {
		parent := (index - 1) / 2
		if emitordering.CompareArtifactOwners(q.owners[parent], owner) <= 0 {
			break
		}
		q.owners[index] = q.owners[parent]
		index = parent
	}
	q.owners[index] = owner
}

func (q *artifactOwnerPriorityQueue) pop() (api.ArtifactOwner, bool) {
	if len(q.owners) == 0 {
		return api.ArtifactOwner{}, false
	}
	selected := q.owners[0]
	lastIndex := len(q.owners) - 1
	last := q.owners[lastIndex]
	q.owners = q.owners[:lastIndex]
	if len(q.owners) == 0 {
		return selected, true
	}
	index := 0
	for {
		left := index*2 + 1
		if left >= len(q.owners) {
			break
		}
		right := left + 1
		next := left
		if right < len(q.owners) &&
			emitordering.CompareArtifactOwners(
				q.owners[right],
				q.owners[left],
			) < 0 {
			next = right
		}
		if emitordering.CompareArtifactOwners(last, q.owners[next]) <= 0 {
			break
		}
		q.owners[index] = q.owners[next]
		index = next
	}
	q.owners[index] = last
	return selected, true
}

type declarationRequirementScheduler struct {
	pending       declarationRequirementLedger
	pendingOwners artifactOwnerPriorityQueue
	applied       declarationRequirementLedger
}

func newDeclarationRequirementScheduler() *declarationRequirementScheduler {
	return &declarationRequirementScheduler{
		pending: newDeclarationRequirementLedger(),
		applied: newDeclarationRequirementLedger(),
	}
}

func (s *declarationRequirementScheduler) enqueue(
	requirement api.DeclarationRequirement,
) {
	if s.applied.contains(requirement) || s.pending.contains(requirement) {
		return
	}
	owner := requirement.Owner()
	if !s.pending.containsOwner(owner) {
		s.pendingOwners.push(owner)
	}
	s.pending.add(requirement)
}

func (s *declarationRequirementScheduler) nextBatch() (
	[]api.DeclarationRequirement,
	bool,
) {
	owner, ok := s.pendingOwners.pop()
	if !ok {
		return nil, false
	}
	requirements := s.pending.takeOwner(owner)
	if len(requirements) == 0 {
		panic("declaration requirement owner queue lost its requirement bucket")
	}
	for _, requirement := range requirements {
		s.applied.add(requirement)
	}
	return requirements, true
}

func (s *declarationRequirementScheduler) hasPending() bool {
	return !s.pending.empty()
}

func (s *declarationRequirementScheduler) wasApplied(
	requirement api.DeclarationRequirement,
) bool {
	return s.applied.contains(requirement)
}

func (s *declarationRequirementScheduler) appliedFor(
	owner api.ArtifactOwner,
) []api.DeclarationRequirement {
	requirements, _ := s.applied.forOwner(owner)
	return requirements
}
