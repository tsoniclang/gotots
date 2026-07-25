package executable

import "fmt"

type occurrenceIdentityIndex struct {
	slots []occurrenceRef
	count uint32
}

func newOccurrenceIdentityIndex(
	capacity int,
) occurrenceIdentityIndex {
	size := 8
	for size < capacity*2 {
		size *= 2
	}
	return occurrenceIdentityIndex{
		slots: make([]occurrenceRef, size),
	}
}

func (index *occurrenceIdentityIndex) reference(
	store *occurrenceStore,
	key occurrenceKey,
) occurrenceRef {
	if index == nil || store == nil || len(index.slots) == 0 {
		return 0
	}
	mask := uint64(len(index.slots) - 1)
	slot := hashOccurrenceKey(key) & mask
	for probes := 0; probes < len(index.slots); probes++ {
		reference := index.slots[slot]
		if !reference.valid() {
			return 0
		}
		if record := store.identityRecord(reference); record != nil &&
			record.file == key.file &&
			record.start == key.start &&
			record.end == key.end &&
			record.kind == key.kind {
			return reference
		}
		slot = (slot + 1) & mask
	}
	return 0
}

func (index *occurrenceIdentityIndex) insert(
	store *occurrenceStore,
	reference occurrenceRef,
) error {
	record := store.identityRecord(reference)
	if index == nil || record == nil {
		return fmt.Errorf(
			"executable occurrence index rejects invalid reference %d",
			reference,
		)
	}
	key := occurrenceKey{
		file:  record.file,
		start: record.start,
		end:   record.end,
		kind:  record.kind,
	}
	if index.reference(store, key).valid() {
		return fmt.Errorf(
			"executable occurrence index rejects duplicate key",
		)
	}
	if (uint64(index.count)+1)*4 >
		uint64(len(index.slots))*3 {
		if err := index.grow(store); err != nil {
			return err
		}
	}
	index.insertAbsent(store, reference)
	index.count++
	return nil
}

func (index *occurrenceIdentityIndex) grow(
	store *occurrenceStore,
) error {
	if len(index.slots) > int(^uint(0)>>1)/2 {
		return fmt.Errorf(
			"executable occurrence index capacity overflows int",
		)
	}
	previous := index.slots
	index.slots = make([]occurrenceRef, len(previous)*2)
	for _, reference := range previous {
		if reference.valid() {
			index.insertAbsent(store, reference)
		}
	}
	return nil
}

func (index *occurrenceIdentityIndex) insertAbsent(
	store *occurrenceStore,
	reference occurrenceRef,
) {
	record := store.identityRecord(reference)
	key := occurrenceKey{
		file:  record.file,
		start: record.start,
		end:   record.end,
		kind:  record.kind,
	}
	mask := uint64(len(index.slots) - 1)
	slot := hashOccurrenceKey(key) & mask
	for index.slots[slot].valid() {
		slot = (slot + 1) & mask
	}
	index.slots[slot] = reference
}

func hashOccurrenceKey(key occurrenceKey) uint64 {
	hash := uint64(key.file) + 0x9e3779b97f4a7c15
	hash ^= uint64(uint(key.start)) + 0x9e3779b97f4a7c15 +
		(hash << 6) + (hash >> 2)
	hash ^= uint64(uint(key.end)) + 0x9e3779b97f4a7c15 +
		(hash << 6) + (hash >> 2)
	hash ^= uint64(key.kind) + 0x9e3779b97f4a7c15 +
		(hash << 6) + (hash >> 2)
	hash ^= hash >> 30
	hash *= 0xbf58476d1ce4e5b9
	hash ^= hash >> 27
	hash *= 0x94d049bb133111eb
	return hash ^ (hash >> 31)
}
