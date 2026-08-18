package requirements

import (
	"slices"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

type Scheduler struct {
	pendingOwners   ownerQueue
	pending         map[api.ArtifactOwner]struct{}
	removed         map[api.ArtifactOwner]struct{}
	orphaned        map[api.DeclarationRequirement]struct{}
	certified       ledger
	roots           ledger
	byConsumer      map[api.ArtifactOwner][]api.RootRequest
	requestRefs     map[api.RootRequest]uint64
	requirementRefs map[api.DeclarationRequirement]uint64
	active          ledger
	applied         ledger
	compare         func(api.DeclarationRequirement, api.DeclarationRequirement) int
}

func New(
	compareOwners func(api.ArtifactOwner, api.ArtifactOwner) int,
	compareRequirements func(
		api.DeclarationRequirement,
		api.DeclarationRequirement,
	) int,
) *Scheduler {
	if compareOwners == nil || compareRequirements == nil {
		panic("declaration requirement scheduler has no ordering owner")
	}
	return &Scheduler{
		pendingOwners: ownerQueue{compare: compareOwners},
		pending:       make(map[api.ArtifactOwner]struct{}),
		removed:       make(map[api.ArtifactOwner]struct{}),
		orphaned:      make(map[api.DeclarationRequirement]struct{}),
		certified:     newLedger(),
		roots:         newLedger(),
		byConsumer:    make(map[api.ArtifactOwner][]api.RootRequest),
		requestRefs:   make(map[api.RootRequest]uint64),
		requirementRefs: make(
			map[api.DeclarationRequirement]uint64,
		),
		active:  newLedger(),
		applied: newLedger(),
		compare: compareRequirements,
	}
}

func (s *Scheduler) Enqueue(requirement api.DeclarationRequirement) {
	if s.roots.contains(requirement) {
		return
	}
	s.roots.add(requirement)
	if !s.active.contains(requirement) {
		s.active.add(requirement)
		s.enqueueOwner(requirement.Owner())
	}
}

func (s *Scheduler) NextBatch() (
	api.ArtifactOwner,
	[]api.DeclarationRequirement,
	bool,
	bool,
) {
	owner, ok := s.pendingOwners.pop()
	if !ok {
		return api.ArtifactOwner{}, nil, false, false
	}
	delete(s.pending, owner)
	_, removed := s.removed[owner]
	delete(s.removed, owner)
	requirements, _ := s.active.forOwner(owner, s.compare)
	s.applied.replaceOwner(owner, requirements)
	return owner, requirements, removed, true
}

func (s *Scheduler) HasPending() bool {
	return len(s.pending) != 0 || len(s.orphaned) != 0
}

func (s *Scheduler) WasApplied(requirement api.DeclarationRequirement) bool {
	return s.applied.contains(requirement)
}

func (s *Scheduler) AppliedFor(
	owner api.ArtifactOwner,
) []api.DeclarationRequirement {
	requirements, _ := s.applied.forOwner(owner, s.compare)
	return requirements
}

func (s *Scheduler) AppliedForWithWork(
	owner api.ArtifactOwner,
) ([]api.DeclarationRequirement, int) {
	return s.applied.forOwner(owner, s.compare)
}

func (s *Scheduler) WasSelected(requirement api.DeclarationRequirement) bool {
	return s.certified.contains(requirement) || s.WasApplied(requirement)
}

func (s *Scheduler) SelectedFor(
	owner api.ArtifactOwner,
) []api.DeclarationRequirement {
	certified, _ := s.certified.forOwner(owner, s.compare)
	applied := s.AppliedFor(owner)
	return merge(certified, applied, s.compare)
}

func merge(
	left []api.DeclarationRequirement,
	right []api.DeclarationRequirement,
	compare func(api.DeclarationRequirement, api.DeclarationRequirement) int,
) []api.DeclarationRequirement {
	if len(left) == 0 {
		return right
	}
	if len(right) == 0 {
		return left
	}
	result := make(
		[]api.DeclarationRequirement,
		0,
		len(left)+len(right),
	)
	leftIndex := 0
	rightIndex := 0
	for leftIndex < len(left) && rightIndex < len(right) {
		switch order := compare(left[leftIndex], right[rightIndex]); {
		case order < 0:
			result = append(result, left[leftIndex])
			leftIndex++
		case order > 0:
			result = append(result, right[rightIndex])
			rightIndex++
		default:
			result = append(result, left[leftIndex])
			leftIndex++
			rightIndex++
		}
	}
	result = append(result, left[leftIndex:]...)
	result = append(result, right[rightIndex:]...)
	return result
}

func (s *Scheduler) ConsumedBy(
	owner api.ArtifactOwner,
) []api.DeclarationRequirement {
	selected := make(map[api.DeclarationRequirement]struct{})
	err := api.WalkUniqueRootRequestPayloads(
		s.byConsumer[owner],
		func(request api.RootRequest) error {
			requirement, ok := request.DeclarationRequirement()
			if ok {
				selected[requirement] = struct{}{}
			}
			return nil
		},
	)
	if err != nil {
		panic(err)
	}
	requirements := make(
		[]api.DeclarationRequirement,
		0,
		len(selected),
	)
	for requirement := range selected {
		requirements = append(requirements, requirement)
	}
	slices.SortFunc(requirements, s.compare)
	return requirements
}

func (s *Scheduler) ConsumedRequestsBy(
	owner api.ArtifactOwner,
) []api.RootRequest {
	return slices.Clone(s.byConsumer[owner])
}

func (s *Scheduler) Replace(
	consumer api.ArtifactOwner,
	requests []api.RootRequest,
) error {
	selected, err := canonicalRoots(requests)
	if err != nil {
		return err
	}
	current := s.byConsumer[consumer]
	currentSet := requestSet(current)
	nextSet := requestSet(selected)
	for request := range nextSet {
		if _, retained := currentSet[request]; !retained {
			s.retainRequest(request)
		}
	}
	for request := range currentSet {
		if _, retained := nextSet[request]; !retained {
			s.releaseRequest(request)
		}
	}
	if len(selected) == 0 {
		delete(s.byConsumer, consumer)
		return nil
	}
	s.byConsumer[consumer] = selected
	return nil
}

func canonicalRoots(requests []api.RootRequest) ([]api.RootRequest, error) {
	selected, err := api.SelectDeclarationRequests(requests)
	if err != nil {
		return nil, err
	}
	seen := make(map[api.RootRequest]struct{}, len(selected))
	result := make([]api.RootRequest, 0, len(selected))
	for _, request := range selected {
		if _, duplicate := seen[request]; duplicate {
			continue
		}
		seen[request] = struct{}{}
		result = append(result, request)
	}
	return result, nil
}

func requestSet(requests []api.RootRequest) map[api.RootRequest]struct{} {
	result := make(map[api.RootRequest]struct{}, len(requests))
	for _, request := range requests {
		result[request] = struct{}{}
	}
	return result
}

func (s *Scheduler) retainRequest(request api.RootRequest) {
	references := s.requestRefs[request]
	s.requestRefs[request] = references + 1
	if references != 0 {
		return
	}
	if children, nested := request.NestedRequests(); nested {
		for _, child := range children {
			s.retainRequest(child)
		}
		return
	}
	requirement, ok := request.DeclarationRequirement()
	if !ok {
		panic("non-declaration request entered declaration liveness")
	}
	references = s.requirementRefs[requirement]
	s.requirementRefs[requirement] = references + 1
	if references != 0 {
		return
	}
	_, orphaned := s.orphaned[requirement]
	delete(s.orphaned, requirement)
	if !orphaned && !s.active.contains(requirement) {
		s.active.add(requirement)
		s.enqueueOwner(requirement.Owner())
	}
}

func (s *Scheduler) releaseRequest(request api.RootRequest) {
	references := s.requestRefs[request]
	if references == 0 {
		panic("declaration request released without liveness")
	}
	if references != 1 {
		s.requestRefs[request] = references - 1
		return
	}
	delete(s.requestRefs, request)
	if children, nested := request.NestedRequests(); nested {
		for _, child := range children {
			s.releaseRequest(child)
		}
		return
	}
	requirement, ok := request.DeclarationRequirement()
	if !ok {
		panic("non-declaration request left declaration liveness")
	}
	references = s.requirementRefs[requirement]
	if references == 0 {
		panic("declaration requirement released without liveness")
	}
	if references != 1 {
		s.requirementRefs[requirement] = references - 1
		return
	}
	delete(s.requirementRefs, requirement)
	if !s.roots.contains(requirement) {
		s.orphaned[requirement] = struct{}{}
	}
}

func (s *Scheduler) FinalizeRemovals() bool {
	if len(s.pending) != 0 {
		panic("declaration requirement removal finalized before additions settled")
	}
	if len(s.orphaned) == 0 {
		return false
	}
	owners := make(map[api.ArtifactOwner]struct{})
	for requirement := range s.orphaned {
		delete(s.orphaned, requirement)
		if s.roots.contains(requirement) ||
			s.requirementRefs[requirement] != 0 {
			continue
		}
		s.active.remove(requirement)
		owners[requirement.Owner()] = struct{}{}
	}
	for owner := range owners {
		s.removed[owner] = struct{}{}
		s.enqueueOwner(owner)
	}
	return len(owners) != 0
}

func (s *Scheduler) enqueueOwner(owner api.ArtifactOwner) {
	if _, queued := s.pending[owner]; queued {
		return
	}
	s.pending[owner] = struct{}{}
	s.pendingOwners.push(owner)
}

func (s *Scheduler) CertifiedEmpty() bool {
	return s.certified.empty()
}

func (s *Scheduler) CertifiedContains(
	requirement api.DeclarationRequirement,
) bool {
	return s.certified.contains(requirement)
}

func (s *Scheduler) InstallCertified(
	requirements []api.DeclarationRequirement,
) bool {
	if !s.certified.empty() {
		return false
	}
	selected := newLedger()
	for _, requirement := range requirements {
		if selected.contains(requirement) {
			return false
		}
		selected.add(requirement)
	}
	s.certified = selected
	return true
}
