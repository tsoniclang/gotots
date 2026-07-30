package artifact

import (
	"fmt"
	"sort"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

type artifactRecord struct {
	contract       Contract
	history        contractHistory
	dependencies   map[api.ArtifactDependency]struct{}
	facetRevisions map[api.ArtifactFacet]uint64
}

type Graph struct {
	records map[api.ArtifactOwner]*artifactRecord
	reverse map[api.ArtifactDependency]map[api.ArtifactOwner]struct{}
	dirty   artifactOwnerQueue
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
		dirty:   newArtifactOwnerQueue(compare),
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
	var previousContract Contract
	changed := changedArtifactFacets(Contract{}, nextContract)
	if current != nil {
		previousContract = current.contract
		changed = changedArtifactFacets(current.contract, nextContract)
		if len(changed) != 0 {
			if current.history.contains(current.contract, nextContract) {
				return &ArtifactConvergenceError{
					Object: owner,
					Facets: append([]api.ArtifactFacet(nil), changed...),
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

	if len(current.history.entries) == 0 {
		current.history.initialize(nextContract)
		for facet := api.ArtifactFacetCallableSignature; facet <= api.ArtifactFacetImplementation; facet++ {
			if !nextContract.hasFacet(facet) {
				continue
			}
			current.facetRevisions[facet] = 1
		}
		return g.invalidateConsumers(owner, changed)
	}
	if len(changed) == 0 {
		return nil
	}
	current.history.append(previousContract, nextContract)
	for _, facet := range changed {
		current.facetRevisions[facet]++
	}
	return g.invalidateConsumers(owner, changed)
}

func (g *Graph) invalidateConsumers(
	owner api.ArtifactOwner,
	facets []api.ArtifactFacet,
) error {
	for _, facet := range facets {
		dependency, dependencyError := api.NewArtifactDependency(owner, facet)
		if dependencyError != nil {
			return dependencyError
		}
		for consumer := range g.reverse[dependency] {
			g.dirty.push(consumer)
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
	return g.dirty.pop()
}

func (g *Graph) DiscardDirty(owner api.ArtifactOwner) {
	g.dirty.discard(owner)
}

func (g *Graph) HasPending() bool {
	return g.dirty.pending()
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
			if !provider.contract.hasFacet(dependency.Facet()) {
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

func (g *Graph) ExportedBindings(
	owner api.ArtifactOwner,
) ([]string, bool) {
	record := g.records[owner]
	if record == nil {
		return nil, false
	}
	return record.contract.ExportedBindings()
}
