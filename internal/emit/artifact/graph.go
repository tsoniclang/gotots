package artifact

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

type Contract map[api.ArtifactFacet][]byte

type artifactRecord struct {
	contract       Contract
	history        []Contract
	dependencies   map[api.ArtifactDependency]struct{}
	facetRevisions map[api.ArtifactFacet]uint64
}

type Graph struct {
	records map[api.ArtifactOwner]*artifactRecord
	reverse map[api.ArtifactDependency]map[api.ArtifactOwner]struct{}
	dirty   map[api.ArtifactOwner]struct{}
	compare func(api.ArtifactOwner, api.ArtifactOwner) int
}

type GraphError struct {
	Object   api.ArtifactOwner
	Provider api.ArtifactOwner
	Facet    api.ArtifactFacet
	Reason   string
}

func (e *GraphError) Error() string {
	name := e.Object.Name()
	if e.Provider.Valid() {
		return fmt.Sprintf(
			"coordinate target artifact %q dependency %q/%s: %s",
			name,
			e.Provider.Name(),
			e.Facet,
			e.Reason,
		)
	}
	return fmt.Sprintf("coordinate target artifact %q: %s", name, e.Reason)
}

type ArtifactConvergenceError struct {
	Object api.ArtifactOwner
	Facets []api.ArtifactFacet
}

func (e *ArtifactConvergenceError) Error() string {
	return fmt.Sprintf(
		"reconstruct target artifact %q: observable contract oscillates on facets %v",
		e.Object.Name(),
		e.Facets,
	)
}

func NewGraph(
	compare func(api.ArtifactOwner, api.ArtifactOwner) int,
) *Graph {
	if compare == nil {
		panic("artifact graph comparator is nil")
	}
	return &Graph{
		records: make(map[api.ArtifactOwner]*artifactRecord),
		reverse: make(
			map[api.ArtifactDependency]map[api.ArtifactOwner]struct{},
		),
		dirty:   make(map[api.ArtifactOwner]struct{}),
		compare: compare,
	}
}

func (g *Graph) Commit(
	owner api.ArtifactOwner,
	contract Contract,
	dependencies []api.ArtifactDependency,
) error {
	if !owner.Valid() {
		return &GraphError{Reason: "target artifact owner is invalid"}
	}
	nextContract, err := validateArtifactContract(owner, contract)
	if err != nil {
		return err
	}
	nextDependencies := make(map[api.ArtifactDependency]struct{}, len(dependencies))
	for _, dependency := range dependencies {
		if !dependency.Valid() {
			return &GraphError{
				Object: owner,
				Reason: "target artifact dependency is invalid",
			}
		}
		nextDependencies[dependency] = struct{}{}
	}

	current := g.records[owner]
	changed := changedArtifactFacets(nil, nextContract)
	if current != nil {
		changed = changedArtifactFacets(current.contract, nextContract)
		if len(changed) != 0 {
			for _, historical := range current.history {
				if equalArtifactContracts(historical, nextContract) {
					return &ArtifactConvergenceError{
						Object: owner,
						Facets: append([]api.ArtifactFacet(nil), changed...),
					}
				}
			}
		}
	}

	if current == nil {
		current = &artifactRecord{
			facetRevisions: make(map[api.ArtifactFacet]uint64),
		}
		g.records[owner] = current
	} else {
		g.removeReverseEdges(owner, current.dependencies)
	}
	current.dependencies = nextDependencies
	current.contract = nextContract
	g.addReverseEdges(owner, nextDependencies)

	if len(current.history) == 0 {
		current.history = append(current.history, nextContract)
		for facet := range nextContract {
			current.facetRevisions[facet] = 1
		}
		return nil
	}
	if len(changed) == 0 {
		return nil
	}
	current.history = append(current.history, nextContract)
	for _, facet := range changed {
		current.facetRevisions[facet]++
		dependency, dependencyError := api.NewArtifactDependency(owner, facet)
		if dependencyError != nil {
			return dependencyError
		}
		for consumer := range g.reverse[dependency] {
			g.dirty[consumer] = struct{}{}
		}
	}
	return nil
}

func (g *Graph) removeReverseEdges(
	consumer api.ArtifactOwner,
	dependencies map[api.ArtifactDependency]struct{},
) {
	for dependency := range dependencies {
		consumers := g.reverse[dependency]
		delete(consumers, consumer)
		if len(consumers) == 0 {
			delete(g.reverse, dependency)
		}
	}
}

