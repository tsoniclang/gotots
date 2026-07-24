package structure

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/identity"
)

type occurrenceStoreIndex struct {
	slots []OccurrenceIndex
}

func newOccurrenceStoreIndex(
	store *OccurrenceStore,
) (occurrenceStoreIndex, error) {
	if store == nil || len(store.records) == 0 {
		return occurrenceStoreIndex{}, fmt.Errorf(
			"occurrence index requires non-empty canonical storage",
		)
	}
	size := 8
	for size < len(store.records)*2 {
		if size > int(^uint(0)>>1)/2 {
			return occurrenceStoreIndex{}, fmt.Errorf(
				"occurrence index capacity overflows int",
			)
		}
		size *= 2
	}
	index := occurrenceStoreIndex{
		slots: make([]OccurrenceIndex, size),
	}
	for value := 1; value <= len(store.records); value++ {
		reference := OccurrenceIndex(value)
		record := store.records[reference-1]
		if index.referenceRecord(store, record).valid() {
			return occurrenceStoreIndex{}, fmt.Errorf(
				"occurrence store repeats identity %s",
				occurrenceIdentity(
					store.file,
					record.start,
					record.end,
					record.kind,
				),
			)
		}
		index.insertAbsent(store, reference)
	}
	return index, nil
}

func (index occurrenceStoreIndex) reference(
	store *OccurrenceStore,
	id identity.OccurrenceID,
) OccurrenceIndex {
	if store == nil || id.IsZero() ||
		id.Span().File() != store.file {
		return 0
	}
	return index.referenceKey(
		store,
		id.Span().Start(),
		id.Span().End(),
		id.KindID(),
	)
}

func (index occurrenceStoreIndex) referenceRecord(
	store *OccurrenceStore,
	record occurrenceStoreRecord,
) OccurrenceIndex {
	return index.referenceKey(
		store,
		record.start,
		record.end,
		record.kind,
	)
}

func (index occurrenceStoreIndex) referenceKey(
	store *OccurrenceStore,
	start int,
	end int,
	kind uint16,
) OccurrenceIndex {
	if store == nil || len(index.slots) == 0 {
		return 0
	}
	mask := uint64(len(index.slots) - 1)
	slot := hashOccurrenceCoordinate(start, end, kind) & mask
	for probes := 0; probes < len(index.slots); probes++ {
		reference := index.slots[slot]
		if !reference.valid() {
			return 0
		}
		if int(reference) <= len(store.records) {
			record := store.records[reference-1]
			if record.start == start &&
				record.end == end &&
				record.kind == kind {
				return reference
			}
		}
		slot = (slot + 1) & mask
	}
	return 0
}

func (index *occurrenceStoreIndex) insertAbsent(
	store *OccurrenceStore,
	reference OccurrenceIndex,
) {
	record := store.records[reference-1]
	mask := uint64(len(index.slots) - 1)
	slot := hashOccurrenceCoordinate(
		record.start,
		record.end,
		record.kind,
	) & mask
	for index.slots[slot].valid() {
		slot = (slot + 1) & mask
	}
	index.slots[slot] = reference
}

func hashOccurrenceCoordinate(
	start int,
	end int,
	kind uint16,
) uint64 {
	hash := uint64(uint(start)) + 0x9e3779b97f4a7c15
	hash ^= uint64(uint(end)) + 0x9e3779b97f4a7c15 +
		(hash << 6) + (hash >> 2)
	hash ^= uint64(kind) + 0x9e3779b97f4a7c15 +
		(hash << 6) + (hash >> 2)
	hash ^= hash >> 30
	hash *= 0xbf58476d1ce4e5b9
	hash ^= hash >> 27
	hash *= 0x94d049bb133111eb
	return hash ^ (hash >> 31)
}
