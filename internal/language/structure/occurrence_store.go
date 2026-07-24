package structure

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
)

// OccurrenceIndex is a validated store-local coordinate. It has no semantic
// meaning outside its owning sealed store.
type OccurrenceIndex uint32

func (index OccurrenceIndex) valid() bool {
	return index != 0
}

type displayFileIndex uint32

type occurrenceStoreRecord struct {
	start int
	end   int
	kind  uint16

	parentStart int
	parentEnd   int
	parentKind  uint16
	edge        catalog.Edge
	ordinal     int

	startLine   int
	startColumn int
	endLine     int
	endColumn   int

	displayStartFile   displayFileIndex
	displayStartLine   int
	displayStartColumn int
	displayEndFile     displayFileIndex
	displayEndLine     int
	displayEndColumn   int

	token catalog.TokenKind
}

// OccurrenceStore owns one file's normalized immutable occurrence payloads.
// Identity file components and display filenames are stored once.
type OccurrenceStore struct {
	file         identity.FileID
	displayFiles []string
	records      []occurrenceStoreRecord
	sealed       bool
}

// OccurrenceStoreBuilder is the only mutable construction route.
type OccurrenceStoreBuilder struct {
	store         *OccurrenceStore
	displayByName map[string]displayFileIndex
	sealed        bool
}

func NewOccurrenceStoreBuilder(
	file identity.FileID,
	capacity int,
) (*OccurrenceStoreBuilder, error) {
	if file.IsZero() || capacity < 0 {
		return nil, fmt.Errorf(
			"occurrence store requires file identity and non-negative capacity",
		)
	}
	return &OccurrenceStoreBuilder{
		store: &OccurrenceStore{
			file:    file,
			records: make([]occurrenceStoreRecord, 0, capacity),
		},
		displayByName: map[string]displayFileIndex{},
	}, nil
}

// Append validates and normalizes one occurrence. The semantic owner that
// supplies the record owns identity uniqueness; sealed graph admission
// independently checks that invariant.
func (builder *OccurrenceStoreBuilder) Append(
	occurrence Occurrence,
) (OccurrenceIndex, error) {
	if builder == nil || builder.sealed || builder.store == nil {
		return 0, fmt.Errorf("occurrence store builder is sealed")
	}
	id := occurrence.ID()
	if id.IsZero() || id.Span().File() != builder.store.file ||
		(!occurrence.Parent().IsZero() &&
			occurrence.Parent().Span().File() != builder.store.file) {
		return 0, fmt.Errorf(
			"occurrence store rejects cross-file or zero occurrence %s",
			id,
		)
	}
	if uint64(len(builder.store.records)) >= uint64(^uint32(0)) {
		return 0, fmt.Errorf(
			"occurrence store overflows uint32",
		)
	}
	record, err := builder.record(occurrence)
	if err != nil {
		return 0, err
	}
	builder.store.records = append(builder.store.records, record)
	index := OccurrenceIndex(len(builder.store.records))
	return index, nil
}

// Matches compares a previously appended coordinate to one candidate without
// constructing a second retained payload.
func (builder *OccurrenceStoreBuilder) Matches(
	index OccurrenceIndex,
	occurrence Occurrence,
) bool {
	return builder != nil &&
		!builder.sealed &&
		builder.store != nil &&
		builder.store.occurrence(index) == occurrence
}

func (builder *OccurrenceStoreBuilder) occurrence(
	index OccurrenceIndex,
) Occurrence {
	if builder == nil ||
		builder.sealed ||
		builder.store == nil {
		return Occurrence{}
	}
	return builder.store.occurrence(index)
}

func (builder *OccurrenceStoreBuilder) record(
	occurrence Occurrence,
) (occurrenceStoreRecord, error) {
	display := occurrence.Display()
	startFile, err := builder.displayFile(display.Start.Filename)
	if err != nil {
		return occurrenceStoreRecord{}, err
	}
	endFile, err := builder.displayFile(display.End.Filename)
	if err != nil {
		return occurrenceStoreRecord{}, err
	}
	id := occurrence.ID()
	parent := occurrence.Parent()
	record := occurrenceStoreRecord{
		start: id.Span().Start(), end: id.Span().End(),
		kind: id.KindID(),
		edge: occurrence.Edge(), ordinal: occurrence.Ordinal(),
		startLine:          occurrence.Span().Start.Line,
		startColumn:        occurrence.Span().Start.Column,
		endLine:            occurrence.Span().End.Line,
		endColumn:          occurrence.Span().End.Column,
		displayStartFile:   startFile,
		displayStartLine:   display.Start.Line,
		displayStartColumn: display.Start.Column,
		displayEndFile:     endFile,
		displayEndLine:     display.End.Line,
		displayEndColumn:   display.End.Column,
		token:              occurrence.Token(),
	}
	if !parent.IsZero() {
		record.parentStart = parent.Span().Start()
		record.parentEnd = parent.Span().End()
		record.parentKind = parent.KindID()
	}
	return record, nil
}

