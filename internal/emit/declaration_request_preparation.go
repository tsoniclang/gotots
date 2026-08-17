package emit

import "github.com/tsoniclang/gotots/internal/emit/api"

func (s *programSession) prepareDeclarationRequestGraph(
	consumer api.ArtifactOwner,
	requests []api.RootRequest,
) error {
	if s.preparedDeclarationRequests == nil {
		s.preparedDeclarationRequests = make(map[api.RootRequest]struct{})
	}
	for _, request := range requests {
		if err := s.prepareDeclarationRequestNode(consumer, request); err != nil {
			return err
		}
	}
	return nil
}

func (s *programSession) prepareDeclarationRequestNode(
	consumer api.ArtifactOwner,
	request api.RootRequest,
) error {
	if _, prepared := s.preparedDeclarationRequests[request]; prepared {
		return nil
	}
	if children, nested := request.NestedRequests(); nested {
		for _, child := range children {
			if err := s.prepareDeclarationRequestNode(consumer, child); err != nil {
				return err
			}
		}
		s.preparedDeclarationRequests[request] = struct{}{}
		return nil
	}
	requirement, ok := request.DeclarationRequirement()
	if !ok {
		return &ScheduleError{
			Object: consumer.Name(),
			Reason: "declaration request graph contains a non-declaration leaf",
		}
	}
	if err := s.prepareDeclarationRequirement(requirement); err != nil {
		return err
	}
	s.preparedDeclarationRequests[request] = struct{}{}
	return nil
}
