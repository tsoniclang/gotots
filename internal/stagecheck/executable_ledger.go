package stagecheck

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
	"github.com/tsoniclang/gotots/internal/language/executable"
	"github.com/tsoniclang/gotots/internal/language/structure"
)

type executableLedgerOccurrenceRef uint32
type executableLedgerDefinitionRef uint32

type executableLedgerOccurrenceKey struct {
	file  uint32
	start int
	end   int
	kind  uint16
}

type executableLedgerArena struct {
	files             []identity.FileID
	fileByID          map[identity.FileID]uint32
	occurrences       []executableLedgerOccurrenceKey
	occurrenceByKey   map[executableLedgerOccurrenceKey]executableLedgerOccurrenceRef
	definitions       []identity.DefinitionID
	definitionByID    map[identity.DefinitionID]executableLedgerDefinitionRef
	displayFiles      []string
	displayFileByName map[string]uint32
}

func newExecutableLedgerArena() *executableLedgerArena {
	return &executableLedgerArena{
		fileByID:          map[identity.FileID]uint32{},
		occurrenceByKey:   map[executableLedgerOccurrenceKey]executableLedgerOccurrenceRef{},
		definitionByID:    map[identity.DefinitionID]executableLedgerDefinitionRef{},
		displayFileByName: map[string]uint32{},
	}
}

func (arena *executableLedgerArena) occurrence(
	id identity.OccurrenceID,
) executableLedgerOccurrenceRef {
	if id.IsZero() {
		return 0
	}
	file := id.Span().File()
	fileReference := arena.fileByID[file]
	if fileReference == 0 {
		arena.files = append(arena.files, file)
		fileReference = uint32(len(arena.files))
		arena.fileByID[file] = fileReference
	}
	key := executableLedgerOccurrenceKey{
		file: fileReference, start: id.Span().Start(),
		end: id.Span().End(), kind: id.KindID(),
	}
	if reference := arena.occurrenceByKey[key]; reference != 0 {
		return reference
	}
	arena.occurrences = append(arena.occurrences, key)
	reference := executableLedgerOccurrenceRef(len(arena.occurrences))
	arena.occurrenceByKey[key] = reference
	return reference
}

func (arena *executableLedgerArena) occurrenceID(
	reference executableLedgerOccurrenceRef,
) identity.OccurrenceID {
	if reference == 0 || int(reference) > len(arena.occurrences) {
		return identity.OccurrenceID{}
	}
	record := arena.occurrences[reference-1]
	if record.file == 0 || int(record.file) > len(arena.files) {
		return identity.OccurrenceID{}
	}
	span, err := identity.NewSpanID(
		arena.files[record.file-1],
		record.start,
		record.end,
	)
	if err != nil {
		return identity.OccurrenceID{}
	}
	id, err := identity.NewOccurrenceID(span, record.kind)
	if err != nil {
		return identity.OccurrenceID{}
	}
	return id
}

func (arena *executableLedgerArena) definition(
	id identity.DefinitionID,
) executableLedgerDefinitionRef {
	if id.IsZero() {
		return 0
	}
	if reference := arena.definitionByID[id]; reference != 0 {
		return reference
	}
	arena.definitions = append(arena.definitions, id)
	reference := executableLedgerDefinitionRef(len(arena.definitions))
	arena.definitionByID[id] = reference
	return reference
}

func (arena *executableLedgerArena) definitionID(
	reference executableLedgerDefinitionRef,
) identity.DefinitionID {
	if reference == 0 || int(reference) > len(arena.definitions) {
		return identity.DefinitionID{}
	}
	return arena.definitions[reference-1]
}

func (arena *executableLedgerArena) displayFile(name string) uint32 {
	if reference := arena.displayFileByName[name]; reference != 0 {
		return reference
	}
	arena.displayFiles = append(arena.displayFiles, name)
	reference := uint32(len(arena.displayFiles))
	arena.displayFileByName[name] = reference
	return reference
}

func (arena *executableLedgerArena) displayFileName(
	reference uint32,
) string {
	if reference == 0 || int(reference) > len(arena.displayFiles) {
		return ""
	}
	return arena.displayFiles[reference-1]
}

