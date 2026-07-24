package stagecheck

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
	"github.com/tsoniclang/gotots/internal/language/structure"
	"github.com/tsoniclang/gotots/internal/scope/contract"
)

type occurrenceLedgerRecord struct {
	id      identity.OccurrenceID
	kind    catalog.Kind
	parent  identity.OccurrenceID
	edge    catalog.Edge
	ordinal int
	span    structure.Span
	display structure.DisplaySpan
	token   catalog.TokenKind
}

func occurrenceLedgerRecordFromOccurrence(
	occurrence structure.Occurrence,
) occurrenceLedgerRecord {
	return occurrenceLedgerRecord{
		id:      occurrence.ID(),
		kind:    occurrence.Kind(),
		parent:  occurrence.Parent(),
		edge:    occurrence.Edge(),
		ordinal: occurrence.Ordinal(),
		span:    occurrence.Span(),
		display: occurrence.Display(),
		token:   occurrence.Token(),
	}
}

func occurrenceLedgerRecordFromRef(
	occurrence structure.OccurrenceRef,
) occurrenceLedgerRecord {
	return occurrenceLedgerRecord{
		id:      occurrence.ID(),
		kind:    occurrence.Kind(),
		parent:  occurrence.Parent(),
		edge:    occurrence.Edge(),
		ordinal: occurrence.Ordinal(),
		span:    occurrence.Span(),
		display: occurrence.Display(),
		token:   occurrence.Token(),
	}
}

func occurrenceLedgerRecordFromDerived(
	occurrence derivedOccurrence,
) occurrenceLedgerRecord {
	return occurrenceLedgerRecord{
		id:      occurrence.id,
		kind:    occurrence.kind,
		parent:  occurrence.parent,
		edge:    occurrence.edge,
		ordinal: occurrence.ordinal,
		span:    occurrence.span,
		display: occurrence.display,
		token:   occurrence.token,
	}
}

type ownerMemberLedgerRecord struct {
	owner  structure.OwnerRegionID
	member identity.OccurrenceID
}

type directiveLedgerRecord struct {
	owner   structure.OwnerRegionID
	kind    catalog.DirectiveKind
	tool    string
	name    string
	args    string
	span    structure.Span
	display structure.DisplaySpan
}

type containmentAnchorLedgerRecord struct {
	owner  structure.OwnerRegionID
	anchor identity.OccurrenceID
}

type checkedMappingLedgerRecord struct {
	definition    identity.DefinitionID
	originLine    int
	originColumn  int
	originMatch   structure.CheckedOriginMatch
	checkedDigest string
}

type definitionLedgerRecord struct {
	id       identity.DefinitionID
	owner    structure.OwnerRegionID
	header   identity.HeaderRegionID
	boundary identity.ExecutionBoundaryID
	name     string
}

type definitionSiteLedgerRecord struct {
	kind       structure.DefinitionSiteKind
	definition identity.DefinitionID
	owner      structure.OwnerRegionID
	parent     identity.DefinitionID
	terminal   identity.OccurrenceID
}

type headerLedgerRecord struct {
	id     identity.HeaderRegionID
	digest string
}

type headerMemberLedgerRecord struct {
	header     identity.HeaderRegionID
	ordinal    int
	occurrence identity.OccurrenceID
}

type executionBoundaryLedgerRecord struct {
	id        identity.ExecutionBoundaryID
	kind      structure.ExecutionBoundaryKind
	digest    string
	implicit  identity.ImplicitDefinitionOp
	synthetic identity.SyntheticDefinitionRole
}

type executionEntryLedgerRecord struct {
	boundary   identity.ExecutionBoundaryID
	occurrence identity.OccurrenceID
	hash       string
}

type definitionCensusLedgerRecord struct {
	pkg        identity.PackageID
	definition identity.DefinitionID
}

type certifiedSelectionFactLedgerRecord struct {
	definition identity.DefinitionID
	kind       contract.SelectionFactKind
}

type certifiedFactLedgerRecord struct {
	definition     identity.DefinitionID
	kind           contract.SelectionFactKind
	value          bool
	producerDigest string
	evidenceDigest string
}

func certifiedFactLedgerRecordFromFact(
	fact structure.CertifiedFact,
) certifiedFactLedgerRecord {
	return certifiedFactLedgerRecord{
		definition:     fact.Definition(),
		kind:           fact.Kind(),
		value:          fact.Value(),
		producerDigest: fact.ProducerDigest(),
		evidenceDigest: fact.EvidenceDigest(),
	}
}

