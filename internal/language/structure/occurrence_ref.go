package structure

import (
	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
)

// OccurrenceRef is an immutable coordinate into one sealed canonical
// occurrence store. It exposes one record at a time without copying or
// retaining the store's normalized payload.
type OccurrenceRef struct {
	store *OccurrenceStore
	index OccurrenceIndex
}

func (reference OccurrenceRef) valid() bool {
	return reference.store != nil &&
		reference.store.sealed &&
		reference.index.valid() &&
		int(reference.index) <= len(reference.store.records)
}

func (reference OccurrenceRef) record() occurrenceStoreRecord {
	if !reference.valid() {
		return occurrenceStoreRecord{}
	}
	return reference.store.records[reference.index-1]
}

func (reference OccurrenceRef) ID() identity.OccurrenceID {
	record := reference.record()
	if record.kind == 0 {
		return identity.OccurrenceID{}
	}
	return occurrenceIdentity(
		reference.store.file,
		record.start,
		record.end,
		record.kind,
	)
}

func (reference OccurrenceRef) Kind() catalog.Kind {
	return catalog.Kind(reference.record().kind)
}

func (reference OccurrenceRef) Parent() identity.OccurrenceID {
	record := reference.record()
	if record.parentKind == 0 {
		return identity.OccurrenceID{}
	}
	return occurrenceIdentity(
		reference.store.file,
		record.parentStart,
		record.parentEnd,
		record.parentKind,
	)
}

func (reference OccurrenceRef) Edge() catalog.Edge {
	return reference.record().edge
}

func (reference OccurrenceRef) Role() catalog.Role {
	return reference.Edge().Role()
}

func (reference OccurrenceRef) Ordinal() int {
	return reference.record().ordinal
}

func (reference OccurrenceRef) Span() Span {
	record := reference.record()
	return Span{
		Start: Position{
			Line: record.startLine, Column: record.startColumn,
			Offset: record.start,
		},
		End: Position{
			Line: record.endLine, Column: record.endColumn,
			Offset: record.end,
		},
	}
}

func (reference OccurrenceRef) Display() DisplaySpan {
	record := reference.record()
	return reference.store.displaySpan(record)
}

func (reference OccurrenceRef) Token() catalog.TokenKind {
	return reference.record().token
}

// Occurrence reconstructs one transient public value from normalized owner
// storage.
func (reference OccurrenceRef) Occurrence() Occurrence {
	if !reference.valid() {
		return Occurrence{}
	}
	return reference.store.occurrence(reference.index)
}

// Equal reports exact semantic payload equality across stores.
func (reference OccurrenceRef) Equal(other OccurrenceRef) bool {
	if !reference.valid() || !other.valid() {
		return reference == other
	}
	if reference.store == other.store && reference.index == other.index {
		return true
	}
	return reference.Occurrence() == other.Occurrence()
}
