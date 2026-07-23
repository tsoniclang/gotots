package stagecheck

import (
	"crypto/sha256"
	"fmt"
	"sort"

	"github.com/tsoniclang/gotots/internal/language/structure"
)

type structuralLedger struct {
	classes map[string]map[string]int
}

func newStructuralLedger() *structuralLedger {
	return &structuralLedger{classes: map[string]map[string]int{}}
}

func (l *structuralLedger) add(class, record string) {
	if l.classes[class] == nil {
		l.classes[class] = map[string]int{}
	}
	l.classes[class][record]++
}

func (l *structuralLedger) merge(other *structuralLedger) {
	for class, records := range other.classes {
		if l.classes[class] == nil {
			l.classes[class] = map[string]int{}
		}
		for record, count := range records {
			l.classes[class][record] += count
		}
	}
}

func compareLedgers(stage string, actual, expected *structuralLedger) error {
	classSet := map[string]bool{}
	for class := range actual.classes {
		classSet[class] = true
	}
	for class := range expected.classes {
		classSet[class] = true
	}
	var classes []string
	for class := range classSet {
		classes = append(classes, class)
	}
	sort.Strings(classes)
	const sampleLimit = 20
	residualHash := sha256.New()
	var samples []string
	residuals := 0
	for _, class := range classes {
		recordSet := map[string]bool{}
		for record := range expected.classes[class] {
			recordSet[record] = true
		}
		for record := range actual.classes[class] {
			recordSet[record] = true
		}
		records := make([]string, 0, len(recordSet))
		for record := range recordSet {
			records = append(records, record)
		}
		sort.Strings(records)
		for _, record := range records {
			expectedCount := expected.classes[class][record]
			actualCount := actual.classes[class][record]
			if expectedCount == actualCount {
				continue
			}
			residual := fmt.Sprintf(
				"%s|%s|expected=%d|actual=%d",
				class, record, expectedCount, actualCount,
			)
			fmt.Fprintln(residualHash, residual)
			residuals++
			if len(samples) < sampleLimit {
				samples = append(samples, residual)
			}
		}
	}
	if residuals == 0 {
		return nil
	}
	return &VerificationError{
		Stage: stage,
		Reason: fmt.Sprintf(
			"exact structural join failed (residuals=%d digest=%x sample=%v)",
			residuals, residualHash.Sum(nil), samples,
		),
	}
}

func ledgerForPackage(pkg structure.PackageGraph) *structuralLedger {
	ledger := newStructuralLedger()
	for _, file := range pkg.Files() {
		addFileGraph(ledger, file)
	}
	for _, owner := range pkg.SyntheticOwners() {
		ledger.add("owner", owner.ID().String())
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
	ledger.add("owner", owner.ID().String())
	for _, occurrence := range file.Occurrences() {
		ledger.add(
			"occurrence",
			occurrenceKey(occurrence),
		)
	}
	for _, member := range owner.Members() {
		ledger.add(
			"owner-member",
			fmt.Sprintf("%s|%s", owner.ID(), member),
		)
	}
	for _, directive := range owner.Directives() {
		ledger.add(
			"directive",
			fmt.Sprintf(
				"%s|%d|%s|%s|%s|%s|%s",
				owner.ID(),
				uint16(directive.Kind()),
				directive.Tool(),
				directive.Name(),
				directive.Args(),
				spanKey(directive.Span()),
				displayKey(directive.Display()),
			),
		)
	}
	containment := file.Containment()
	ledger.add("containment-graph", containment.Owner().String())
	for _, anchor := range containment.Anchors() {
		ledger.add(
			"containment-anchor",
			fmt.Sprintf("%s|%s", containment.Owner(), anchor),
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
		ledger.add(
			"checked-mapping",
			fmt.Sprintf(
				"%s|%d|%d|%d|%s",
				mapping.Definition(),
				mapping.OriginLine(),
				mapping.OriginColumn(),
				uint8(mapping.OriginMatch()),
				mapping.CheckedDigest(),
			),
		)
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
		ledger.add(
			"definition",
			fmt.Sprintf(
				"%s|%s|%s|%s|%s",
				definition.ID(),
				definition.Owner(),
				definition.Header(),
				definition.Boundary(),
				definition.Name(),
			),
		)
	}
	for _, site := range sites {
		if implicitOnly && !site.Definition().File().IsZero() {
			continue
		}
		ledger.add(
			"definition-site",
			fmt.Sprintf(
				"%d|%s|%s|%s|%s",
				uint8(site.Kind()),
				site.Definition(),
				site.Owner(),
				site.ParentDefinition(),
				site.Terminal(),
			),
		)
	}
	for _, header := range headers {
		if implicitOnly && !header.ID().Definition().File().IsZero() {
			continue
		}
		ledger.add(
			"header",
			fmt.Sprintf("%s|%s", header.ID(), header.Digest()),
		)
		for index, occurrence := range header.Members() {
			ledger.add(
				"header-member",
				fmt.Sprintf("%s|%d|%s", header.ID(), index, occurrence),
			)
		}
	}
	for _, boundary := range boundaries {
		if implicitOnly && !boundary.ID().Definition().File().IsZero() {
			continue
		}
		ledger.add(
			"execution-boundary",
			fmt.Sprintf(
				"%s|%d|%s|%d|%d",
				boundary.ID(),
				uint8(boundary.Kind()),
				boundary.CombinedDigest(),
				uint8(boundary.ImplicitOp()),
				uint8(boundary.SyntheticRole()),
			),
		)
		for _, entry := range boundary.Entries() {
			ledger.add(
				"execution-entry",
				fmt.Sprintf(
					"%s|%s|%s",
					boundary.ID(),
					entry.ID(),
					entry.Hash(),
				),
			)
		}
	}
}

func occurrenceKey(occurrence structure.Occurrence) string {
	return fmt.Sprintf(
		"%s|%d|%s|%d|%d|%s|%s|%d",
		occurrence.ID(),
		uint16(occurrence.Kind()),
		occurrence.Parent(),
		uint16(occurrence.Edge()),
		occurrence.Ordinal(),
		spanKey(occurrence.Span()),
		displayKey(occurrence.Display()),
		uint16(occurrence.Token()),
	)
}

func spanKey(span structure.Span) string {
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

func displayKey(span structure.DisplaySpan) string {
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
