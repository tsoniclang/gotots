package executable

import (
	"cmp"
	"fmt"
	"go/ast"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/structure"
)

const occurrenceIdentityPageSize = 2048

type occurrenceRef uint32

func (reference occurrenceRef) valid() bool {
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
	payload structure.OccurrenceIndex
}

// occurrenceStore is the one package-local executable occurrence dictionary.
// Every region member has one compact identity reference. Only identities
// absent from the structural graph own payloads, normalized by the shared
// Stage-1 occurrence store.
type occurrenceStore struct {
	files           []identity.FileID
	fileByID        map[identity.FileID]uint32
	byIdentity      occurrenceIdentityIndex
	identities      [][]occurrenceIdentityRecord
	payloadBuilders []*structure.OccurrenceStoreBuilder
	payloadStores   []*structure.OccurrenceStore
	identityCount   uint32
	payloadCount    uint32
	sealed          bool
}

func newOccurrenceStore() *occurrenceStore {
	return &occurrenceStore{
		fileByID:   map[identity.FileID]uint32{},
		byIdentity: newOccurrenceIdentityIndex(0),
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
		if store.sealed {
			return occurrenceKey{}, false, fmt.Errorf(
				"executable occurrence store is sealed",
			)
		}
		if uint64(len(store.files)) >= uint64(^uint32(0)) {
			return occurrenceKey{}, false, fmt.Errorf(
				"executable occurrence file table overflows uint32",
			)
		}
		store.files = append(store.files, file)
		store.payloadBuilders = append(store.payloadBuilders, nil)
		store.payloadStores = append(store.payloadStores, nil)
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
	if reference := store.byIdentity.reference(
		store,
		key,
	); reference.valid() {
		return reference, nil
	}
	if store.identityCount == ^uint32(0) {
		return 0, fmt.Errorf(
			"executable occurrence store overflows uint32",
		)
	}
	store.appendIdentityRecord(key)
	reference := occurrenceRef(store.identityCount)
	if err := store.byIdentity.insert(store, reference); err != nil {
		store.removeLastIdentityRecord()
		return 0, err
	}
	return reference, nil
}

func (store *occurrenceStore) appendIdentityRecord(
	key occurrenceKey,
) {
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
			file:  key.file,
			start: key.start,
			end:   key.end,
			kind:  key.kind,
		},
	)
	store.identityCount++
}

func (store *occurrenceStore) removeLastIdentityRecord() {
	if store == nil || store.identityCount == 0 {
		return
	}
	index := int(store.identityCount - 1)
	pageIndex := index / occurrenceIdentityPageSize
	page := store.identities[pageIndex]
	store.identities[pageIndex] = page[:len(page)-1]
	if len(store.identities[pageIndex]) == 0 {
		store.identities = store.identities[:pageIndex]
	}
	store.identityCount--
}

func (store *occurrenceStore) put(
	occurrence structure.Occurrence,
) (occurrenceRef, bool, error) {
	if store == nil || store.sealed {
		return 0, false, fmt.Errorf(
			"executable occurrence store is sealed",
		)
	}
	reference, err := store.admit(occurrence.ID())
	if err != nil {
		return 0, false, err
	}
	identityRecord := store.identityRecord(reference)
	if identityRecord == nil || identityRecord.file == 0 {
		return 0, false, fmt.Errorf(
			"executable occurrence %s has no identity record",
			occurrence.ID(),
		)
	}
	fileIndex := int(identityRecord.file - 1)
	builder := store.payloadBuilders[fileIndex]
	if builder == nil {
		builder, err = structure.NewOccurrenceStoreBuilder(
			store.files[fileIndex],
			64,
		)
		if err != nil {
			return 0, false, err
		}
		store.payloadBuilders[fileIndex] = builder
	}
	if identityRecord.payload != 0 {
		if !builder.Matches(identityRecord.payload, occurrence) {
			return 0, false, fmt.Errorf(
				"occurrence %s has conflicting canonical payloads",
				occurrence.ID(),
			)
		}
		return reference, false, nil
	}
	index, err := builder.Append(occurrence)
	if err != nil {
		return 0, false, err
	}
	identityRecord.payload = index
	if store.payloadCount == ^uint32(0) {
		return 0, false, fmt.Errorf(
			"executable occurrence payload store overflows uint32",
		)
	}
	store.payloadCount++
	return reference, true, nil
}

func (store *occurrenceStore) bindNode(
	index *structure.TransientIndex,
	reference occurrenceRef,
	node ast.Node,
) error {
	if store == nil || store.sealed || index == nil || node == nil {
		return fmt.Errorf(
			"executable occurrence node binding requires mutable storage",
		)
	}
	record := store.identityRecord(reference)
	if record == nil || record.payload == 0 ||
		record.file == 0 ||
		int(record.file) > len(store.payloadBuilders) {
		return fmt.Errorf(
			"executable occurrence node has no additional payload",
		)
	}
	builder := store.payloadBuilders[record.file-1]
	if builder == nil {
		return fmt.Errorf(
			"executable occurrence node has no canonical builder",
		)
	}
	return index.BindPendingExecutableOccurrence(
		builder,
		record.payload,
		node,
	)
}

func (store *occurrenceStore) seal(
	index *structure.TransientIndex,
) error {
	if store == nil || store.sealed {
		return fmt.Errorf(
			"executable occurrence store is absent or already sealed",
		)
	}
	for fileIndex, builder := range store.payloadBuilders {
		if builder == nil {
			continue
		}
		sealed, err := builder.Seal()
		if err != nil {
			return err
		}
		store.payloadStores[fileIndex] = sealed
		store.payloadBuilders[fileIndex] = nil
		if err := index.BindExecutableOccurrenceStore(
			builder,
			sealed,
		); err != nil {
			return err
		}
	}
	store.sealed = true
	return nil
}

func (store *occurrenceStore) reference(
	id identity.OccurrenceID,
) occurrenceRef {
	key, present, err := store.key(id, false)
	if err != nil || !present {
		return 0
	}
	return store.byIdentity.reference(store, key)
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
) (structure.OccurrenceRef, bool) {
	return store.payloadFor(store.reference(id))
}

func (store *occurrenceStore) payloadFor(
	reference occurrenceRef,
) (structure.OccurrenceRef, bool) {
	record := store.identityRecord(reference)
	if record == nil || record.payload == 0 ||
		record.file == 0 ||
		int(record.file) > len(store.payloadStores) {
		return structure.OccurrenceRef{}, false
	}
	payloadStore := store.payloadStores[record.file-1]
	if payloadStore == nil {
		return structure.OccurrenceRef{}, false
	}
	payload, err := payloadStore.Reference(record.payload)
	return payload, err == nil
}

func (store *occurrenceStore) visitPayloads(
	visit func(occurrenceRef, structure.OccurrenceRef) error,
) error {
	if store == nil || !store.sealed || visit == nil {
		return fmt.Errorf(
			"executable occurrence visit requires sealed store and visitor",
		)
	}
	for reference := occurrenceRef(1); uint32(reference) <= store.identityCount; reference++ {
		payload, present := store.payloadFor(reference)
		if !present {
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
