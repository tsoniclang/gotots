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

type semanticOccurrenceStore struct {
	files      map[identity.FileID]uint32
	nextFile   uint32
	byIdentity map[semanticOccurrenceKey]*semanticExpectedOccurrence
}

func newSemanticOccurrenceStore() *semanticOccurrenceStore {
	return &semanticOccurrenceStore{
		files:      map[identity.FileID]uint32{},
		byIdentity: map[semanticOccurrenceKey]*semanticExpectedOccurrence{},
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
	key, present := store.key(id, false)
	if !present {
		return nil, false
	}
	record := store.byIdentity[key]
	return record, record != nil
}

func (store *semanticOccurrenceStore) put(
	id identity.OccurrenceID,
	record *semanticExpectedOccurrence,
) error {
	if record == nil || record.ID() != id {
		return fmt.Errorf(
			"semantic occurrence store requires identity-aligned record",
		)
	}
	key, present := store.key(id, true)
	if !present {
		return fmt.Errorf("semantic occurrence store rejects zero identity")
	}
	if existing := store.byIdentity[key]; existing != nil &&
		existing != record {
		return fmt.Errorf(
			"semantic occurrence key collides for %s and %s",
			existing.ID(),
			id,
		)
	}
	store.byIdentity[key] = record
	return nil
}

func (store *semanticOccurrenceStore) count() int {
	if store == nil {
		return 0
	}
	return len(store.byIdentity)
}
