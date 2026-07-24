package stagecheck

import (
	"github.com/tsoniclang/gotots/internal/language/structure"
)

type recordMultiset[Record comparable] map[Record]int

func addRecord[Record comparable](
	records *recordMultiset[Record],
	record Record,
) {
	if *records == nil {
		*records = recordMultiset[Record]{}
	}
	(*records)[record]++
}

func mergeRecords[Record comparable](
	target *recordMultiset[Record],
	source recordMultiset[Record],
) {
	for record, count := range source {
		if *target == nil {
			*target = recordMultiset[Record]{}
		}
		(*target)[record] += count
	}
}

type structuralLedger struct {
	owners                   recordMultiset[structure.OwnerRegionID]
	occurrences              recordMultiset[occurrenceLedgerRecord]
	ownerMembers             recordMultiset[ownerMemberLedgerRecord]
	directives               recordMultiset[directiveLedgerRecord]
	containmentGraphs        recordMultiset[structure.OwnerRegionID]
	containmentAnchors       recordMultiset[containmentAnchorLedgerRecord]
	checkedMappings          recordMultiset[checkedMappingLedgerRecord]
	definitions              recordMultiset[definitionLedgerRecord]
	definitionSites          recordMultiset[definitionSiteLedgerRecord]
	headers                  recordMultiset[headerLedgerRecord]
	headerMembers            recordMultiset[headerMemberLedgerRecord]
	executionBoundaries      recordMultiset[executionBoundaryLedgerRecord]
	executionEntries         recordMultiset[executionEntryLedgerRecord]
	additionalOccurrences    recordMultiset[occurrenceLedgerRecord]
	executableRegions        recordMultiset[executableRegionLedgerRecord]
	executableMembers        recordMultiset[executableMemberLedgerRecord]
	definitionReferences     recordMultiset[definitionReferenceLedgerRecord]
	implicitOperations       recordMultiset[implicitOperationLedgerRecord]
	definitionCensus         recordMultiset[definitionCensusLedgerRecord]
	providerDefinitionCensus recordMultiset[definitionCensusLedgerRecord]
	certifiedSelectionFacts  recordMultiset[certifiedSelectionFactLedgerRecord]
	certifiedFacts           recordMultiset[certifiedFactLedgerRecord]
}

func newStructuralLedger() *structuralLedger {
	return &structuralLedger{}
}

func (ledger *structuralLedger) merge(other *structuralLedger) {
	mergeRecords(&ledger.owners, other.owners)
	mergeRecords(&ledger.occurrences, other.occurrences)
	mergeRecords(&ledger.ownerMembers, other.ownerMembers)
	mergeRecords(&ledger.directives, other.directives)
	mergeRecords(&ledger.containmentGraphs, other.containmentGraphs)
	mergeRecords(&ledger.containmentAnchors, other.containmentAnchors)
	mergeRecords(&ledger.checkedMappings, other.checkedMappings)
	mergeRecords(&ledger.definitions, other.definitions)
	mergeRecords(&ledger.definitionSites, other.definitionSites)
	mergeRecords(&ledger.headers, other.headers)
	mergeRecords(&ledger.headerMembers, other.headerMembers)
	mergeRecords(&ledger.executionBoundaries, other.executionBoundaries)
	mergeRecords(&ledger.executionEntries, other.executionEntries)
	mergeRecords(&ledger.additionalOccurrences, other.additionalOccurrences)
	mergeRecords(&ledger.executableRegions, other.executableRegions)
	mergeRecords(&ledger.executableMembers, other.executableMembers)
	mergeRecords(&ledger.definitionReferences, other.definitionReferences)
	mergeRecords(&ledger.implicitOperations, other.implicitOperations)
	mergeRecords(&ledger.definitionCensus, other.definitionCensus)
	mergeRecords(
		&ledger.providerDefinitionCensus,
		other.providerDefinitionCensus,
	)
	mergeRecords(
		&ledger.certifiedSelectionFacts,
		other.certifiedSelectionFacts,
	)
	mergeRecords(&ledger.certifiedFacts, other.certifiedFacts)
}

