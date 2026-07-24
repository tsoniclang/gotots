package structure

import (
	"fmt"
	"go/ast"
	"sort"

	"github.com/tsoniclang/gotots/internal/identity"
)

type transientOccurrenceKey struct {
	start int
	end   int
	kind  uint16
}

type transientOccurrenceRef uint32

func (reference transientOccurrenceRef) valid() bool {
	return reference != 0
}

type transientFileRef uint32

func (reference transientFileRef) valid() bool {
	return reference != 0
}

type transientOccurrenceAddress struct {
	file       transientFileRef
	occurrence transientOccurrenceRef
}

type transientOccurrenceBinding struct {
	key  transientOccurrenceKey
	node ast.Node
}

type transientOccurrenceFile struct {
	id        identity.FileID
	positions map[transientOccurrenceKey]transientOccurrenceRef
	records   []transientOccurrenceBinding
}

type transientOccurrenceStore struct {
	filesByID map[identity.FileID]transientFileRef
	files     []transientOccurrenceFile
	reverse   map[ast.Node]transientOccurrenceAddress
	sealed    bool
}

func newTransientOccurrenceStore() *transientOccurrenceStore {
	return &transientOccurrenceStore{
		filesByID: map[identity.FileID]transientFileRef{},
		reverse:   map[ast.Node]transientOccurrenceAddress{},
	}
}

func transientKey(
	id identity.OccurrenceID,
) (transientOccurrenceKey, bool) {
	if id.IsZero() {
		return transientOccurrenceKey{}, false
	}
	return transientOccurrenceKey{
		start: id.Span().Start(),
		end:   id.Span().End(),
		kind:  id.KindID(),
	}, true
}

func (store *transientOccurrenceStore) file(
	id identity.FileID,
	admit bool,
) (transientFileRef, *transientOccurrenceFile) {
	if store == nil || id.IsZero() {
		return 0, nil
	}
	reference := store.filesByID[id]
	if !reference.valid() {
		if !admit || store.sealed {
			return 0, nil
		}
		if uint64(len(store.files)) >= uint64(^uint32(0)) {
			panic("transient occurrence file table overflows uint32")
		}
		store.files = append(store.files, transientOccurrenceFile{
			id:        id,
			positions: map[transientOccurrenceKey]transientOccurrenceRef{},
		})
		reference = transientFileRef(len(store.files))
		store.filesByID[id] = reference
	}
	return reference, &store.files[reference-1]
}

func (store *transientOccurrenceStore) node(
	id identity.OccurrenceID,
) (ast.Node, bool) {
	key, valid := transientKey(id)
	if !valid {
		return nil, false
	}
	_, file := store.file(id.Span().File(), false)
	if file == nil {
		return nil, false
	}
	if !store.sealed {
		reference := file.positions[key]
		if !reference.valid() {
			return nil, false
		}
		return file.records[reference-1].node, true
	}
	index := sort.Search(len(file.records), func(index int) bool {
		return !lessTransientOccurrenceKey(
			file.records[index].key, key,
		)
	})
	if index == len(file.records) ||
		file.records[index].key != key {
		return nil, false
	}
	return file.records[index].node, true
}

func (store *transientOccurrenceStore) bind(
	id identity.OccurrenceID,
	node ast.Node,
) error {
	if store == nil || store.sealed || node == nil {
		return fmt.Errorf(
			"transient occurrence store rejects sealed or empty binding",
		)
	}
	key, valid := transientKey(id)
	if !valid {
		return fmt.Errorf(
			"transient occurrence store rejects zero identity",
		)
	}
	fileReference, file := store.file(id.Span().File(), true)
	reference := file.positions[key]
	if !reference.valid() {
		if uint64(len(file.records)) >= uint64(^uint32(0)) {
			return fmt.Errorf(
				"transient occurrence table overflows uint32",
			)
		}
		file.records = append(file.records, transientOccurrenceBinding{
			key: key, node: node,
		})
		reference = transientOccurrenceRef(len(file.records))
		file.positions[key] = reference
	}
	address := transientOccurrenceAddress{
		file: fileReference, occurrence: reference,
	}
	if existing, present := store.reverse[node]; present &&
		existing != address {
		existingID, err := store.identity(existing)
		if err != nil {
			return err
		}
		return fmt.Errorf(
			"transient node has conflicting occurrences %s and %s",
			existingID, id,
		)
	}
	store.reverse[node] = address
	file.records[reference-1].node = node
	return nil
}

