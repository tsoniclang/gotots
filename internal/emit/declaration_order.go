package emit

import (
	"container/heap"
	"sort"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func orderTargetDeclarations(
	declarations []targetDeclaration,
) ([]targetDeclaration, error) {
	indexByOwner := make(map[api.ArtifactOwner]int, len(declarations))
	for index := range declarations {
		indexByOwner[declarations[index].owner] = index
	}
	dependents := make([][]int, len(declarations))
	indegree := make([]int, len(declarations))
	edges := make(map[[2]int]struct{})
	for consumer := range declarations {
		for _, dependency := range declarations[consumer].eagerDependencies {
			provider, local := indexByOwner[dependency]
			if !local || provider == consumer {
				continue
			}
			edge := [2]int{provider, consumer}
			if _, duplicate := edges[edge]; duplicate {
				continue
			}
			edges[edge] = struct{}{}
			dependents[provider] = append(dependents[provider], consumer)
			indegree[consumer]++
		}
	}
	ready := &declarationHeap{declarations: declarations}
	for index := range declarations {
		if indegree[index] == 0 {
			heap.Push(ready, index)
		}
	}
	ordered := make([]targetDeclaration, 0, len(declarations))
	for ready.Len() != 0 {
		provider := heap.Pop(ready).(int)
		ordered = append(ordered, declarations[provider])
		for _, consumer := range dependents[provider] {
			indegree[consumer]--
			if indegree[consumer] == 0 {
				heap.Push(ready, consumer)
			}
		}
	}
	if len(ordered) == len(declarations) {
		return ordered, nil
	}
	var cycle []string
	for index, remaining := range indegree {
		if remaining != 0 {
			cycle = append(cycle, declarations[index].name)
		}
	}
	sort.Strings(cycle)
	return nil, &ScheduleError{
		Object: cycle[0],
		Reason: "eager target declaration dependency cycle",
	}
}

type declarationHeap struct {
	declarations []targetDeclaration
	indices      []int
}

func (h declarationHeap) Len() int {
	return len(h.indices)
}

func (h declarationHeap) Less(left, right int) bool {
	leftDeclaration := h.declarations[h.indices[left]]
	rightDeclaration := h.declarations[h.indices[right]]
	if leftDeclaration.position != rightDeclaration.position {
		return leftDeclaration.position < rightDeclaration.position
	}
	if leftDeclaration.name != rightDeclaration.name {
		return leftDeclaration.name < rightDeclaration.name
	}
	return compareArtifactOwners(
		leftDeclaration.owner,
		rightDeclaration.owner,
	) < 0
}

func (h declarationHeap) Swap(left, right int) {
	h.indices[left], h.indices[right] = h.indices[right], h.indices[left]
}

func (h *declarationHeap) Push(value any) {
	h.indices = append(h.indices, value.(int))
}

func (h *declarationHeap) Pop() any {
	last := len(h.indices) - 1
	value := h.indices[last]
	h.indices = h.indices[:last]
	return value
}
