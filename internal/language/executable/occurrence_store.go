package executable

import (
	"cmp"
	"fmt"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/structure"
)

const (
	occurrenceIdentityPageSize = 2048
	occurrencePayloadPageSize  = 1024
)

type occurrenceRef uint32
type occurrencePayloadRef uint32

func (reference occurrenceRef) valid() bool {
	return reference != 0
}

func (reference occurrencePayloadRef) valid() bool {
	return reference != 0
}

type occurrenceKey struct {
	file  uint32
	start int
	end   int
	kind  uint16
}

type occurrenceIdentityRecord struct {
	file    uint32
	start   int
	end     int
	kind    uint16
	payload occurrencePayloadRef
}

// occurrenceStore is the one package-local executable occurrence dictionary.
// Every region member has one occurrenceRef. Only identities absent from the
// structural graph attach a canonical payload to that same reference.
type occurrenceStore struct {
	files         []identity.FileID
	fileByID      map[identity.FileID]uint32
	byIdentity    map[occurrenceKey]occurrenceRef
	identities    [][]occurrenceIdentityRecord
	payloads      [][]structure.Occurrence
	identityCount uint32
	payloadCount  uint32
}

func newOccurrenceStore() *occurrenceStore {
	return &occurrenceStore{
		fileByID:   map[identity.FileID]uint32{},
		byIdentity: map[occurrenceKey]occurrenceRef{},
	}
}

func (store *occurrenceStore) key(
	id identity.OccurrenceID,
	admitFile bool,
) (occurrenceKey, bool, error) {
	if store == nil || id.IsZero() {
		return occurrenceKey{}, false, nil
	}
	file := id.Span().File()
	fileReference, present := store.fileByID[file]
	if !present {
		if !admitFile {
			return occurrenceKey{}, false, nil
		}
		if uint64(len(store.files)) >= uint64(^uint32(0)) {
			return occurrenceKey{}, false, fmt.Errorf(
				"executable occurrence file table overflows uint32",
			)
		}
		store.files = append(store.files, file)
		fileReference = uint32(len(store.files))
		store.fileByID[file] = fileReference
	}
	return occurrenceKey{
		file:  fileReference,
		start: id.Span().Start(),
		end:   id.Span().End(),
		kind:  id.KindID(),
	}, true, nil
}

func (store *occurrenceStore) admit(
	id identity.OccurrenceID,
) (occurrenceRef, error) {
	key, present, err := store.key(id, true)
	if err != nil {
		return 0, err
	}
	if !present {
		return 0, fmt.Errorf(
			"executable occurrence store rejects zero identity",
		)
	}
	if reference := store.byIdentity[key]; reference.valid() {
		return reference, nil
	}
	if store.identityCount == ^uint32(0) {
		return 0, fmt.Errorf(
			"executable occurrence store overflows uint32",
		)
	}
	index := int(store.identityCount)
	pageIndex := index / occurrenceIdentityPageSize
	if pageIndex == len(store.identities) {
		store.identities = append(
			store.identities,
			make(
				[]occurrenceIdentityRecord,
				0,
				occurrenceIdentityPageSize,
			),
		)
	}
	store.identities[pageIndex] = append(
		store.identities[pageIndex],
		occurrenceIdentityRecord{
			file: key.file, start: key.start, end: key.end, kind: key.kind,
		},
	)
	store.identityCount++
	reference := occurrenceRef(store.identityCount)
	store.byIdentity[key] = reference
	return reference, nil
}

func (store *occurrenceStore) put(
	occurrence structure.Occurrence,
) (occurrenceRef, bool, error) {
	reference, err := store.admit(occurrence.ID())
	if err != nil {
		return 0, false, err
	}
	identityRecord := store.identityRecord(reference)
	if identityRecord == nil {
		return 0, false, fmt.Errorf(
			"executable occurrence %s has no identity record",
			occurrence.ID(),
		)
	}
	if identityRecord.payload.valid() {
		existing := store.payload(identityRecord.payload)
		if existing != nil && *existing == occurrence {
			return reference, false, nil
		}
		return 0, false, fmt.Errorf(
			"executable occurrence %s has conflicting payloads",
			occurrence.ID(),
		)
	}
	if store.payloadCount == ^uint32(0) {
		return 0, false, fmt.Errorf(
			"executable occurrence payload store overflows uint32",
		)
	}
	index := int(store.payloadCount)
	pageIndex := index / occurrencePayloadPageSize
	if pageIndex == len(store.payloads) {
		store.payloads = append(
			store.payloads,
			make(
				[]structure.Occurrence,
				0,
				occurrencePayloadPageSize,
			),
		)
	}
	store.payloads[pageIndex] = append(
		store.payloads[pageIndex], occurrence,
	)
	store.payloadCount++
	identityRecord.payload = occurrencePayloadRef(store.payloadCount)
	return reference, true, nil
}

