package artifact

import (
	"sort"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func (g *Graph) Owners() []api.ArtifactOwner {
	result := make([]api.ArtifactOwner, 0, len(g.records))
	for owner := range g.records {
		result = append(result, owner)
	}
	sort.Slice(result, func(left, right int) bool {
		return g.compare(result[left], result[right]) < 0
	})
	return result
}

func (g *Graph) ObservableContract(
	owner api.ArtifactOwner,
) (Contract, error) {
	if !owner.Valid() {
		return Contract{}, &GraphError{Reason: "artifact contract owner is invalid"}
	}
	record := g.records[owner]
	if record == nil {
		return Contract{}, &GraphError{
			Object: owner,
			Reason: "artifact contract was not published",
		}
	}
	return record.contract.withoutImplementation(), nil
}
