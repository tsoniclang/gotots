// Package executable owns the Stage-1 grammatical traversal of selected
// full-semantic execution entries. It interprets no go/types fact.
package executable

import (
	"sort"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
	"github.com/tsoniclang/gotots/internal/language/structure"
)

// DefinitionReference records a nested implementation definition used at one
// executable grammatical position.
type DefinitionReference struct {
	parent  identity.OccurrenceID
	edge    catalog.Edge
	ordinal int
	child   identity.DefinitionID
}

func (r DefinitionReference) Parent() identity.OccurrenceID { return r.parent }
func (r DefinitionReference) Edge() catalog.Edge            { return r.edge }
func (r DefinitionReference) Role() catalog.Role            { return r.edge.Role() }
func (r DefinitionReference) Ordinal() int                  { return r.ordinal }
func (r DefinitionReference) Child() identity.DefinitionID  { return r.child }

// ImplicitOperationKind is the closed executable operation vocabulary for
// implicit definitions.
type ImplicitOperationKind uint8

const (
	ImplicitOperationInvalid ImplicitOperationKind = iota
	ImplicitOperationCoordinatePackageInitialization
)

func (k ImplicitOperationKind) Valid() bool {
	return k == ImplicitOperationCoordinatePackageInitialization
}

// ImplicitOperation is one typed operation of an implicit executable region.
type ImplicitOperation struct {
	kind ImplicitOperationKind
	pkg  identity.PackageID
}

func (o ImplicitOperation) Kind() ImplicitOperationKind { return o.kind }
func (o ImplicitOperation) Package() identity.PackageID { return o.pkg }

// Region is one full-semantic definition's exact executable membership graph.
// Occurrence payload is resolved from the structural or additional occurrence
// store and is never copied into the region.
type Region struct {
	id         identity.ExecutableRegionID
	members    []identity.OccurrenceID
	references []DefinitionReference
	implicit   []ImplicitOperation
}

func (r Region) ID() identity.ExecutableRegionID   { return r.id }
func (r Region) Definition() identity.DefinitionID { return r.id.Definition() }
func (r Region) Members() []identity.OccurrenceID {
	return append([]identity.OccurrenceID(nil), r.members...)
}
func (r Region) References() []DefinitionReference {
	return append([]DefinitionReference(nil), r.references...)
}
func (r Region) ImplicitOperations() []ImplicitOperation {
	return append([]ImplicitOperation(nil), r.implicit...)
}

// Work is the executable traversal's measured scalable work.
type Work struct {
	CatalogEdges    int
	BoundaryProbes  int
	RecordAppends   int
	IdentityProbes  int
	JoinProbes      int
	SortComparisons int
}

// Inventory is the immutable exact set of full-semantic regions plus only the
// canonical occurrences absent from the depth-independent structural store.
type Inventory struct {
	regionIDs     []identity.DefinitionID
	byID          map[identity.DefinitionID]Region
	additionalIDs []identity.OccurrenceID
	byOccurrence  map[identity.OccurrenceID]structure.Occurrence
	work          Work
}

func (i *Inventory) Regions() []Region {
	out := make([]Region, 0, len(i.regionIDs))
	for _, id := range i.regionIDs {
		out = append(out, i.byID[id])
	}
	return out
}
func (i *Inventory) AdditionalOccurrences() []structure.Occurrence {
	out := make(
		[]structure.Occurrence, 0, len(i.additionalIDs),
	)
	for _, id := range i.additionalIDs {
		out = append(out, i.byOccurrence[id])
	}
	return out
}
func (i *Inventory) Work() Work { return i.work }
func (i *Inventory) For(
	definition identity.DefinitionID,
) (Region, bool) {
	region, ok := i.byID[definition]
	return region, ok
}
func (i *Inventory) AdditionalOccurrence(
	id identity.OccurrenceID,
) (structure.Occurrence, bool) {
	occurrence, ok := i.byOccurrence[id]
	return occurrence, ok
}

func (i *Inventory) sort() {
	sort.Slice(i.regionIDs, func(left, right int) bool {
		i.work.SortComparisons++
		return i.regionIDs[left].String() <
			i.regionIDs[right].String()
	})
	sort.Slice(i.additionalIDs, func(left, right int) bool {
		i.work.SortComparisons++
		return i.additionalIDs[left].String() <
			i.additionalIDs[right].String()
	})
}
