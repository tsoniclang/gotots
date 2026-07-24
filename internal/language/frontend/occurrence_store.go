package frontend

import (
	"fmt"
	"go/ast"
	"sort"

	"github.com/tsoniclang/gotots/internal/identity"
)

type localOccurrenceKey struct {
	file  uint32
	start int
	end   int
	kind  uint16
	order uint32
}

type packageOccurrenceRef uint32

func (reference packageOccurrenceRef) valid() bool {
	return reference != 0
}

type occurrenceRelationRange struct {
	start uint32
	count uint32
}

type occurrenceStore struct {
	files            map[identity.FileID]uint32
	fileIDs          []identity.FileID
	keys             []localOccurrenceKey
	records          []occurrenceInput
	byNode           map[ast.Node]packageOccurrenceRef
	parents          []packageOccurrenceRef
	childRanges      []occurrenceRelationRange
	children         []packageOccurrenceRef
	active           int
	identitiesSealed bool
	sealed           bool
}

func newOccurrenceStore(capacity int) *occurrenceStore {
	if capacity < 0 {
		panic("package occurrence store has negative capacity")
	}
	return &occurrenceStore{
		files:   map[identity.FileID]uint32{},
		keys:    make([]localOccurrenceKey, 0, capacity),
		records: make([]occurrenceInput, 0, capacity),
		byNode:  map[ast.Node]packageOccurrenceRef{},
	}
}

func (store *occurrenceStore) reserve(capacity int) error {
	if store == nil || store.identitiesSealed ||
		store.sealed || capacity < len(store.records) {
		return fmt.Errorf(
			"package occurrence store cannot reserve %d for %d records",
			capacity, len(store.records),
		)
	}
	if cap(store.records) >= capacity {
		return nil
	}
	keys := make(
		[]localOccurrenceKey, len(store.keys), capacity,
	)
	copy(keys, store.keys)
	store.keys = keys
	records := make(
		[]occurrenceInput, len(store.records), capacity,
	)
	copy(records, store.records)
	store.records = records
	return nil
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
		if !admitFile || store.sealed {
			return localOccurrenceKey{}, false
		}
		if uint64(len(store.files)) >= uint64(^uint32(0)) {
			panic("package occurrence file table overflow")
		}
		fileRef = uint32(len(store.files) + 1)
		store.files[file] = fileRef
		store.fileIDs = append(store.fileIDs, file)
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
	record := store.record(store.reference(id))
	if record == nil || record.occurrence.ID().IsZero() {
		return nil
	}
	return record
}

func (store *occurrenceStore) reference(
	id identity.OccurrenceID,
) packageOccurrenceRef {
	reference := store.identityReference(id)
	record := store.record(reference)
	if record == nil || record.occurrence.ID().IsZero() {
		return 0
	}
	return reference
}

func (store *occurrenceStore) identityReference(
	id identity.OccurrenceID,
) packageOccurrenceRef {
	if store == nil || !store.identitiesSealed {
		return 0
	}
	key, present := store.key(id, false)
	if !present {
		return 0
	}
	index := sort.Search(len(store.keys), func(index int) bool {
		return !lessLocalOccurrenceKey(store.keys[index], key)
	})
	if index == len(store.keys) ||
		!sameLocalOccurrenceKey(store.keys[index], key) {
		return 0
	}
	return packageOccurrenceRef(index + 1)
}

func (store *occurrenceStore) identity(
	reference packageOccurrenceRef,
) (identity.OccurrenceID, error) {
	if store == nil ||
		!reference.valid() ||
		int(reference) > len(store.keys) {
		return identity.OccurrenceID{}, fmt.Errorf(
			"package occurrence identity reference is invalid",
		)
	}
	key := store.keys[reference-1]
	if key.file == 0 || int(key.file) > len(store.fileIDs) {
		return identity.OccurrenceID{}, fmt.Errorf(
			"package occurrence file reference is invalid",
		)
	}
	span, err := identity.NewSpanID(
		store.fileIDs[key.file-1], key.start, key.end,
	)
	if err != nil {
		return identity.OccurrenceID{}, err
	}
	return identity.NewOccurrenceID(span, key.kind)
}

func (store *occurrenceStore) record(
	reference packageOccurrenceRef,
) *occurrenceInput {
	if !reference.valid() || int(reference) > len(store.records) {
		return nil
	}
	return &store.records[reference-1]
}