func (builder *OccurrenceStoreBuilder) displayFile(
	name string,
) (displayFileIndex, error) {
	if name == "" {
		return 0, fmt.Errorf("occurrence display filename is empty")
	}
	if index := builder.displayByName[name]; index != 0 {
		return index, nil
	}
	if uint64(len(builder.store.displayFiles)) >= uint64(^uint32(0)) {
		return 0, fmt.Errorf("occurrence display-file table overflows uint32")
	}
	builder.store.displayFiles = append(
		builder.store.displayFiles,
		name,
	)
	index := displayFileIndex(len(builder.store.displayFiles))
	builder.displayByName[name] = index
	return index, nil
}

// Seal transfers the normalized immutable store and invalidates the builder.
func (builder *OccurrenceStoreBuilder) Seal() (
	*OccurrenceStore,
	error,
) {
	if builder == nil || builder.sealed || builder.store == nil ||
		len(builder.store.records) == 0 {
		return nil, fmt.Errorf(
			"occurrence store requires one unsealed non-empty payload",
		)
	}
	builder.sealed = true
	builder.store.sealed = true
	store := builder.store
	builder.store = nil
	builder.displayByName = nil
	return store, nil
}

func (store *OccurrenceStore) Count() int {
	if store == nil || !store.sealed {
		return 0
	}
	return len(store.records)
}

func (store *OccurrenceStore) Reference(
	index OccurrenceIndex,
) (OccurrenceRef, error) {
	if store == nil || !store.sealed ||
		!index.valid() || int(index) > len(store.records) {
		return OccurrenceRef{}, fmt.Errorf(
			"occurrence index %d is outside sealed storage",
			index,
		)
	}
	return OccurrenceRef{store: store, index: index}, nil
}

func (store *OccurrenceStore) Visit(
	visit func(OccurrenceRef) error,
) error {
	if store == nil || !store.sealed || visit == nil {
		return fmt.Errorf(
			"occurrence store visit requires sealed storage and visitor",
		)
	}
	for index := 1; index <= len(store.records); index++ {
		if err := visit(OccurrenceRef{
			store: store,
			index: OccurrenceIndex(index),
		}); err != nil {
			return err
		}
	}
	return nil
}

func (store *OccurrenceStore) displayFile(
	index displayFileIndex,
) string {
	if store == nil || index == 0 ||
		int(index) > len(store.displayFiles) {
		return ""
	}
	return store.displayFiles[index-1]
}

func (store *OccurrenceStore) occurrence(
	index OccurrenceIndex,
) Occurrence {
	if store == nil || !index.valid() ||
		int(index) > len(store.records) {
		return Occurrence{}
	}
	record := store.records[index-1]
	id := occurrenceIdentity(
		store.file, record.start, record.end, record.kind,
	)
	var parent identity.OccurrenceID
	if record.parentKind != 0 {
		parent = occurrenceIdentity(
			store.file,
			record.parentStart,
			record.parentEnd,
			record.parentKind,
		)
	}
	return Occurrence{
		id: id, kind: catalog.Kind(record.kind),
		parent: parent, edge: record.edge, ordinal: record.ordinal,
		span: Span{
			Start: Position{
				Line: record.startLine, Column: record.startColumn,
				Offset: record.start,
			},
			End: Position{
				Line: record.endLine, Column: record.endColumn,
				Offset: record.end,
			},
		},
		display: DisplaySpan{
			Start: DisplayPosition{
				Filename: store.displayFile(record.displayStartFile),
				Line:     record.displayStartLine,
				Column:   record.displayStartColumn,
			},
			End: DisplayPosition{
				Filename: store.displayFile(record.displayEndFile),
				Line:     record.displayEndLine,
				Column:   record.displayEndColumn,
			},
		},
		token: record.token,
	}
}

func occurrenceIdentity(
	file identity.FileID,
	start int,
	end int,
	kind uint16,
) identity.OccurrenceID {
	span, err := identity.NewSpanID(file, start, end)
	if err != nil {
		return identity.OccurrenceID{}
	}
	id, err := identity.NewOccurrenceID(span, kind)
	if err != nil {
		return identity.OccurrenceID{}
	}
	return id
}
