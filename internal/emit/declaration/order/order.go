package order

import (
	"container/heap"
	"go/token"
	"sort"

	"github.com/tsoniclang/gotots/internal/emit/api"
	emitordering "github.com/tsoniclang/gotots/internal/emit/ordering"
)

type Declaration struct {
	Owner             api.ArtifactOwner
	Name              string
	SourcePath        string
	Position          token.Pos
	EagerDependencies []api.ArtifactOwner
}

type SourcePositionError struct {
	Declaration string
}

func (e *SourcePositionError) Error() string {
	return "order target declaration " + e.Declaration +
		": source declaration has no canonical source position"
}

type CycleError struct {
	Declaration string
}

func (e *CycleError) Error() string {
	return "order target declaration " + e.Declaration +
		": eager dependency cycle"
}

func Indices(declarations []Declaration) ([]int, error) {
	indexByOwner := make(map[api.ArtifactOwner]int, len(declarations))
	for index := range declarations {
		if _, sourceOwned := declarations[index].Owner.Source(); sourceOwned &&
			(declarations[index].SourcePath == "" ||
				declarations[index].Position == token.NoPos) {
			return nil, &SourcePositionError{
				Declaration: declarations[index].Name,
			}
		}
		indexByOwner[declarations[index].Owner] = index
	}
	dependents := make([][]int, len(declarations))
	indegree := make([]int, len(declarations))
	edges := make(map[[2]int]struct{})
	for consumer := range declarations {
		for _, dependency := range declarations[consumer].EagerDependencies {
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
	ordered := make([]int, 0, len(declarations))
	for ready.Len() != 0 {
		provider := heap.Pop(ready).(int)
		ordered = append(ordered, provider)
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
			cycle = append(cycle, declarations[index].Name)
		}
	}
	sort.Strings(cycle)
	return nil, &CycleError{Declaration: cycle[0]}
}

type declarationHeap struct {
	declarations []Declaration
	indices      []int
}

func (h declarationHeap) Len() int {
	return len(h.indices)
}

func (h declarationHeap) Less(left, right int) bool {
	leftDeclaration := h.declarations[h.indices[left]]
	rightDeclaration := h.declarations[h.indices[right]]
	if leftDeclaration.SourcePath != rightDeclaration.SourcePath {
		return leftDeclaration.SourcePath < rightDeclaration.SourcePath
	}
	if leftDeclaration.Position != rightDeclaration.Position {
		return leftDeclaration.Position < rightDeclaration.Position
	}
	if leftDeclaration.Name != rightDeclaration.Name {
		return leftDeclaration.Name < rightDeclaration.Name
	}
	return emitordering.CompareArtifactOwners(
		leftDeclaration.Owner,
		rightDeclaration.Owner,
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