func (store *occurrenceStore) put(
	id identity.OccurrenceID,
	record *occurrenceInput,
) (packageOccurrenceRef, error) {
	if store == nil || !store.identitiesSealed || store.sealed ||
		record == nil || record.occurrence.ID() != id {
		return 0, fmt.Errorf(
			"package occurrence store requires identity-aligned record",
		)
	}
	reference := store.identityReference(id)
	if !reference.valid() {
		return 0, fmt.Errorf(
			"package occurrence store lacks bound identity %s",
			id,
		)
	}
	existing := store.record(reference)
	if !existing.occurrence.ID().IsZero() {
		if existing.occurrence.Equal(record.occurrence) &&
			existing.node == record.node {
			return reference, nil
		}
		return 0, fmt.Errorf(
			"package occurrence %s has conflicting semantic records",
			id,
		)
	}
	if existing.node != nil && existing.node != record.node {
		return 0, fmt.Errorf(
			"package occurrence %s changes its checker node",
			id,
		)
	}
	recordOrder := uint32(store.active)
	store.keys[reference-1].order = recordOrder
	node := existing.node
	*existing = *record
	if node != nil {
		existing.node = node
	}
	store.active++
	return reference, nil
}

func (store *occurrenceStore) bindNode(
	id identity.OccurrenceID,
	node ast.Node,
) error {
	if store == nil || store.identitiesSealed ||
		store.sealed || node == nil {
		return fmt.Errorf(
			"package occurrence identity binding requires mutable storage",
		)
	}
	key, present := store.key(id, true)
	if !present {
		return fmt.Errorf(
			"package occurrence store rejects zero identity",
		)
	}
	if uint64(len(store.records)) >= uint64(^uint32(0)) {
		return fmt.Errorf("package occurrence table overflows uint32")
	}
	store.keys = append(store.keys, key)
	store.records = append(store.records, occurrenceInput{
		node: node,
	})
	return nil
}

func (store *occurrenceStore) sealIdentities() error {
	if store == nil || store.identitiesSealed || store.sealed ||
		len(store.keys) != len(store.records) {
		return fmt.Errorf(
			"package occurrence identities cannot seal invalid storage",
		)
	}
	sort.Sort(occurrenceStoreOrder{store: store})
	for index := 1; index < len(store.keys); index++ {
		if sameLocalOccurrenceKey(
			store.keys[index-1], store.keys[index],
		) {
			return fmt.Errorf(
				"package occurrence key is duplicated",
			)
		}
	}
	for index, record := range store.records {
		reference := packageOccurrenceRef(index + 1)
		if existing := store.byNode[record.node]; existing.valid() &&
			existing != reference {
			return fmt.Errorf(
				"checker node has package occurrence references %d and %d",
				existing, reference,
			)
		}
		store.byNode[record.node] = reference
	}
	store.identitiesSealed = true
	return nil
}

func (store *occurrenceStore) seal() error {
	if store == nil || !store.identitiesSealed || store.sealed {
		return fmt.Errorf(
			"package occurrence store cannot seal invalid storage",
		)
	}
	store.sealed = true
	return nil
}

func (store *occurrenceStore) occurrenceID(
	node ast.Node,
) (identity.OccurrenceID, bool) {
	if store == nil || !store.identitiesSealed {
		return identity.OccurrenceID{}, false
	}
	reference := store.byNode[node]
	if !reference.valid() {
		return identity.OccurrenceID{}, false
	}
	id, err := store.identity(reference)
	return id, err == nil
}

func (store *occurrenceStore) insertionOrder() (
	[]packageOccurrenceRef,
	error,
) {
	if store == nil || !store.sealed {
		return nil, fmt.Errorf(
			"package occurrence order requires sealed storage",
		)
	}
	order := make([]packageOccurrenceRef, store.active)
	for index, key := range store.keys {
		if store.records[index].occurrence.ID().IsZero() {
			continue
		}
		if int(key.order) >= len(order) ||
			order[key.order].valid() {
			return nil, fmt.Errorf(
				"package occurrence insertion order is invalid",
			)
		}
		order[key.order] = packageOccurrenceRef(index + 1)
	}
	return order, nil
}

func (store *occurrenceStore) remove(
	id identity.OccurrenceID,
) {
	if reference := store.reference(id); reference.valid() {
		store.records[reference-1] = occurrenceInput{}
		store.active--
	}
}

func (store *occurrenceStore) count() int {
	if store == nil {
		return 0
	}
	return store.active
}

