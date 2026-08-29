package emit

import (
	"sort"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func (s *programSession) generatedRepresentationTransports() (
	[]api.GeneratedRepresentationTransport,
	error,
) {
	if s == nil || s.registry == nil {
		return nil, &ScheduleError{Reason: "generated representation transport owner is absent"}
	}
	transports := make([]api.GeneratedRepresentationTransport, 0, len(s.generatedGenericKernels))
	for owner := range s.generatedGenericKernels {
		transport, err := s.registry.GeneratedGenericKernelTransport(owner)
		if err != nil {
			return nil, err
		}
		transports = append(transports, transport)
	}
	sort.Slice(transports, func(left, right int) bool {
		return transports[left].Key() < transports[right].Key()
	})
	for index := 1; index < len(transports); index++ {
		if transports[index-1].Key() == transports[index].Key() {
			return nil, &ScheduleError{Reason: "generated representation transport is duplicated"}
		}
	}
	return transports, nil
}
