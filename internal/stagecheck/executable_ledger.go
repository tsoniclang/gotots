package stagecheck

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/executable"
	"github.com/tsoniclang/gotots/internal/language/structure"
)

type executableLedgerDefinitionRef uint32

type executableLedgerArena struct {
	definitions    []identity.DefinitionID
	definitionByID map[identity.DefinitionID]executableLedgerDefinitionRef
}

func newExecutableLedgerArena() *executableLedgerArena {
	return &executableLedgerArena{
		definitionByID: map[identity.DefinitionID]executableLedgerDefinitionRef{},
	}
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

type compactExecutableMember struct {
	region     executableLedgerDefinitionRef
	ordinal    int
	occurrence structure.OccurrenceRef
}

type compactDefinitionReference struct {
	region  executableLedgerDefinitionRef
	parent  identity.OccurrenceID
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
	additionalOccurrences recordMultiset[structure.OccurrenceRef]
	regions               recordMultiset[executableLedgerDefinitionRef]
	members               recordMultiset[compactExecutableMember]
	definitionReferences  recordMultiset[compactDefinitionReference]
	implicitOperations    recordMultiset[compactImplicitOperation]
}

type compactExecutableCapacity struct {
	additionalOccurrences int
	regions               int
	members               int
	definitionReferences  int
	implicitOperations    int
}

func newCompactExecutableLedger(
	arena *executableLedgerArena,
) *compactExecutableLedger {
	return newSizedCompactExecutableLedger(
		arena, compactExecutableCapacity{},
	)
}

func newSizedCompactExecutableLedger(
	arena *executableLedgerArena,
	capacity compactExecutableCapacity,
) *compactExecutableLedger {
	if capacity.additionalOccurrences < 0 ||
		capacity.regions < 0 ||
		capacity.members < 0 ||
		capacity.definitionReferences < 0 ||
		capacity.implicitOperations < 0 {
		panic("compact executable ledger has negative capacity")
	}
	ledger := &compactExecutableLedger{arena: arena}
	if capacity.additionalOccurrences != 0 {
		ledger.additionalOccurrences = make(
			recordMultiset[structure.OccurrenceRef],
			capacity.additionalOccurrences,
		)
	}
	if capacity.regions != 0 {
		ledger.regions = make(
			recordMultiset[executableLedgerDefinitionRef],
			capacity.regions,
		)
	}
	if capacity.members != 0 {
		ledger.members = make(
			recordMultiset[compactExecutableMember],
			capacity.members,
		)
	}
	if capacity.definitionReferences != 0 {
		ledger.definitionReferences = make(
			recordMultiset[compactDefinitionReference],
			capacity.definitionReferences,
		)
	}
	if capacity.implicitOperations != 0 {
		ledger.implicitOperations = make(
			recordMultiset[compactImplicitOperation],
			capacity.implicitOperations,
		)
	}
	return ledger
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
	record structure.OccurrenceRef,
) string {
	return record.ID().String()
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
		record.occurrence.ID(),
	)
}

func (ledger *compactExecutableLedger) renderDefinitionReference(
	record compactDefinitionReference,
) string {
	return fmt.Sprintf(
		"%s|%s|%d|%d|%s",
		ledger.arena.definitionID(record.region),
		record.parent,
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