func renderOwnerLedgerRecord(record structure.OwnerRegionID) string {
	return record.String()
}

func renderOccurrenceLedgerRecord(record occurrenceLedgerRecord) string {
	return fmt.Sprintf(
		"%s|%d|%s|%d|%d|%s|%s|%d",
		record.id,
		uint16(record.kind),
		record.parent,
		uint16(record.edge),
		record.ordinal,
		renderSpan(record.span),
		renderDisplaySpan(record.display),
		uint16(record.token),
	)
}

func renderOwnerMemberLedgerRecord(record ownerMemberLedgerRecord) string {
	return fmt.Sprintf("%s|%s", record.owner, record.member)
}

func renderDirectiveLedgerRecord(record directiveLedgerRecord) string {
	return fmt.Sprintf(
		"%s|%d|%s|%s|%s|%s|%s",
		record.owner,
		uint16(record.kind),
		record.tool,
		record.name,
		record.args,
		renderSpan(record.span),
		renderDisplaySpan(record.display),
	)
}

func renderContainmentAnchorLedgerRecord(
	record containmentAnchorLedgerRecord,
) string {
	return fmt.Sprintf("%s|%s", record.owner, record.anchor)
}

func renderCheckedMappingLedgerRecord(
	record checkedMappingLedgerRecord,
) string {
	return fmt.Sprintf(
		"%s|%d|%d|%d|%s",
		record.definition,
		record.originLine,
		record.originColumn,
		uint8(record.originMatch),
		record.checkedDigest,
	)
}

func renderDefinitionLedgerRecord(record definitionLedgerRecord) string {
	return fmt.Sprintf(
		"%s|%s|%s|%s|%s",
		record.id,
		record.owner,
		record.header,
		record.boundary,
		record.name,
	)
}

func renderDefinitionSiteLedgerRecord(
	record definitionSiteLedgerRecord,
) string {
	return fmt.Sprintf(
		"%d|%s|%s|%s|%s",
		uint8(record.kind),
		record.definition,
		record.owner,
		record.parent,
		record.terminal,
	)
}

func renderHeaderLedgerRecord(record headerLedgerRecord) string {
	return fmt.Sprintf("%s|%s", record.id, record.digest)
}

func renderHeaderMemberLedgerRecord(record headerMemberLedgerRecord) string {
	return fmt.Sprintf(
		"%s|%d|%s",
		record.header,
		record.ordinal,
		record.occurrence,
	)
}

func renderExecutionBoundaryLedgerRecord(
	record executionBoundaryLedgerRecord,
) string {
	return fmt.Sprintf(
		"%s|%d|%s|%d|%d",
		record.id,
		uint8(record.kind),
		record.digest,
		uint8(record.implicit),
		uint8(record.synthetic),
	)
}

func renderExecutionEntryLedgerRecord(
	record executionEntryLedgerRecord,
) string {
	return fmt.Sprintf(
		"%s|%s|%s",
		record.boundary,
		record.occurrence,
		record.hash,
	)
}

func renderDefinitionCensusLedgerRecord(
	record definitionCensusLedgerRecord,
) string {
	return fmt.Sprintf("%s|%s", record.pkg, record.definition)
}

func renderCertifiedSelectionFactLedgerRecord(
	record certifiedSelectionFactLedgerRecord,
) string {
	return fmt.Sprintf("%s|%d", record.definition, uint8(record.kind))
}

func renderCertifiedFactLedgerRecord(
	record certifiedFactLedgerRecord,
) string {
	return fmt.Sprintf(
		"%s|%d|%t|%s|%s",
		record.definition,
		record.kind,
		record.value,
		record.producerDigest,
		record.evidenceDigest,
	)
}

func renderSpan(span structure.Span) string {
	return fmt.Sprintf(
		"%d:%d:%d-%d:%d:%d",
		span.Start.Line,
		span.Start.Column,
		span.Start.Offset,
		span.End.Line,
		span.End.Column,
		span.End.Offset,
	)
}

func renderDisplaySpan(span structure.DisplaySpan) string {
	return fmt.Sprintf(
		"%s@%d:%d-%s@%d:%d",
		span.Start.Filename,
		span.Start.Line,
		span.Start.Column,
		span.End.Filename,
		span.End.Line,
		span.End.Column,
	)
}
