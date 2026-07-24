// Package executable owns the Stage-1 grammatical traversal of selected
// full-semantic execution entries. It interprets no go/types fact.
package executable

import (
	"fmt"
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
	id          identity.ExecutableRegionID
	occurrences *occurrenceStore
	members     []occurrenceRef
	references  []DefinitionReference
	implicit    []ImplicitOperation
}

func (r Region) ID() identity.ExecutableRegionID   { return r.id }
func (r Region) Definition() identity.DefinitionID { return r.id.Definition() }
func (r Region) Members() []identity.OccurrenceID {
	out := make([]identity.OccurrenceID, 0, len(r.members))
	for _, reference := range r.members {
		out = append(out, r.occurrences.mustID(reference))
	}
	return out
}
func (r Region) MemberCount() int { return len(r.members) }
func (r Region) VisitMembers(
	visit func(int, identity.OccurrenceID) error,
) error {
	if visit == nil {
		return fmt.Errorf("executable member visit requires a visitor")
	}
	for ordinal, reference := range r.members {
		member, err := r.occurrences.id(reference)
		if err != nil {
			return err
		}
		if err := visit(ordinal, member); err != nil {
			return err
		}
	}
	return nil
}
func (r Region) References() []DefinitionReference {
	return append([]DefinitionReference(nil), r.references...)
}
func (r Region) VisitReferences(
	visit func(DefinitionReference) error,
) error {
	if visit == nil {
		return fmt.Errorf("executable reference visit requires a visitor")
	}
	for _, reference := range r.references {
		if err := visit(reference); err != nil {
			return err
		}
	}
	return nil
}
func (r Region) ImplicitOperations() []ImplicitOperation {
	return append([]ImplicitOperation(nil), r.implicit...)
}
func (r Region) VisitImplicitOperations(
	visit func(ImplicitOperation) error,
) error {
	if visit == nil {
		return fmt.Errorf(
			"executable implicit-operation visit requires a visitor",
		)
	}
	for _, operation := range r.implicit {
		if err := visit(operation); err != nil {
			return err
		}
	}
	return nil
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
	regionIDs        []identity.DefinitionID
	byID             map[identity.DefinitionID]Region
	additional       []occurrenceRef
	additionalByFile map[identity.FileID]occurrenceRange
	occurrences      *occurrenceStore
	work             Work
}

type occurrenceRange struct {
	start int
	end   int
}

func (i *Inventory) Regions() []Region {
	out := make([]Region, 0, len(i.regionIDs))
	for _, id := range i.regionIDs {
		out = append(out, i.byID[id])
	}
	return out
}
func (i *Inventory) RegionCount() int { return len(i.regionIDs) }
func (i *Inventory) AdditionalOccurrences() []structure.Occurrence {
	out := make(
		[]structure.Occurrence, 0, len(i.additional),
	)
	for _, reference := range i.additional {
		out = append(out, *i.occurrences.payloadFor(reference))
	}
	return out
}
func (i *Inventory) AdditionalOccurrenceCount() int {
	return len(i.additional)
}
func (i *Inventory) VisitAdditionalOccurrenceRefsForFile(
	file identity.FileID,
	visit func(structure.OccurrenceRef) error,
) error {
	if file.IsZero() || visit == nil {
		return fmt.Errorf(
			"additional occurrence visit requires a file and visitor",
		)
	}
	indexed := i.additionalByFile[file]
	for _, stored := range i.additional[indexed.start:indexed.end] {
		reference, err := structure.NewOccurrenceRef(
			i.occurrences.payloadFor(stored),
		)
		if err != nil {
			return err
		}
		if err := visit(reference); err != nil {
			return err
		}
	}
	return nil
}
func (i *Inventory) AdditionalOccurrenceRefs() (
	[]structure.OccurrenceRef,
	error,
) {
	out := make(
		[]structure.OccurrenceRef, 0, len(i.additional),
	)
	for _, stored := range i.additional {
		reference, err := structure.NewOccurrenceRef(
			i.occurrences.payloadFor(stored),
		)
		if err != nil {
			return nil, err
		}
		out = append(out, reference)
	}
	return out, nil
}

func (i *Inventory) AdditionalOccurrenceRefsForFiles(
	files []identity.FileID,
) ([]structure.OccurrenceRef, error) {
	var out []structure.OccurrenceRef
	err := i.visitAdditionalOccurrenceIDsForFiles(
		files,
		func(id identity.OccurrenceID) error {
			occurrence, present := i.occurrences.get(id)
			if !present {
				return fmt.Errorf(
					"additional occurrence %s has no canonical payload",
					id,
				)
			}
			reference, err := structure.NewOccurrenceRef(occurrence)
			if err != nil {
				return err
			}
			out = append(out, reference)
			return nil
		},
	)
	return out, err
}

func (i *Inventory) AdditionalOccurrenceCountForFiles(
	files []identity.FileID,
) (int, error) {
	count := 0
	err := i.visitAdditionalOccurrenceIDsForFiles(
		files,
		func(identity.OccurrenceID) error {
			count++
			return nil
		},
	)
	return count, err
}

func (i *Inventory) visitAdditionalOccurrenceIDsForFiles(
	files []identity.FileID,
	visit func(identity.OccurrenceID) error,
) error {
	if i == nil || visit == nil {
		return fmt.Errorf(
			"additional occurrence lookup requires inventory and visitor",
		)
	}
	ordered := append([]identity.FileID(nil), files...)
	sort.Slice(ordered, func(left, right int) bool {
		return ordered[left].Compare(ordered[right]) < 0
	})
	previous := identity.FileID{}
	for _, file := range ordered {
		if file.IsZero() {
			return fmt.Errorf(
				"additional occurrence lookup has a zero file identity",
			)
		}
		if file == previous {
			return fmt.Errorf(
				"additional occurrence lookup repeats file %s", file,
			)
		}
		previous = file
		indexed := i.additionalByFile[file]
		for _, stored := range i.additional[indexed.start:indexed.end] {
			id := i.occurrences.mustID(stored)
			if err := visit(id); err != nil {
				return err
			}
		}
	}
	return nil
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
	occurrence, ok := i.occurrences.get(id)
	if !ok {
		return structure.Occurrence{}, false
	}
	return *occurrence, true
}
func (i *Inventory) AdditionalOccurrenceRef(
	id identity.OccurrenceID,
) (structure.OccurrenceRef, bool, error) {
	occurrence, ok := i.occurrences.get(id)
	if !ok {
		return structure.OccurrenceRef{}, false, nil
	}
	reference, err := structure.NewOccurrenceRef(occurrence)
	if err != nil {
		return structure.OccurrenceRef{}, false, err
	}
	return reference, true, nil
}

func (i *Inventory) sort() {
	sort.Slice(i.regionIDs, func(left, right int) bool {
		i.work.SortComparisons++
		return i.regionIDs[left].Compare(i.regionIDs[right]) < 0
	})
	sort.Slice(i.additional, func(left, right int) bool {
		i.work.SortComparisons++
		return i.occurrences.compare(
			i.additional[left], i.additional[right],
		) < 0
	})
	i.additionalByFile = map[identity.FileID]occurrenceRange{}
	for index, reference := range i.additional {
		id := i.occurrences.mustID(reference)
		file := id.Span().File()
		indexed, present := i.additionalByFile[file]
		if !present {
			indexed.start = index
		}
		indexed.end = index + 1
		i.additionalByFile[file] = indexed
	}
}
