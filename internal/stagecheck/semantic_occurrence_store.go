package stagecheck

import (
	"fmt"

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

type semanticOccurrenceStore struct {
	files      map[identity.FileID]uint32
	nextFile   uint32
	byIdentity map[semanticOccurrenceKey]semanticOccurrenceRef
	records    []semanticExpectedOccurrence
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
	return record, record != nil
}

func (store *semanticOccurrenceStore) reference(
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
		if existing.OccurrenceRef == record.OccurrenceRef {
			return reference, nil
		}
		return 0, fmt.Errorf(
			"semantic occurrence key collides for %s and %s",
			existing.ID(),
			id,
		)
	}
	if uint64(len(store.records)) >= uint64(^uint32(0)) {
		return 0, fmt.Errorf(
			"semantic occurrence table overflows uint32",
		)
	}
	store.records = append(store.records, *record)
	reference := semanticOccurrenceRef(len(store.records))
	store.byIdentity[key] = reference
	return reference, nil
}

func (store *semanticOccurrenceStore) count() int {
	if store == nil {
		return 0
	}
	return len(store.byIdentity)
}

func (store *semanticOccurrenceStore) referenceCount() int {
	if store == nil {
		return 0
	}
	return len(store.records)
}
