package structure

import (
	"fmt"
	"go/ast"
	"sort"

	"github.com/tsoniclang/gotots/internal/identity"
)

type transientOccurrenceDomain uint8

const (
	transientOccurrenceSupplement transientOccurrenceDomain = iota
	transientOccurrenceStructural
	transientOccurrenceExecutable
	transientOccurrenceDomainCount
)

func (domain transientOccurrenceDomain) valid() bool {
	return domain > transientOccurrenceSupplement &&
		domain < transientOccurrenceDomainCount
}

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

type transientStoreAddress struct {
	file   transientFileRef
	domain transientOccurrenceDomain
}

type transientOccurrenceBinding struct {
	key  transientOccurrenceKey
	node ast.Node
}

type transientCanonicalOccurrences struct {
	store *OccurrenceStore
	nodes []ast.Node
	slots []OccurrenceIndex
}

type transientOccurrenceFile struct {
	id identity.FileID

	canonical [transientOccurrenceDomainCount]transientCanonicalOccurrences

	supplementPositions map[transientOccurrenceKey]transientOccurrenceRef
	supplements         []transientOccurrenceBinding
}

type transientOccurrenceStore struct {
	filesByID map[identity.FileID]transientFileRef
	files     []transientOccurrenceFile
	byStore   map[*OccurrenceStore]transientStoreAddress
	pending   map[*OccurrenceStoreBuilder][]ast.Node
	sealed    bool
}

func newTransientOccurrenceStore() *transientOccurrenceStore {
	return &transientOccurrenceStore{
		filesByID: map[identity.FileID]transientFileRef{},
		byStore:   map[*OccurrenceStore]transientStoreAddress{},
		pending:   map[*OccurrenceStoreBuilder][]ast.Node{},
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
			id:                  id,
			supplementPositions: map[transientOccurrenceKey]transientOccurrenceRef{},
		})
		reference = transientFileRef(len(store.files))
		store.filesByID[id] = reference
	}
	return reference, &store.files[reference-1]
}

func (store *transientOccurrenceStore) register(
	domain transientOccurrenceDomain,
	canonical *OccurrenceStore,
	nodes []ast.Node,
	compatible func(ast.Node, ast.Node) bool,
) error {
	if store == nil || store.sealed || !domain.valid() ||
		canonical == nil || !canonical.sealed ||
		canonical.file.IsZero() ||
		len(nodes) != canonical.Count() || len(nodes) == 0 ||
		compatible == nil {
		return fmt.Errorf(
			"transient occurrence registration requires one sealed canonical store",
		)
	}
	if _, duplicate := store.byStore[canonical]; duplicate {
		return fmt.Errorf(
			"canonical occurrence store for %s is already registered",
			canonical.file,
		)
	}
	fileReference, file := store.file(canonical.file, true)
	if file.canonical[domain].store != nil {
		return fmt.Errorf(
			"transient occurrence domain %d repeats for %s",
			domain,
			canonical.file,
		)
	}
	indexed, err := newTransientCanonicalOccurrences(canonical, nodes)
	if err != nil {
		return err
	}
	for index := 1; index <= canonical.Count(); index++ {
		reference := OccurrenceIndex(index)
		record := canonical.records[reference-1]
		key := transientOccurrenceKey{
			start: record.start,
			end:   record.end,
			kind:  record.kind,
		}
		for other := transientOccurrenceStructural; other < transientOccurrenceDomainCount; other++ {
			if other == domain {
				continue
			}
			if file.canonical[other].reference(key).valid() {
				return fmt.Errorf(
					"canonical occurrence %s is stored in domains %d and %d",
					occurrenceIdentity(canonical.file, key.start, key.end, key.kind),
					other,
					domain,
				)
			}
		}
		node := nodes[reference-1]
		if node == nil {
			return fmt.Errorf(
				"canonical occurrence %s has no transient node",
				occurrenceIdentity(canonical.file, key.start, key.end, key.kind),
			)
		}
		kind, classifyErr := Classify(node)
		if classifyErr != nil {
			return classifyErr
		}
		if uint16(kind) != key.kind {
			return fmt.Errorf(
				"canonical occurrence %s has node kind %s",
				occurrenceIdentity(canonical.file, key.start, key.end, key.kind),
				kind,
			)
		}
	}
	file.canonical[domain] = indexed
	store.byStore[canonical] = transientStoreAddress{
		file: fileReference, domain: domain,
	}
	for index := 1; index <= canonical.Count(); index++ {
		reference := OccurrenceIndex(index)
		record := canonical.records[reference-1]
		key := transientOccurrenceKey{
			start: record.start,
			end:   record.end,
			kind:  record.kind,
		}
		if supplement := file.supplementPositions[key]; supplement.valid() {
			binding := &file.supplements[supplement-1]
			if !compatible(binding.node, nodes[reference-1]) {
				return fmt.Errorf(
					"canonical occurrence %s conflicts with supplemental node",
					occurrenceIdentity(canonical.file, key.start, key.end, key.kind),
				)
			}
			delete(file.supplementPositions, key)
			*binding = transientOccurrenceBinding{}
		}
	}
	return nil
}

