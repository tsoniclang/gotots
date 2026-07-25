package stagecheck

import "fmt"

type semanticOccurrenceIndex struct {
	slots []semanticOccurrenceRef
	count uint32
}

func newSemanticOccurrenceIndex(
	capacity int,
) semanticOccurrenceIndex {
	size := 8
	for size < capacity*2 {
		size *= 2
	}
	return semanticOccurrenceIndex{
		slots: make([]semanticOccurrenceRef, size),
	}
}

func (index *semanticOccurrenceIndex) reference(
	keys []semanticOccurrenceKey,
	key semanticOccurrenceKey,
) semanticOccurrenceRef {
	if index == nil || len(index.slots) == 0 {
		return 0
	}
	mask := uint64(len(index.slots) - 1)
	slot := hashSemanticOccurrenceKey(key) & mask
	for probes := 0; probes < len(index.slots); probes++ {
		reference := index.slots[slot]
		if !reference.valid() {
			return 0
		}
		if int(reference) <= len(keys) &&
			keys[reference-1] == key {
			return reference
		}
		slot = (slot + 1) & mask
	}
	return 0
}

func (index *semanticOccurrenceIndex) insert(
	keys []semanticOccurrenceKey,
	reference semanticOccurrenceRef,
) error {
	if index == nil || !reference.valid() ||
		int(reference) > len(keys) {
		return fmt.Errorf(
			"semantic occurrence index rejects invalid reference %d",
			reference,
		)
	}
	if index.reference(keys, keys[reference-1]).valid() {
		return fmt.Errorf(
			"semantic occurrence index rejects duplicate key",
		)
	}
	if (uint64(index.count)+1)*4 >
		uint64(len(index.slots))*3 {
		if err := index.grow(keys); err != nil {
			return err
		}
	}
	index.insertAbsent(keys, reference)
	index.count++
	return nil
}

func (index *semanticOccurrenceIndex) grow(
	keys []semanticOccurrenceKey,
) error {
	if len(index.slots) > int(^uint(0)>>1)/2 {
		return fmt.Errorf(
			"semantic occurrence index capacity overflows int",
		)
	}
	previous := index.slots
	index.slots = make(
		[]semanticOccurrenceRef,
		len(previous)*2,
	)
	for _, reference := range previous {
		if reference.valid() {
			index.insertAbsent(keys, reference)
		}
	}
	return nil
}

func (index *semanticOccurrenceIndex) insertAbsent(
	keys []semanticOccurrenceKey,
	reference semanticOccurrenceRef,
) {
	key := keys[reference-1]
	mask := uint64(len(index.slots) - 1)
	slot := hashSemanticOccurrenceKey(key) & mask
	for index.slots[slot].valid() {
		slot = (slot + 1) & mask
	}
	index.slots[slot] = reference
}

func hashSemanticOccurrenceKey(
	key semanticOccurrenceKey,
) uint64 {
	hash := uint64(key.file) + 0x9e3779b97f4a7c15
	hash ^= uint64(key.start) + 0x9e3779b97f4a7c15 +
		(hash << 6) + (hash >> 2)
	hash ^= uint64(key.end) + 0x9e3779b97f4a7c15 +
		(hash << 6) + (hash >> 2)
	hash ^= uint64(key.kind) + 0x9e3779b97f4a7c15 +
		(hash << 6) + (hash >> 2)
	hash ^= hash >> 30
	hash *= 0xbf58476d1ce4e5b9
	hash ^= hash >> 27
	hash *= 0x94d049bb133111eb
	return hash ^ (hash >> 31)
}
