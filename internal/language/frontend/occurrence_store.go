package frontend

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/identity"
)

type localOccurrenceKey struct {
	file  uint32
	start int
	end   int
	kind  uint16
}

type packageOccurrenceRef uint32

func (reference packageOccurrenceRef) valid() bool {
	return reference != 0
}

type occurrenceStore struct {
	files      map[identity.FileID]uint32
	nextFile   uint32
	byIdentity map[localOccurrenceKey]packageOccurrenceRef
	records    []occurrenceInput
}

func newOccurrenceStore(capacity int) *occurrenceStore {
	if capacity < 0 {
		panic("package occurrence store has negative capacity")
	}
	return &occurrenceStore{
		files: map[identity.FileID]uint32{},
		byIdentity: make(
			map[localOccurrenceKey]packageOccurrenceRef,
			capacity,
		),
		records: make([]occurrenceInput, 0, capacity),
	}
}

func (store *occurrenceStore) reserve(capacity int) error {
	if store == nil || capacity < len(store.records) {
		return fmt.Errorf(
			"package occurrence store cannot reserve %d for %d records",
			capacity, len(store.records),
		)
	}
	if cap(store.records) >= capacity {
		return nil
	}
	records := make(
		[]occurrenceInput, len(store.records), capacity,
	)
	copy(records, store.records)
	store.records = records
	identities := make(
		map[localOccurrenceKey]packageOccurrenceRef,
		capacity,
	)
	for key, reference := range store.byIdentity {
		identities[key] = reference
	}
	store.byIdentity = identities
	return nil
}

func (store *occurrenceStore) key(
	id identity.OccurrenceID,
	admitFile bool,
) (localOccurrenceKey, bool) {
	if store == nil || id.IsZero() {
		return localOccurrenceKey{}, false
	}
	file := id.Span().File()
	fileRef, present := store.files[file]
	if !present {
		if !admitFile {
			return localOccurrenceKey{}, false
		}
		store.nextFile++
		if store.nextFile == 0 {
			panic("package occurrence file table overflow")
		}
		fileRef = store.nextFile
		store.files[file] = fileRef
	}
	return localOccurrenceKey{
		file:  fileRef,
		start: id.Span().Start(),
		end:   id.Span().End(),
		kind:  id.KindID(),
	}, true
}

func (store *occurrenceStore) get(
	id identity.OccurrenceID,
) *occurrenceInput {
	return store.record(store.reference(id))
}

func (store *occurrenceStore) reference(
	id identity.OccurrenceID,
) packageOccurrenceRef {
	key, present := store.key(id, false)
	if !present {
		return 0
	}
	return store.byIdentity[key]
}

func (store *occurrenceStore) record(
	reference packageOccurrenceRef,
) *occurrenceInput {
	if !reference.valid() || int(reference) > len(store.records) {
		return nil
	}
	return &store.records[reference-1]
}

func (store *occurrenceStore) put(
	id identity.OccurrenceID,
	record *occurrenceInput,
) (packageOccurrenceRef, error) {
	if record == nil || record.occurrence.ID() != id {
		return 0, fmt.Errorf(
			"package occurrence store requires identity-aligned record",
		)
	}
	key, present := store.key(id, true)
	if !present {
		return 0, fmt.Errorf(
			"package occurrence store rejects zero identity",
		)
	}
	if reference := store.byIdentity[key]; reference.valid() {
		existing := store.record(reference)
		if existing.occurrence == record.occurrence &&
			existing.node == record.node {
			return reference, nil
		}
		return 0, fmt.Errorf(
			"package occurrence key collides for %s and %s",
			existing.occurrence.ID(),
			id,
		)
	}
	if uint64(len(store.records)) >= uint64(^uint32(0)) {
		return 0, fmt.Errorf("package occurrence table overflows uint32")
	}
	store.records = append(store.records, *record)
	reference := packageOccurrenceRef(len(store.records))
	store.byIdentity[key] = reference
	return reference, nil
}

func (store *occurrenceStore) remove(
	id identity.OccurrenceID,
) {
	key, present := store.key(id, false)
	if present {
		reference := store.byIdentity[key]
		store.records[reference-1] = occurrenceInput{}
		delete(store.byIdentity, key)
	}
}

func (store *occurrenceStore) count() int {
	if store == nil {
		return 0
	}
	return len(store.byIdentity)
}

func (store *occurrenceStore) referenceCount() int {
	if store == nil {
		return 0
	}
	return len(store.records)
}

func (store *occurrenceStore) visit(
	visit func(packageOccurrenceRef, *occurrenceInput) error,
) error {
	if store == nil || visit == nil {
		return fmt.Errorf(
			"package occurrence store visit requires store and visitor",
		)
	}
	for index, record := range store.records {
		if record.occurrence.ID().IsZero() {
			continue
		}
		if err := visit(
			packageOccurrenceRef(index+1), &store.records[index],
		); err != nil {
			return err
		}
	}
	return nil
}
