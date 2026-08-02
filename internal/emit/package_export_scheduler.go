package emit

import "sort"

type packageExportScheduler struct {
	pending map[*packageTargetBuilder]struct{}
}

func newPackageExportScheduler() *packageExportScheduler {
	return &packageExportScheduler{
		pending: make(map[*packageTargetBuilder]struct{}),
	}
}

func (s *packageExportScheduler) enqueue(builder *packageTargetBuilder) {
	if builder == nil {
		panic("package export builder is nil")
	}
	s.pending[builder] = struct{}{}
}

func (s *packageExportScheduler) nextBatch() []*packageTargetBuilder {
	if len(s.pending) == 0 {
		return nil
	}
	builders := make([]*packageTargetBuilder, 0, len(s.pending))
	for builder := range s.pending {
		builders = append(builders, builder)
	}
	clear(s.pending)
	sort.Slice(builders, func(left, right int) bool {
		return builders[left].assemblyPath < builders[right].assemblyPath
	})
	return builders
}

func (s *packageExportScheduler) hasPending() bool {
	return len(s.pending) != 0
}