type compactExecutableOccurrence struct {
	id               executableLedgerOccurrenceRef
	parent           executableLedgerOccurrenceRef
	kind             uint16
	edge             uint16
	ordinal          int
	startLine        int
	startColumn      int
	startOffset      int
	endLine          int
	endColumn        int
	endOffset        int
	displayStartFile uint32
	displayStartLine int
	displayStartCol  int
	displayEndFile   uint32
	displayEndLine   int
	displayEndCol    int
	token            uint16
}

func (arena *executableLedgerArena) occurrenceRecord(
	id identity.OccurrenceID,
	kind catalog.Kind,
	parent identity.OccurrenceID,
	edge catalog.Edge,
	ordinal int,
	span structure.Span,
	display structure.DisplaySpan,
	token catalog.TokenKind,
) compactExecutableOccurrence {
	return compactExecutableOccurrence{
		id:               arena.occurrence(id),
		parent:           arena.occurrence(parent),
		kind:             uint16(kind),
		edge:             uint16(edge),
		ordinal:          ordinal,
		startLine:        span.Start.Line,
		startColumn:      span.Start.Column,
		startOffset:      span.Start.Offset,
		endLine:          span.End.Line,
		endColumn:        span.End.Column,
		endOffset:        span.End.Offset,
		displayStartFile: arena.displayFile(display.Start.Filename),
		displayStartLine: display.Start.Line,
		displayStartCol:  display.Start.Column,
		displayEndFile:   arena.displayFile(display.End.Filename),
		displayEndLine:   display.End.Line,
		displayEndCol:    display.End.Column,
		token:            uint16(token),
	}
}

func (arena *executableLedgerArena) recordFromRef(
	occurrence structure.OccurrenceRef,
) compactExecutableOccurrence {
	return arena.occurrenceRecord(
		occurrence.ID(),
		occurrence.Kind(),
		occurrence.Parent(),
		occurrence.Edge(),
		occurrence.Ordinal(),
		occurrence.Span(),
		occurrence.Display(),
		occurrence.Token(),
	)
}

func (arena *executableLedgerArena) recordFromDerived(
	occurrence derivedOccurrence,
) compactExecutableOccurrence {
	return arena.occurrenceRecord(
		occurrence.id,
		occurrence.kind,
		occurrence.parent,
		occurrence.edge,
		occurrence.ordinal,
		occurrence.span,
		occurrence.display,
		occurrence.token,
	)
}

type compactExecutableMember struct {
	region     executableLedgerDefinitionRef
	ordinal    int
	occurrence executableLedgerOccurrenceRef
}

type compactDefinitionReference struct {
	region  executableLedgerDefinitionRef
	parent  executableLedgerOccurrenceRef
	edge    uint16
	ordinal int
	child   executableLedgerDefinitionRef
}

type compactImplicitOperation struct {
	region executableLedgerDefinitionRef
	kind   executable.ImplicitOperationKind
	pkg    identity.PackageID
}

type compactExecutableLedger struct {
	arena                 *executableLedgerArena
	additionalOccurrences recordMultiset[compactExecutableOccurrence]
	regions               recordMultiset[executableLedgerDefinitionRef]
	members               recordMultiset[compactExecutableMember]
	definitionReferences  recordMultiset[compactDefinitionReference]
	implicitOperations    recordMultiset[compactImplicitOperation]
}

func newCompactExecutableLedger(
	arena *executableLedgerArena,
) *compactExecutableLedger {
	return &compactExecutableLedger{arena: arena}
}

func compareCompactExecutableLedgers(
	stage string,
	actual *compactExecutableLedger,
	expected *compactExecutableLedger,
) error {
	difference := newCompactExecutableLedger(actual.arena)
	mergeExecutableRecordDifference(
		&difference.additionalOccurrences,
		actual.additionalOccurrences,
		1,
	)
	mergeExecutableRecordDifference(
		&difference.additionalOccurrences,
		expected.additionalOccurrences,
		-1,
	)
	mergeExecutableRecordDifference(
		&difference.regions,
		actual.regions,
		1,
	)
	mergeExecutableRecordDifference(
		&difference.regions,
		expected.regions,
		-1,
	)
	mergeExecutableRecordDifference(
		&difference.members,
		actual.members,
		1,
	)
	mergeExecutableRecordDifference(
		&difference.members,
		expected.members,
		-1,
	)
	mergeExecutableRecordDifference(
		&difference.definitionReferences,
		actual.definitionReferences,
		1,
	)
	mergeExecutableRecordDifference(
		&difference.definitionReferences,
		expected.definitionReferences,
		-1,
	)
	mergeExecutableRecordDifference(
		&difference.implicitOperations,
		actual.implicitOperations,
		1,
	)
	mergeExecutableRecordDifference(
		&difference.implicitOperations,
		expected.implicitOperations,
		-1,
	)
	return verifyCompactExecutableLedgerDifference(stage, difference)
}