func (store *transientOccurrenceStore) identity(
	address transientOccurrenceAddress,
) (identity.OccurrenceID, error) {
	if store == nil ||
		!address.file.valid() ||
		int(address.file) > len(store.files) {
		return identity.OccurrenceID{}, fmt.Errorf(
			"transient occurrence address has absent file",
		)
	}
	file := &store.files[address.file-1]
	if !address.occurrence.valid() ||
		int(address.occurrence) > len(file.records) {
		return identity.OccurrenceID{}, fmt.Errorf(
			"transient occurrence address has absent record",
		)
	}
	return transientOccurrenceIdentity(
		file.id,
		file.records[address.occurrence-1].key,
	)
}

func transientOccurrenceIdentity(
	file identity.FileID,
	key transientOccurrenceKey,
) (identity.OccurrenceID, error) {
	span, err := identity.NewSpanID(file, key.start, key.end)
	if err != nil {
		return identity.OccurrenceID{}, err
	}
	return identity.NewOccurrenceID(span, key.kind)
}

func (store *transientOccurrenceStore) seal(
	counterparts map[ast.Node]ast.Node,
	originals map[ast.Node]ast.Node,
) error {
	if store == nil || store.sealed || store.reverse == nil {
		return fmt.Errorf(
			"transient occurrence store cannot seal",
		)
	}
	for node, address := range store.reverse {
		file := &store.files[address.file-1]
		bound := file.records[address.occurrence-1].node
		if bound != node &&
			counterparts[bound] != node &&
			originals[bound] != node &&
			counterparts[node] != bound &&
			originals[node] != bound {
			id, err := store.identity(address)
			if err != nil {
				return err
			}
			return fmt.Errorf(
				"transient reverse occurrence %s disagrees with its forward binding",
				id,
			)
		}
	}
	for index := range store.files {
		file := &store.files[index]
		sort.Slice(file.records, func(left, right int) bool {
			return lessTransientOccurrenceKey(
				file.records[left].key,
				file.records[right].key,
			)
		})
		for record := 1; record < len(file.records); record++ {
			if file.records[record-1].key ==
				file.records[record].key {
				return fmt.Errorf(
					"transient occurrence key repeats in %s",
					file.id,
				)
			}
		}
		file.positions = nil
	}
	store.reverse = nil
	store.sealed = true
	return nil
}

func (store *transientOccurrenceStore) visitFiles(
	files []identity.FileID,
	visit func(identity.OccurrenceID, ast.Node) error,
) error {
	if store == nil || !store.sealed || visit == nil {
		return fmt.Errorf(
			"transient occurrence visit requires sealed storage and visitor",
		)
	}
	seen := map[identity.FileID]bool{}
	for _, fileID := range files {
		if fileID.IsZero() || seen[fileID] {
			return fmt.Errorf(
				"transient occurrence visit has zero or duplicate file %s",
				fileID,
			)
		}
		seen[fileID] = true
		_, file := store.file(fileID, false)
		if file == nil {
			continue
		}
		for _, record := range file.records {
			id, err := transientOccurrenceIdentity(
				file.id, record.key,
			)
			if err != nil {
				return err
			}
			if err := visit(id, record.node); err != nil {
				return err
			}
		}
	}
	return nil
}

func (store *transientOccurrenceStore) countFiles(
	files []identity.FileID,
) (int, error) {
	if store == nil || !store.sealed {
		return 0, fmt.Errorf(
			"transient occurrence count requires sealed storage",
		)
	}
	seen := map[identity.FileID]bool{}
	total := 0
	for _, fileID := range files {
		if fileID.IsZero() || seen[fileID] {
			return 0, fmt.Errorf(
				"transient occurrence count has zero or duplicate file %s",
				fileID,
			)
		}
		seen[fileID] = true
		_, file := store.file(fileID, false)
		if file != nil {
			total += len(file.records)
		}
	}
	return total, nil
}

func lessTransientOccurrenceKey(
	left transientOccurrenceKey,
	right transientOccurrenceKey,
) bool {
	switch {
	case left.start != right.start:
		return left.start < right.start
	case left.end != right.end:
		return left.end < right.end
	default:
		return left.kind < right.kind
	}
}
