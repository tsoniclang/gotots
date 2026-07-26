package emit

import (
	"fmt"
	"go/types"
)

type scheduler struct {
	queue   []types.Object
	pending map[types.Object]struct{}
	emitted map[types.Object]struct{}
}

type ScheduleError struct {
	Object string
	Reason string
}

func (e *ScheduleError) Error() string {
	if e.Object == "" {
		return "schedule declaration: " + e.Reason
	}
	return fmt.Sprintf("schedule declaration %q: %s", e.Object, e.Reason)
}

func newScheduler() *scheduler {
	return &scheduler{
		pending: make(map[types.Object]struct{}),
		emitted: make(map[types.Object]struct{}),
	}
}

func (s *scheduler) enqueue(object types.Object) {
	if _, done := s.emitted[object]; done {
		return
	}
	if _, queued := s.pending[object]; queued {
		return
	}
	s.pending[object] = struct{}{}
	s.queue = append(s.queue, object)
}

func (s *scheduler) next() (types.Object, bool) {
	if len(s.queue) == 0 {
		return nil, false
	}
	object := s.queue[0]
	s.queue = s.queue[1:]
	delete(s.pending, object)
	s.emitted[object] = struct{}{}
	return object, true
}