func verifyCompactExecutableLedgerDifference(
	stage string,
	difference *compactExecutableLedger,
) error {
	problems := newProblemSet()
	addExecutableRecordDifferences(
		problems,
		"executable-additional-occurrence",
		difference.additionalOccurrences,
		difference.renderOccurrence,
	)
	addExecutableRecordDifferences(
		problems,
		"executable-region",
		difference.regions,
		difference.renderRegion,
	)
	addExecutableRecordDifferences(
		problems,
		"executable-member",
		difference.members,
		difference.renderMember,
	)
	addExecutableRecordDifferences(
		problems,
		"executable-definition-reference",
		difference.definitionReferences,
		difference.renderDefinitionReference,
	)
	addExecutableRecordDifferences(
		problems,
		"executable-implicit-operation",
		difference.implicitOperations,
		difference.renderImplicitOperation,
	)
	return problems.verificationError(
		stage, "exact executable join failed",
	)
}

func adjustExecutableRecordDifference[Record comparable](
	records *recordMultiset[Record],
	record Record,
	delta int,
) {
	if delta == 0 {
		return
	}
	if *records == nil {
		*records = recordMultiset[Record]{}
	}
	count := (*records)[record] + delta
	if count == 0 {
		delete(*records, record)
		return
	}
	(*records)[record] = count
}

func mergeExecutableRecordDifference[Record comparable](
	target *recordMultiset[Record],
	source recordMultiset[Record],
	delta int,
) {
	for record, count := range source {
		adjustExecutableRecordDifference(
			target,
			record,
			count*delta,
		)
	}
}

func addExecutableRecordDifferences[Record comparable](
	problems *problemSet,
	class string,
	differences recordMultiset[Record],
	render func(Record) string,
) {
	for record, difference := range differences {
		expected := 0
		actual := difference
		if difference < 0 {
			expected = -difference
			actual = 0
		}
		problems.addf(
			"%s|%s|expected=%d|actual=%d",
			class,
			render(record),
			expected,
			actual,
		)
	}
}

func (ledger *compactExecutableLedger) renderOccurrence(
	record compactExecutableOccurrence,
) string {
	return fmt.Sprintf(
		"%s|%d|%s|%d|%d|%d:%d:%d-%d:%d:%d|%s@%d:%d-%s@%d:%d|%d",
		ledger.arena.occurrenceID(record.id),
		record.kind,
		ledger.arena.occurrenceID(record.parent),
		record.edge,
		record.ordinal,
		record.startLine,
		record.startColumn,
		record.startOffset,
		record.endLine,
		record.endColumn,
		record.endOffset,
		ledger.arena.displayFileName(record.displayStartFile),
		record.displayStartLine,
		record.displayStartCol,
		ledger.arena.displayFileName(record.displayEndFile),
		record.displayEndLine,
		record.displayEndCol,
		record.token,
	)
}

func (ledger *compactExecutableLedger) renderRegion(
	reference executableLedgerDefinitionRef,
) string {
	return ledger.arena.definitionID(reference).String()
}

func (ledger *compactExecutableLedger) renderMember(
	record compactExecutableMember,
) string {
	return fmt.Sprintf(
		"%s|%d|%s",
		ledger.arena.definitionID(record.region),
		record.ordinal,
		ledger.arena.occurrenceID(record.occurrence),
	)
}

func (ledger *compactExecutableLedger) renderDefinitionReference(
	record compactDefinitionReference,
) string {
	return fmt.Sprintf(
		"%s|%s|%d|%d|%s",
		ledger.arena.definitionID(record.region),
		ledger.arena.occurrenceID(record.parent),
		record.edge,
		record.ordinal,
		ledger.arena.definitionID(record.child),
	)
}

func (ledger *compactExecutableLedger) renderImplicitOperation(
	record compactImplicitOperation,
) string {
	return fmt.Sprintf(
		"%s|%d|%s",
		ledger.arena.definitionID(record.region),
		record.kind,
		record.pkg,
	)
}