func newTransientCanonicalOccurrences(
	store *OccurrenceStore,
	nodes []ast.Node,
) (transientCanonicalOccurrences, error) {
	capacity := 8
	for capacity < store.Count()*2 {
		if capacity > int(^uint(0)>>2) {
			return transientCanonicalOccurrences{}, fmt.Errorf(
				"transient canonical occurrence index overflows",
			)
		}
		capacity <<= 1
	}
	out := transientCanonicalOccurrences{
		store: store,
		nodes: nodes,
		slots: make([]OccurrenceIndex, capacity),
	}
	for index := 1; index <= store.Count(); index++ {
		reference := OccurrenceIndex(index)
		record := store.records[reference-1]
		key := transientOccurrenceKey{
			start: record.start,
			end:   record.end,
			kind:  record.kind,
		}
		slot := transientOccurrenceHash(key) & uint64(len(out.slots)-1)
		for probes := 0; probes < len(out.slots); probes++ {
			if !out.slots[slot].valid() {
				out.slots[slot] = reference
				break
			}
			existing := store.records[out.slots[slot]-1]
			if existing.start == key.start &&
				existing.end == key.end &&
				existing.kind == key.kind {
				return transientCanonicalOccurrences{}, fmt.Errorf(
					"canonical occurrence key repeats in %s",
					store.file,
				)
			}
			slot = (slot + 1) & uint64(len(out.slots)-1)
		}
	}
	return out, nil
}

func (canonical transientCanonicalOccurrences) reference(
	key transientOccurrenceKey,
) OccurrenceIndex {
	if canonical.store == nil || len(canonical.slots) == 0 {
		return 0
	}
	slot := transientOccurrenceHash(key) & uint64(len(canonical.slots)-1)
	for probes := 0; probes < len(canonical.slots); probes++ {
		reference := canonical.slots[slot]
		if !reference.valid() {
			return 0
		}
		record := canonical.store.records[reference-1]
		if record.start == key.start &&
			record.end == key.end &&
			record.kind == key.kind {
			return reference
		}
		slot = (slot + 1) & uint64(len(canonical.slots)-1)
	}
	return 0
}

func transientOccurrenceHash(key transientOccurrenceKey) uint64 {
	hash := uint64(uint(key.start)) + 0x9e3779b97f4a7c15
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

func (store *transientOccurrenceStore) bindSupplement(
	id identity.OccurrenceID,
	node ast.Node,
) error {
	if store == nil || store.sealed || node == nil {
		return fmt.Errorf(
			"transient supplemental occurrence rejects sealed or empty binding",
		)
	}
	key, valid := transientKey(id)
	if !valid {
		return fmt.Errorf(
			"transient supplemental occurrence rejects zero identity",
		)
	}
	_, file := store.file(id.Span().File(), true)
	for domain := transientOccurrenceStructural; domain < transientOccurrenceDomainCount; domain++ {
		if reference := file.canonical[domain].reference(key); reference.valid() {
			existing := file.canonical[domain].nodes[reference-1]
			if existing != node {
				return fmt.Errorf(
					"transient occurrence %s has conflicting canonical nodes",
					id,
				)
			}
			return nil
		}
	}
	reference := file.supplementPositions[key]
	if !reference.valid() {
		if uint64(len(file.supplements)) >= uint64(^uint32(0)) {
			return fmt.Errorf(
				"transient supplemental occurrence table overflows uint32",
			)
		}
		file.supplements = append(
			file.supplements,
			transientOccurrenceBinding{key: key, node: node},
		)
		reference = transientOccurrenceRef(len(file.supplements))
		file.supplementPositions[key] = reference
	} else if file.supplements[reference-1].node != node {
		return fmt.Errorf(
			"transient occurrence %s has conflicting supplemental nodes",
			id,
		)
	}
	return nil
}

func (store *transientOccurrenceStore) replace(
	reference OccurrenceRef,
	node ast.Node,
) error {
	if store == nil || store.sealed || !reference.valid() || node == nil {
		return fmt.Errorf(
			"transient executable occurrence replacement is invalid",
		)
	}
	owner, present := store.byStore[reference.store]
	if !present || !owner.file.valid() ||
		int(owner.file) > len(store.files) ||
		!owner.domain.valid() {
		return fmt.Errorf(
			"transient executable occurrence has no canonical store",
		)
	}
	file := &store.files[owner.file-1]
	canonical := &file.canonical[owner.domain]
	if int(reference.index) > len(canonical.nodes) {
		return fmt.Errorf(
			"transient executable occurrence index is outside canonical storage",
		)
	}
	canonical.nodes[reference.index-1] = node
	return nil
}

func (store *transientOccurrenceStore) nodeForReference(
	reference OccurrenceRef,
) (ast.Node, bool) {
	if store == nil || !reference.valid() {
		return nil, false
	}
	owner, present := store.byStore[reference.store]
	if !present || !owner.file.valid() ||
		int(owner.file) > len(store.files) ||
		!owner.domain.valid() {
		return nil, false
	}
	canonical := store.files[owner.file-1].canonical[owner.domain]
	if int(reference.index) > len(canonical.nodes) {
		return nil, false
	}
	node := canonical.nodes[reference.index-1]
	return node, node != nil
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
	for domain := transientOccurrenceStructural; domain < transientOccurrenceDomainCount; domain++ {
		canonical := file.canonical[domain]
		if reference := canonical.reference(key); reference.valid() {
			node := canonical.nodes[reference-1]
			return node, node != nil
		}
	}
	if !store.sealed {
		reference := file.supplementPositions[key]
		if !reference.valid() {
			return nil, false
		}
		node := file.supplements[reference-1].node
		return node, node != nil
	}
	index := sort.Search(len(file.supplements), func(index int) bool {
		return !lessTransientOccurrenceKey(
			file.supplements[index].key,
			key,
		)
	})
	if index == len(file.supplements) ||
		file.supplements[index].key != key {
		return nil, false
	}
	return file.supplements[index].node, true
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
