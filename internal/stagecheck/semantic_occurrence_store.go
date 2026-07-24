package stagecheck

import (
	"fmt"
	"go/ast"
	"sort"

	"github.com/tsoniclang/gotots/internal/identity"
)

type semanticOccurrenceKey struct {
	file  uint32
	start int
	end   int
	kind  uint16
}

type semanticOccurrenceRef uint32

func (reference semanticOccurrenceRef) valid() bool {
	return reference != 0
}

type semanticOccurrenceRange struct {
	start uint32
	count uint32
}

type semanticOccurrenceStore struct {
	files       map[identity.FileID]uint32
	fileIDs     []identity.FileID
	nextFile    uint32
	byIdentity  map[semanticOccurrenceKey]semanticOccurrenceRef
	keys        []semanticOccurrenceKey
	records     []semanticExpectedOccurrence
	byNode      map[ast.Node]semanticOccurrenceRef
	childRanges []semanticOccurrenceRange
	children    []semanticOccurrenceRef
	active      int
}

func newSemanticOccurrenceStore(capacity int) *semanticOccurrenceStore {
	if capacity < 0 {
		panic("semantic expectation occurrence store has negative capacity")
	}
	return &semanticOccurrenceStore{
		files: map[identity.FileID]uint32{},
		byIdentity: make(
			map[semanticOccurrenceKey]semanticOccurrenceRef,
			capacity,
		),
		records: make(
			[]semanticExpectedOccurrence, 0, capacity,
		),
		byNode: make(
			map[ast.Node]semanticOccurrenceRef,
			capacity,
		),
	}
}

func (store *semanticOccurrenceStore) key(
	id identity.OccurrenceID,
	admitFile bool,
) (semanticOccurrenceKey, bool) {
	if store == nil || id.IsZero() {
		return semanticOccurrenceKey{}, false
	}
	file := id.Span().File()
	fileRef, present := store.files[file]
	if !present {
		if !admitFile {
			return semanticOccurrenceKey{}, false
		}
		store.nextFile++
		if store.nextFile == 0 {
			panic("semantic expectation file table overflow")
		}
		fileRef = store.nextFile
		store.files[file] = fileRef
		store.fileIDs = append(store.fileIDs, file)
	}
	return semanticOccurrenceKey{
		file:  fileRef,
		start: id.Span().Start(),
		end:   id.Span().End(),
		kind:  id.KindID(),
	}, true
}

func (store *semanticOccurrenceStore) get(
	id identity.OccurrenceID,
) (*semanticExpectedOccurrence, bool) {
	record := store.record(store.reference(id))
	return record, record != nil && !record.ID().IsZero()
}

func (store *semanticOccurrenceStore) reference(
	id identity.OccurrenceID,
) semanticOccurrenceRef {
	reference := store.identityReference(id)
	record := store.record(reference)
	if record == nil || record.ID().IsZero() {
		return 0
	}
	return reference
}

func (store *semanticOccurrenceStore) identityReference(
	id identity.OccurrenceID,
) semanticOccurrenceRef {
	key, present := store.key(id, false)
	if !present {
		return 0
	}
	return store.byIdentity[key]
}

func (store *semanticOccurrenceStore) record(
	reference semanticOccurrenceRef,
) *semanticExpectedOccurrence {
	if store == nil ||
		!reference.valid() ||
		int(reference) > len(store.records) {
		return nil
	}
	return &store.records[reference-1]
}

func (store *semanticOccurrenceStore) put(
	id identity.OccurrenceID,
	record *semanticExpectedOccurrence,
) (semanticOccurrenceRef, error) {
	if record == nil || record.ID() != id {
		return 0, fmt.Errorf(
			"semantic occurrence store requires identity-aligned record",
		)
	}
	key, present := store.key(id, true)
	if !present {
		return 0, fmt.Errorf(
			"semantic occurrence store rejects zero identity",
		)
	}
	if reference := store.byIdentity[key]; reference.valid() {
		existing := store.record(reference)
		if existing.ID().IsZero() {
			*existing = *record
			store.active++
			return reference, nil
		}
		if existing.OccurrenceRef.Equal(record.OccurrenceRef) {
			return reference, nil
		}
		return 0, fmt.Errorf(
			"semantic occurrence key collides for %s and %s",
			existing.ID(),
			id,
		)
	}
	reference, err := store.admitIdentity(key)
	if err != nil {
		return 0, err
	}
	store.records[reference-1] = *record
	store.active++
	return reference, nil
}

func (store *semanticOccurrenceStore) admitIdentity(
	key semanticOccurrenceKey,
) (semanticOccurrenceRef, error) {
	if reference := store.byIdentity[key]; reference.valid() {
		return reference, nil
	}
	if uint64(len(store.records)) >= uint64(^uint32(0)) {
		return 0, fmt.Errorf(
			"semantic occurrence table overflows uint32",
		)
	}
	store.keys = append(store.keys, key)
	store.records = append(
		store.records,
		semanticExpectedOccurrence{},
	)
	reference := semanticOccurrenceRef(len(store.records))
	store.byIdentity[key] = reference
	return reference, nil
}