func (g *Graph) addReverseEdges(
	consumer api.ArtifactOwner,
	dependencies map[api.ArtifactDependency]struct{},
) {
	for dependency := range dependencies {
		consumers := g.reverse[dependency]
		if consumers == nil {
			consumers = make(map[api.ArtifactOwner]struct{})
			g.reverse[dependency] = consumers
		}
		consumers[consumer] = struct{}{}
	}
}

func (g *Graph) NextDirty() (api.ArtifactOwner, bool) {
	var selected api.ArtifactOwner
	found := false
	for object := range g.dirty {
		if !found || g.compare(object, selected) < 0 {
			selected = object
			found = true
		}
	}
	if !found {
		return api.ArtifactOwner{}, false
	}
	delete(g.dirty, selected)
	return selected, true
}

func (g *Graph) DiscardDirty(owner api.ArtifactOwner) {
	delete(g.dirty, owner)
}

func (g *Graph) HasPending() bool {
	return len(g.dirty) != 0
}

func (g *Graph) VerifyClosure() error {
	consumers := make([]api.ArtifactOwner, 0, len(g.records))
	for consumer := range g.records {
		consumers = append(consumers, consumer)
	}
	sort.Slice(consumers, func(left, right int) bool {
		return g.compare(consumers[left], consumers[right]) < 0
	})
	for _, consumer := range consumers {
		record := g.records[consumer]
		dependencies := make(
			[]api.ArtifactDependency,
			0,
			len(record.dependencies),
		)
		for dependency := range record.dependencies {
			dependencies = append(dependencies, dependency)
		}
		sort.Slice(dependencies, func(left, right int) bool {
			order := g.compare(
				dependencies[left].Provider(),
				dependencies[right].Provider(),
			)
			if order != 0 {
				return order < 0
			}
			return dependencies[left].Facet() < dependencies[right].Facet()
		})
		for _, dependency := range dependencies {
			provider := g.records[dependency.Provider()]
			if provider == nil {
				return &GraphError{
					Object:   consumer,
					Provider: dependency.Provider(),
					Facet:    dependency.Facet(),
					Reason:   "artifact dependency provider was not published",
				}
			}
			if _, ok := provider.contract[dependency.Facet()]; !ok {
				return &GraphError{
					Object:   consumer,
					Provider: dependency.Provider(),
					Facet:    dependency.Facet(),
					Reason: fmt.Sprintf(
						"artifact dependency provider has no %s facet",
						dependency.Facet(),
					),
				}
			}
		}
	}
	return nil
}

func (g *Graph) edgeCount() int {
	count := 0
	for _, consumers := range g.reverse {
		count += len(consumers)
	}
	return count
}

func (g *Graph) FacetRevision(
	owner api.ArtifactOwner,
	facet api.ArtifactFacet,
) uint64 {
	record := g.records[owner]
	if record == nil {
		return 0
	}
	return record.facetRevisions[facet]
}

func validateArtifactContract(
	owner api.ArtifactOwner,
	contract Contract,
) (Contract, error) {
	if contract == nil {
		return nil, &GraphError{
			Object: owner,
			Reason: "target artifact observable contract is absent",
		}
	}
	result := make(Contract, len(contract))
	for facet, encoded := range contract {
		if !facet.Valid() {
			return nil, &GraphError{
				Object: owner,
				Facet:  facet,
				Reason: "target artifact observable contract has an invalid facet",
			}
		}
		if len(encoded) == 0 {
			return nil, &GraphError{
				Object: owner,
				Facet:  facet,
				Reason: "target artifact observable contract has an empty facet",
			}
		}
		result[facet] = bytes.Clone(encoded)
	}
	return result, nil
}

func changedArtifactFacets(
	current Contract,
	next Contract,
) []api.ArtifactFacet {
	var changed []api.ArtifactFacet
	for facet := api.ArtifactFacetCallableSignature; facet <= api.ArtifactFacetImplementation; facet++ {
		currentValue, currentOK := current[facet]
		nextValue, nextOK := next[facet]
		if currentOK != nextOK || !bytes.Equal(currentValue, nextValue) {
			changed = append(changed, facet)
		}
	}
	return changed
}

func equalArtifactContracts(left Contract, right Contract) bool {
	return len(changedArtifactFacets(left, right)) == 0
}
