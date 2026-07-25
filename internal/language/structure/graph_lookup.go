package structure

import "github.com/tsoniclang/gotots/internal/identity"

func (g *Graph) Version() int { return g.version }
func (g *Graph) Work() Work   { return g.work }

func (g *Graph) residentOccurrences() []Occurrence {
	out := make([]Occurrence, 0, len(g.occurrenceOrder))
	for _, reference := range g.occurrenceOrder {
		out = append(out, reference.Occurrence())
	}
	return out
}

func (g *Graph) residentOccurrenceRef(
	id identity.OccurrenceID,
) (OccurrenceRef, bool) {
	if g == nil || id.IsZero() {
		return OccurrenceRef{}, false
	}
	store := g.occurrenceStores[id.Span().File()]
	if store == nil {
		return OccurrenceRef{}, false
	}
	return store.reference(id)
}

func (g *Graph) residentOccurrence(
	id identity.OccurrenceID,
) (Occurrence, bool) {
	reference, present := g.residentOccurrenceRef(id)
	if !present {
		return Occurrence{}, false
	}
	return reference.Occurrence(), true
}

func (g *Graph) residentDefinitions() []ImplementationDefinition {
	out := make([]ImplementationDefinition, 0, len(g.definitionIDs))
	for _, id := range g.definitionIDs {
		out = append(out, *g.byDefinition[id])
	}
	return out
}

func (g *Graph) residentDefinition(
	id identity.DefinitionID,
) (ImplementationDefinition, bool) {
	definition, present := g.byDefinition[id]
	if !present {
		return ImplementationDefinition{}, false
	}
	return *definition, true
}

func (g *Graph) residentBoundary(
	id identity.DefinitionID,
) (ExecutionBoundary, bool) {
	boundary, present := g.byBoundary[id]
	if !present {
		return ExecutionBoundary{}, false
	}
	return *boundary, true
}
