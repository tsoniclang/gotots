package emit

import (
	"fmt"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
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

type companionScheduler struct {
	queue   []api.CompanionOwner
	pending map[api.CompanionOwner]struct{}
	emitted map[api.CompanionOwner]struct{}
}

func newCompanionScheduler() *companionScheduler {
	return &companionScheduler{
		pending: make(map[api.CompanionOwner]struct{}),
		emitted: make(map[api.CompanionOwner]struct{}),
	}
}

func (s *companionScheduler) enqueue(owner api.CompanionOwner) {
	if _, done := s.emitted[owner]; done {
		return
	}
	if _, queued := s.pending[owner]; queued {
		return
	}
	s.pending[owner] = struct{}{}
	s.queue = append(s.queue, owner)
}

func (s *companionScheduler) next() (api.CompanionOwner, bool) {
	if len(s.queue) == 0 {
		return api.CompanionOwner{}, false
	}
	owner := s.queue[0]
	s.queue = s.queue[1:]
	delete(s.pending, owner)
	s.emitted[owner] = struct{}{}
	return owner, true
}