func (store *semanticOccurrenceStore) remove(
	id identity.OccurrenceID,
) {
	key, present := store.key(id, false)
	if !present {
		return
	}
	reference := store.byIdentity[key]
	if !reference.valid() ||
		store.records[reference-1].ID().IsZero() {
		return
	}
	store.records[reference-1] = semanticExpectedOccurrence{}
	store.active--
}

func (store *semanticOccurrenceStore) count() int {
	if store == nil {
		return 0
	}
	return store.active
}

func (store *semanticOccurrenceStore) bindNode(
	id identity.OccurrenceID,
	node ast.Node,
) error {
	if store == nil || node == nil {
		return fmt.Errorf(
			"semantic verifier node binding requires store and node",
		)
	}
	key, present := store.key(id, true)
	if !present {
		return fmt.Errorf(
			"semantic verifier node binding rejects zero identity",
		)
	}
	reference, err := store.admitIdentity(key)
	if err != nil {
		return err
	}
	if existing := store.byNode[node]; existing.valid() &&
		existing != reference {
		return fmt.Errorf(
			"checker node has verifier references %d and %d",
			existing, reference,
		)
	}
	store.byNode[node] = reference
	return nil
}

func (store *semanticOccurrenceStore) occurrenceID(
	node ast.Node,
) (identity.OccurrenceID, bool) {
	reference := store.byNode[node]
	if !reference.valid() || int(reference) > len(store.keys) {
		return identity.OccurrenceID{}, false
	}
	key := store.keys[reference-1]
	if key.file == 0 || int(key.file) > len(store.fileIDs) {
		return identity.OccurrenceID{}, false
	}
	span, err := identity.NewSpanID(
		store.fileIDs[key.file-1], key.start, key.end,
	)
	if err != nil {
		return identity.OccurrenceID{}, false
	}
	id, err := identity.NewOccurrenceID(span, key.kind)
	return id, err == nil
}

func (store *semanticOccurrenceStore) buildChildRelations(
	order []semanticOccurrenceRef,
) error {
	if store == nil ||
		store.childRanges != nil ||
		store.children != nil {
		return fmt.Errorf(
			"semantic occurrence child relations require one store build",
		)
	}
	store.childRanges = make(
		[]semanticOccurrenceRange,
		len(store.records)+1,
	)
	total := 0
	for _, childReference := range order {
		child := store.record(childReference)
		if child == nil {
			return fmt.Errorf(
				"semantic occurrence order names absent child %d",
				childReference,
			)
		}
		parentReference := store.reference(child.Parent())
		if !parentReference.valid() {
			continue
		}
		relation := &store.childRanges[parentReference]
		if relation.count == ^uint32(0) ||
			total == int(^uint32(0)) {
			return fmt.Errorf(
				"semantic occurrence child relations overflow uint32",
			)
		}
		relation.count++
		total++
	}
	offset := uint32(0)
	for reference := 1; reference < len(store.childRanges); reference++ {
		relation := &store.childRanges[reference]
		relation.start = offset
		offset += relation.count
	}
	store.children = make([]semanticOccurrenceRef, total)
	cursor := make([]uint32, len(store.childRanges))
	for reference := 1; reference < len(store.childRanges); reference++ {
		cursor[reference] = store.childRanges[reference].start
	}
	for _, childReference := range order {
		child := store.record(childReference)
		parentReference := store.reference(child.Parent())
		if !parentReference.valid() {
			continue
		}
		index := cursor[parentReference]
		store.children[index] = childReference
		cursor[parentReference]++
	}
	for reference := semanticOccurrenceRef(1); int(reference) <= len(store.records); reference++ {
		children := store.childReferences(reference)
		if len(children) < 2 {
			continue
		}
		sort.Slice(children, func(left, right int) bool {
			leftRecord := store.record(children[left])
			rightRecord := store.record(children[right])
			if leftRecord.Edge() != rightRecord.Edge() {
				return leftRecord.Edge() < rightRecord.Edge()
			}
			return leftRecord.Ordinal() < rightRecord.Ordinal()
		})
	}
	return nil
}

func (store *semanticOccurrenceStore) childReferences(
	parent semanticOccurrenceRef,
) []semanticOccurrenceRef {
	if store == nil || !parent.valid() ||
		int(parent) >= len(store.childRanges) {
		return nil
	}
	relation := store.childRanges[parent]
	start := int(relation.start)
	end := start + int(relation.count)
	if start < 0 || end < start || end > len(store.children) {
		return nil
	}
	return store.children[start:end]
}

func (store *semanticOccurrenceStore) referenceCount() int {
	if store == nil {
		return 0
	}
	return len(store.records)
}