func compareLedgerClass[Record comparable](
	problems *problemSet,
	class string,
	actual recordMultiset[Record],
	expected recordMultiset[Record],
	render func(Record) string,
) {
	for record, actualCount := range actual {
		expectedCount, present := expected[record]
		if present && actualCount == expectedCount {
			continue
		}
		problems.addf(
			"%s|%s|expected=%d|actual=%d",
			class,
			render(record),
			expectedCount,
			actualCount,
		)
	}
	for record, expectedCount := range expected {
		if _, present := actual[record]; present {
			continue
		}
		problems.addf(
			"%s|%s|expected=%d|actual=0",
			class,
			render(record),
			expectedCount,
		)
	}
}

func compareLedgers(stage string, actual, expected *structuralLedger) error {
	problems := newProblemSet()
	compareLedgerClass(
		problems, "owner", actual.owners, expected.owners,
		renderOwnerLedgerRecord,
	)
	compareLedgerClass(
		problems, "occurrence", actual.occurrences, expected.occurrences,
		renderOccurrenceLedgerRecord,
	)
	compareLedgerClass(
		problems, "owner-member", actual.ownerMembers, expected.ownerMembers,
		renderOwnerMemberLedgerRecord,
	)
	compareLedgerClass(
		problems, "directive", actual.directives, expected.directives,
		renderDirectiveLedgerRecord,
	)
	compareLedgerClass(
		problems, "containment-graph",
		actual.containmentGraphs, expected.containmentGraphs,
		renderOwnerLedgerRecord,
	)
	compareLedgerClass(
		problems, "containment-anchor",
		actual.containmentAnchors, expected.containmentAnchors,
		renderContainmentAnchorLedgerRecord,
	)
	compareLedgerClass(
		problems, "checked-mapping",
		actual.checkedMappings, expected.checkedMappings,
		renderCheckedMappingLedgerRecord,
	)
	compareLedgerClass(
		problems, "definition", actual.definitions, expected.definitions,
		renderDefinitionLedgerRecord,
	)
	compareLedgerClass(
		problems, "definition-site",
		actual.definitionSites, expected.definitionSites,
		renderDefinitionSiteLedgerRecord,
	)
	compareLedgerClass(
		problems, "header", actual.headers, expected.headers,
		renderHeaderLedgerRecord,
	)
	compareLedgerClass(
		problems, "header-member",
		actual.headerMembers, expected.headerMembers,
		renderHeaderMemberLedgerRecord,
	)
	compareLedgerClass(
		problems, "execution-boundary",
		actual.executionBoundaries, expected.executionBoundaries,
		renderExecutionBoundaryLedgerRecord,
	)
	compareLedgerClass(
		problems, "execution-entry",
		actual.executionEntries, expected.executionEntries,
		renderExecutionEntryLedgerRecord,
	)
	compareLedgerClass(
		problems, "executable-additional-occurrence",
		actual.additionalOccurrences, expected.additionalOccurrences,
		renderOccurrenceLedgerRecord,
	)
	compareLedgerClass(
		problems, "executable-region",
		actual.executableRegions, expected.executableRegions,
		renderExecutableRegionLedgerRecord,
	)
	compareLedgerClass(
		problems, "executable-member",
		actual.executableMembers, expected.executableMembers,
		renderExecutableMemberLedgerRecord,
	)
	compareLedgerClass(
		problems, "executable-definition-reference",
		actual.definitionReferences, expected.definitionReferences,
		renderDefinitionReferenceLedgerRecord,
	)
	compareLedgerClass(
		problems, "executable-implicit-operation",
		actual.implicitOperations, expected.implicitOperations,
		renderImplicitOperationLedgerRecord,
	)
	compareLedgerClass(
		problems, "definition-census",
		actual.definitionCensus, expected.definitionCensus,
		renderDefinitionCensusLedgerRecord,
	)
	compareLedgerClass(
		problems, "provider-definition-census",
		actual.providerDefinitionCensus,
		expected.providerDefinitionCensus,
		renderDefinitionCensusLedgerRecord,
	)
	compareLedgerClass(
		problems, "certified-selection-fact",
		actual.certifiedSelectionFacts,
		expected.certifiedSelectionFacts,
		renderCertifiedSelectionFactLedgerRecord,
	)
	compareLedgerClass(
		problems, "certified-fact",
		actual.certifiedFacts, expected.certifiedFacts,
		renderCertifiedFactLedgerRecord,
	)
	return problems.verificationError(stage, "exact structural join failed")
}

