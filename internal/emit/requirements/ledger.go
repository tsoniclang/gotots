package requirements

import (
	"sort"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

type ledger struct {
	byOwner map[api.ArtifactOwner]map[api.DeclarationRequirement]struct{}
}

func newLedger() ledger {
	return ledger{
		byOwner: make(
			map[api.ArtifactOwner]map[api.DeclarationRequirement]struct{},
		),
	}
}

func (l ledger) contains(requirement api.DeclarationRequirement) bool {
	requirements := l.byOwner[requirement.Owner()]
	if requirements == nil {
		return false
	}
	_, ok := requirements[requirement]
	return ok
}

func (l ledger) containsOwner(owner api.ArtifactOwner) bool {
	return len(l.byOwner[owner]) != 0
}

func (l ledger) add(requirement api.DeclarationRequirement) {
	owner := requirement.Owner()
	requirements := l.byOwner[owner]
	if requirements == nil {
		requirements = make(map[api.DeclarationRequirement]struct{})
		l.byOwner[owner] = requirements
	}
	requirements[requirement] = struct{}{}
}

func (l ledger) remove(requirement api.DeclarationRequirement) {
	owner := requirement.Owner()
	requirements := l.byOwner[owner]
	delete(requirements, requirement)
	if len(requirements) == 0 {
		delete(l.byOwner, owner)
	}
}

func (l ledger) replaceOwner(
	owner api.ArtifactOwner,
	requirements []api.DeclarationRequirement,
) {
	delete(l.byOwner, owner)
	for _, requirement := range requirements {
		l.add(requirement)
	}
}

func (l ledger) forOwner(
	owner api.ArtifactOwner,
	compare func(api.DeclarationRequirement, api.DeclarationRequirement) int,
) ([]api.DeclarationRequirement, int) {
	selected := l.byOwner[owner]
	requirements := make([]api.DeclarationRequirement, 0, len(selected))
	for requirement := range selected {
		requirements = append(requirements, requirement)
	}
	sort.Slice(requirements, func(left, right int) bool {
		return compare(requirements[left], requirements[right]) < 0
	})
	return requirements, len(selected)
}

func (l ledger) empty() bool {
	return len(l.byOwner) == 0
}

type ownerQueue struct {
	owners  []api.ArtifactOwner
	compare func(api.ArtifactOwner, api.ArtifactOwner) int
}

func (q *ownerQueue) push(owner api.ArtifactOwner) {
	q.owners = append(q.owners, owner)
	index := len(q.owners) - 1
	for index > 0 {
		parent := (index - 1) / 2
		if q.compare(q.owners[parent], owner) <= 0 {
			break
		}
		q.owners[index] = q.owners[parent]
		index = parent
	}
	q.owners[index] = owner
}

func (q *ownerQueue) pop() (api.ArtifactOwner, bool) {
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
			q.compare(q.owners[right], q.owners[left]) < 0 {
			next = right
		}
		if q.compare(last, q.owners[next]) <= 0 {
			break
		}
		q.owners[index] = q.owners[next]
		index = next
	}
	q.owners[index] = last
	return selected, true
}
