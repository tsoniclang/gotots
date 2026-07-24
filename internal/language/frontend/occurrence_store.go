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

type occurrenceStore struct {
	files      map[identity.FileID]uint32
	nextFile   uint32
	byIdentity map[localOccurrenceKey]*occurrenceInput
}

func newOccurrenceStore() *occurrenceStore {
	return &occurrenceStore{
		files:      map[identity.FileID]uint32{},
		byIdentity: map[localOccurrenceKey]*occurrenceInput{},
	}
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
	key, present := store.key(id, false)
	if !present {
		return nil
	}
	return store.byIdentity[key]
}

func (store *occurrenceStore) put(
	id identity.OccurrenceID,
	record *occurrenceInput,
) error {
	if record == nil || record.occurrence.ID() != id {
		return fmt.Errorf(
			"package occurrence store requires identity-aligned record",
		)
	}
	key, present := store.key(id, true)
	if !present {
		return fmt.Errorf("package occurrence store rejects zero identity")
	}
	if existing := store.byIdentity[key]; existing != nil &&
		existing != record {
		return fmt.Errorf(
			"package occurrence key collides for %s and %s",
			existing.occurrence.ID(),
			id,
		)
	}
	store.byIdentity[key] = record
	return nil
}

func (store *occurrenceStore) remove(
	id identity.OccurrenceID,
) {
	key, present := store.key(id, false)
	if present {
		delete(store.byIdentity, key)
	}
}

func (store *occurrenceStore) count() int {
	if store == nil {
		return 0
	}
	return len(store.byIdentity)
}

func (store *occurrenceStore) visit(
	visit func(*occurrenceInput) error,
) error {
	if store == nil || visit == nil {
		return fmt.Errorf(
			"package occurrence store visit requires store and visitor",
		)
	}
	for _, record := range store.byIdentity {
		if err := visit(record); err != nil {
			return err
		}
	}
	return nil
}