func ledgerForPackage(pkg structure.PackageGraph) *structuralLedger {
	ledger := newStructuralLedger()
	for _, file := range pkg.Files() {
		addFileGraph(ledger, file)
	}
	for _, owner := range pkg.SyntheticOwners() {
		addRecord(&ledger.owners, owner.ID())
	}
	addDefinitionRecords(
		ledger,
		pkg.Definitions(),
		pkg.Sites(),
		pkg.Headers(),
		pkg.Boundaries(),
		true,
	)
	return ledger
}

func ledgerForFile(file structure.FileGraph) *structuralLedger {
	ledger := newStructuralLedger()
	addFileGraph(ledger, file)
	return ledger
}

func addFileGraph(ledger *structuralLedger, file structure.FileGraph) {
	owner := file.Owner()
	addRecord(&ledger.owners, owner.ID())
	for _, occurrence := range file.Occurrences() {
		addRecord(
			&ledger.occurrences,
			occurrenceLedgerRecordFromOccurrence(occurrence),
		)
	}
	for _, member := range owner.Members() {
		addRecord(&ledger.ownerMembers, ownerMemberLedgerRecord{
			owner: owner.ID(), member: member,
		})
	}
	for _, directive := range owner.Directives() {
		addRecord(&ledger.directives, directiveLedgerRecord{
			owner:   owner.ID(),
			kind:    directive.Kind(),
			tool:    directive.Tool(),
			name:    directive.Name(),
			args:    directive.Args(),
			span:    directive.Span(),
			display: directive.Display(),
		})
	}
	containment := file.Containment()
	addRecord(&ledger.containmentGraphs, containment.Owner())
	for _, anchor := range containment.Anchors() {
		addRecord(
			&ledger.containmentAnchors,
			containmentAnchorLedgerRecord{
				owner: containment.Owner(), anchor: anchor,
			},
		)
	}
	addDefinitionRecords(
		ledger,
		file.Definitions(),
		file.Sites(),
		file.Headers(),
		file.Boundaries(),
		false,
	)
	for _, mapping := range file.CheckedMappings() {
		addRecord(&ledger.checkedMappings, checkedMappingLedgerRecord{
			definition:    mapping.Definition(),
			originLine:    mapping.OriginLine(),
			originColumn:  mapping.OriginColumn(),
			originMatch:   mapping.OriginMatch(),
			checkedDigest: mapping.CheckedDigest(),
		})
	}
}

func addDefinitionRecords(
	ledger *structuralLedger,
	definitions []structure.ImplementationDefinition,
	sites []structure.DefinitionSite,
	headers []structure.HeaderRegion,
	boundaries []structure.ExecutionBoundary,
	implicitOnly bool,
) {
	for _, definition := range definitions {
		if implicitOnly && !definition.ID().File().IsZero() {
			continue
		}
		addRecord(&ledger.definitions, definitionLedgerRecord{
			id:       definition.ID(),
			owner:    definition.Owner(),
			header:   definition.Header(),
			boundary: definition.Boundary(),
			name:     definition.Name(),
		})
	}
	for _, site := range sites {
		if implicitOnly && !site.Definition().File().IsZero() {
			continue
		}
		addRecord(&ledger.definitionSites, definitionSiteLedgerRecord{
			kind:       site.Kind(),
			definition: site.Definition(),
			owner:      site.Owner(),
			parent:     site.ParentDefinition(),
			terminal:   site.Terminal(),
		})
	}
	for _, header := range headers {
		if implicitOnly && !header.ID().Definition().File().IsZero() {
			continue
		}
		addRecord(&ledger.headers, headerLedgerRecord{
			id: header.ID(), digest: header.Digest(),
		})
		for index, occurrence := range header.Members() {
			addRecord(&ledger.headerMembers, headerMemberLedgerRecord{
				header: header.ID(), ordinal: index, occurrence: occurrence,
			})
		}
	}
	for _, boundary := range boundaries {
		if implicitOnly && !boundary.ID().Definition().File().IsZero() {
			continue
		}
		addRecord(
			&ledger.executionBoundaries,
			executionBoundaryLedgerRecord{
				id:        boundary.ID(),
				kind:      boundary.Kind(),
				digest:    boundary.CombinedDigest(),
				implicit:  boundary.ImplicitOp(),
				synthetic: boundary.SyntheticRole(),
			},
		)
		for _, entry := range boundary.Entries() {
			addRecord(&ledger.executionEntries, executionEntryLedgerRecord{
				boundary:   boundary.ID(),
				occurrence: entry.ID(),
				hash:       entry.Hash(),
			})
		}
	}
}
