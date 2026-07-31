package artifact

import "github.com/tsoniclang/gotots/internal/emit/api"

func (g *Graph) DirtyBatch() []api.ArtifactOwner {
	if !g.dirty.pending() {
		return nil
	}
	pending := make(map[api.ArtifactOwner]struct{}, len(g.dirty.indices))
	indegree := make(map[api.ArtifactOwner]int, len(g.dirty.indices))
	consumers := make(
		map[api.ArtifactOwner][]api.ArtifactOwner,
		len(g.dirty.indices),
	)
	remaining := newArtifactOwnerQueue(g.compare)
	ready := newArtifactOwnerQueue(g.compare)
	for owner := range g.dirty.indices {
		pending[owner] = struct{}{}
		remaining.push(owner)
	}
	for consumer := range pending {
		record := g.records[consumer]
		if record == nil {
			continue
		}
		providers := make(map[api.ArtifactOwner]struct{})
		for dependency := range record.dependencies {
			provider := dependency.Provider()
			if _, dirty := pending[provider]; !dirty {
				continue
			}
			providers[provider] = struct{}{}
		}
		indegree[consumer] = len(providers)
		for provider := range providers {
			consumers[provider] = append(consumers[provider], consumer)
		}
	}
	for owner := range pending {
		if indegree[owner] == 0 {
			ready.push(owner)
		}
	}

	result := make([]api.ArtifactOwner, 0, len(pending))
	for len(pending) != 0 {
		selected, ok := popPendingOwner(&ready, pending)
		if !ok {
			selected, ok = popPendingOwner(&remaining, pending)
			if !ok {
				panic("dirty artifact batch lost its pending owner")
			}
		}
		delete(pending, selected)
		result = append(result, selected)
		for _, consumer := range consumers[selected] {
			if _, stillPending := pending[consumer]; !stillPending {
				continue
			}
			indegree[consumer]--
			if indegree[consumer] == 0 {
				ready.push(consumer)
			}
		}
	}
	return result
}

func popPendingOwner(
	queue *artifactOwnerQueue,
	pending map[api.ArtifactOwner]struct{},
) (api.ArtifactOwner, bool) {
	for {
		owner, ok := queue.pop()
		if !ok {
			return api.ArtifactOwner{}, false
		}
		if _, exists := pending[owner]; exists {
			return owner, true
		}
	}
}
