package emit

import (
	"slices"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func (s *declarationRequirementScheduler) consumedBy(
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
	sortDeclarationRequirements(requirements)
	return requirements
}

func (s *declarationRequirementScheduler) consumedRequestsBy(
	owner api.ArtifactOwner,
) []api.RootRequest {
	return slices.Clone(s.byConsumer[owner])
}

func (s *declarationRequirementScheduler) replace(
	consumer api.ArtifactOwner,
	requests []api.RootRequest,
) error {
	selected, err := canonicalDeclarationRequestRoots(requests)
	if err != nil {
		return err
	}
	current := s.byConsumer[consumer]
	currentSet := rootRequestSet(current)
	nextSet := rootRequestSet(selected)
	for request := range nextSet {
		if _, retained := currentSet[request]; retained {
			continue
		}
		s.retainRequest(request)
	}
	for request := range currentSet {
		if _, retained := nextSet[request]; retained {
			continue
		}
		s.releaseRequest(request)
	}
	if len(selected) == 0 {
		delete(s.byConsumer, consumer)
		return nil
	}
	s.byConsumer[consumer] = selected
	return nil
}

func canonicalDeclarationRequestRoots(
	requests []api.RootRequest,
) ([]api.RootRequest, error) {
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

func rootRequestSet(
	requests []api.RootRequest,
) map[api.RootRequest]struct{} {
	result := make(map[api.RootRequest]struct{}, len(requests))
	for _, request := range requests {
		result[request] = struct{}{}
	}
	return result
}

func (s *declarationRequirementScheduler) retainRequest(
	request api.RootRequest,
) {
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

func (s *declarationRequirementScheduler) releaseRequest(
	request api.RootRequest,
) {
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
