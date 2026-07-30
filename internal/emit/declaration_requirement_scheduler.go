package emit

import (
	"sort"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

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
		if compareArtifactOwners(q.owners[parent], owner) <= 0 {
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
			compareArtifactOwners(q.owners[right], q.owners[left]) < 0 {
			next = right
		}
		if compareArtifactOwners(last, q.owners[next]) <= 0 {
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