func (store *occurrenceStore) reference(
	id identity.OccurrenceID,
) occurrenceRef {
	key, present, err := store.key(id, false)
	if err != nil || !present {
		return 0
	}
	return store.byIdentity[key]
}

func (store *occurrenceStore) identityRecord(
	reference occurrenceRef,
) *occurrenceIdentityRecord {
	if store == nil ||
		!reference.valid() ||
		uint32(reference) > store.identityCount {
		return nil
	}
	index := int(reference - 1)
	return &store.identities[index/occurrenceIdentityPageSize][index%occurrenceIdentityPageSize]
}

func (store *occurrenceStore) payload(
	reference occurrencePayloadRef,
) *structure.Occurrence {
	if store == nil ||
		!reference.valid() ||
		uint32(reference) > store.payloadCount {
		return nil
	}
	index := int(reference - 1)
	return &store.payloads[index/occurrencePayloadPageSize][index%occurrencePayloadPageSize]
}

func (store *occurrenceStore) id(
	reference occurrenceRef,
) (identity.OccurrenceID, error) {
	record := store.identityRecord(reference)
	if record == nil ||
		record.file == 0 ||
		int(record.file) > len(store.files) {
		return identity.OccurrenceID{}, fmt.Errorf(
			"executable occurrence reference %d is out of range",
			reference,
		)
	}
	span, err := identity.NewSpanID(
		store.files[record.file-1],
		record.start,
		record.end,
	)
	if err != nil {
		return identity.OccurrenceID{}, err
	}
	return identity.NewOccurrenceID(span, record.kind)
}

func (store *occurrenceStore) mustID(
	reference occurrenceRef,
) identity.OccurrenceID {
	id, err := store.id(reference)
	if err != nil {
		panic(err)
	}
	return id
}

func (store *occurrenceStore) get(
	id identity.OccurrenceID,
) (*structure.Occurrence, bool) {
	record := store.identityRecord(store.reference(id))
	if record == nil || !record.payload.valid() {
		return nil, false
	}
	payload := store.payload(record.payload)
	return payload, payload != nil
}

func (store *occurrenceStore) payloadFor(
	reference occurrenceRef,
) *structure.Occurrence {
	record := store.identityRecord(reference)
	if record == nil || !record.payload.valid() {
		return nil
	}
	return store.payload(record.payload)
}

func (store *occurrenceStore) visitPayloads(
	visit func(occurrenceRef, *structure.Occurrence) error,
) error {
	if store == nil || visit == nil {
		return fmt.Errorf(
			"executable occurrence visit requires store and visitor",
		)
	}
	for reference := occurrenceRef(1); uint32(reference) <= store.identityCount; reference++ {
		payload := store.payloadFor(reference)
		if payload == nil {
			continue
		}
		if err := visit(reference, payload); err != nil {
			return err
		}
	}
	return nil
}

func (store *occurrenceStore) compare(
	left occurrenceRef,
	right occurrenceRef,
) int {
	leftRecord := store.identityRecord(left)
	rightRecord := store.identityRecord(right)
	if leftRecord == nil || rightRecord == nil {
		return cmp.Compare(left, right)
	}
	leftFile := store.files[leftRecord.file-1]
	rightFile := store.files[rightRecord.file-1]
	if order := leftFile.Compare(rightFile); order != 0 {
		return order
	}
	if order := cmp.Compare(leftRecord.start, rightRecord.start); order != 0 {
		return order
	}
	if order := cmp.Compare(leftRecord.end, rightRecord.end); order != 0 {
		return order
	}
	return cmp.Compare(leftRecord.kind, rightRecord.kind)
}

func (store *occurrenceStore) length() int {
	if store == nil {
		return 0
	}
	return int(store.identityCount)
}

func (store *occurrenceStore) payloadLength() int {
	if store == nil {
		return 0
	}
	return int(store.payloadCount)
}