func (store *occurrenceStore) buildChildRelations(
	order []packageOccurrenceRef,
) (int, error) {
	if store == nil || !store.sealed ||
		store.parents != nil ||
		store.childRanges != nil || store.children != nil {
		return 0, fmt.Errorf(
			"package occurrence child relations require one sealed build",
		)
	}
	store.parents = make(
		[]packageOccurrenceRef,
		len(store.records)+1,
	)
	store.childRanges = make(
		[]occurrenceRelationRange,
		len(store.records)+1,
	)
	total := 0
	for _, childReference := range order {
		child := store.record(childReference)
		if child == nil {
			return 0, fmt.Errorf(
				"package occurrence order names absent child %d",
				childReference,
			)
		}
		parentReference := store.reference(
			child.occurrence.Parent(),
		)
		store.parents[childReference] = parentReference
		if !parentReference.valid() {
			continue
		}
		relation := &store.childRanges[parentReference]
		if relation.count == ^uint32(0) ||
			total == int(^uint32(0)) {
			return 0, fmt.Errorf(
				"package occurrence child relations overflow uint32",
			)
		}
		relation.count++
		total++
	}
	offset := uint32(0)
	for reference := 1; reference < len(store.childRanges); reference++ {
		relation := &store.childRanges[reference]
		relation.start = offset
		offset += relation.count
	}
	store.children = make([]packageOccurrenceRef, total)
	cursor := make([]uint32, len(store.childRanges))
	for reference := 1; reference < len(store.childRanges); reference++ {
		cursor[reference] = store.childRanges[reference].start
	}
	for _, childReference := range order {
		parentReference := store.parents[childReference]
		if !parentReference.valid() {
			continue
		}
		index := cursor[parentReference]
		store.children[index] = childReference
		cursor[parentReference]++
	}
	for reference := packageOccurrenceRef(1); int(reference) <= len(store.records); reference++ {
		parent := store.record(reference)
		children := store.childReferences(reference)
		if parent == nil || len(children) == 0 {
			continue
		}
		for _, childReference := range children {
			child := store.record(childReference)
			if child == nil ||
				child.occurrence.Edge().Parent() !=
					parent.occurrence.Kind() {
				return 0, fmt.Errorf(
					"package occurrence child relation disagrees with parent %s",
					parent.occurrence.ID(),
				)
			}
		}
		sort.Slice(children, func(left, right int) bool {
			leftRecord := store.record(children[left]).occurrence
			rightRecord := store.record(children[right]).occurrence
			if leftRecord.Edge() != rightRecord.Edge() {
				return leftRecord.Edge() < rightRecord.Edge()
			}
			return leftRecord.Ordinal() < rightRecord.Ordinal()
		})
	}
	return total, nil
}

func (store *occurrenceStore) parentReference(
	child packageOccurrenceRef,
) packageOccurrenceRef {
	if store == nil || !child.valid() ||
		int(child) >= len(store.parents) {
		return 0
	}
	return store.parents[child]
}

func (store *occurrenceStore) childReferences(
	parent packageOccurrenceRef,
) []packageOccurrenceRef {
	if store == nil || !parent.valid() ||
		int(parent) >= len(store.childRanges) {
		return nil
	}
	relation := store.childRanges[parent]
	start := int(relation.start)
	end := start + int(relation.count)
	if start < 0 || end < start || end > len(store.children) {
		return nil
	}
	return store.children[start:end]
}

func (store *occurrenceStore) referenceCount() int {
	if store == nil {
		return 0
	}
	return len(store.records)
}

func (store *occurrenceStore) visit(
	visit func(packageOccurrenceRef, *occurrenceInput) error,
) error {
	if store == nil || !store.sealed || visit == nil {
		return fmt.Errorf(
			"package occurrence store visit requires store and visitor",
		)
	}
	for index, record := range store.records {
		if record.occurrence.ID().IsZero() {
			continue
		}
		if err := visit(
			packageOccurrenceRef(index+1), &store.records[index],
		); err != nil {
			return err
		}
	}
	return nil
}

type occurrenceStoreOrder struct {
	store *occurrenceStore
}

func (order occurrenceStoreOrder) Len() int {
	return len(order.store.keys)
}

func (order occurrenceStoreOrder) Less(left, right int) bool {
	return lessLocalOccurrenceKey(
		order.store.keys[left], order.store.keys[right],
	)
}

func (order occurrenceStoreOrder) Swap(left, right int) {
	order.store.keys[left], order.store.keys[right] =
		order.store.keys[right], order.store.keys[left]
	order.store.records[left], order.store.records[right] =
		order.store.records[right], order.store.records[left]
}

func lessLocalOccurrenceKey(
	left localOccurrenceKey,
	right localOccurrenceKey,
) bool {
	switch {
	case left.file != right.file:
		return left.file < right.file
	case left.start != right.start:
		return left.start < right.start
	case left.end != right.end:
		return left.end < right.end
	default:
		return left.kind < right.kind
	}
}

func sameLocalOccurrenceKey(
	left localOccurrenceKey,
	right localOccurrenceKey,
) bool {
	return left.file == right.file &&
		left.start == right.start &&
		left.end == right.end &&
		left.kind == right.kind
}
