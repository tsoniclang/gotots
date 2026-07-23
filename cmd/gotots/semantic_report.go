package main

import (
	"fmt"
	"io"

	"github.com/tsoniclang/gotots/internal/language/frontend"
	"github.com/tsoniclang/gotots/internal/language/semantic"
)

func printSemanticWork(
	output io.Writer,
	category string,
	work frontend.Work,
) error {
	_, err := fmt.Fprintf(
		output,
		"semantic-%s-work: packages=%d inputOccurrences=%d childEdges=%d contexts=%d objectVisits=%d implicitBindingVisits=%d intrinsicVisits=%d captureVisits=%d resolutionVisits=%d containmentVisits=%d containmentEdges=%d memberTypeVisits=%d containmentProbes=%d occurrenceScopeProbes=%d checkerScopeProbes=%d typeConstructions=%d objectConstructions=%d operationConstructions=%d resolutions=%d containmentEntries=%d canonicalSortInputs=%d linearOperations=%d\n",
		category,
		work.Packages,
		work.InputOccurrences,
		work.ChildEdgeAssignments,
		work.ContextAssignments,
		work.ObjectOccurrenceVisits,
		work.ImplicitBindingVisits,
		work.IntrinsicOccurrenceVisits,
		work.CaptureOccurrenceVisits,
		work.ResolutionVisits,
		work.DefinitionContainmentVisits,
		work.DefinitionContainmentEdges,
		work.MemberTypeVisits,
		work.ContainmentProbes,
		work.OccurrenceScopeProbes,
		work.CheckerScopeProbes,
		work.TypeConstructions,
		work.ObjectConstructions,
		work.OperationConstructions,
		work.OccurrenceResolutions,
		work.DefinitionContainmentEntries,
		work.CanonicalSortInputs,
		work.LinearOperations(),
	)
	return err
}

func printSemanticMetrics(
	output io.Writer,
	category string,
	metrics semantic.Metrics,
) error {
	if _, err := fmt.Fprintf(
		output,
		"semantic-%s: packages=%d definitions=%d resolutions=%d declarations=%d bindings=%d types=%d operations=%d unsupported=%d encodedBytes=%d\n",
		category,
		metrics.Packages(),
		metrics.Definitions(),
		metrics.Resolutions(),
		metrics.Declarations(),
		metrics.Bindings(),
		metrics.Types(),
		metrics.Operations(),
		metrics.Unsupported(),
		metrics.EncodedBytes(),
	); err != nil {
		return err
	}
	for index, record := range metrics.LargestPackages() {
		if _, err := fmt.Fprintf(
			output,
			"semantic-%s-package-tail rank=%d package=%s encodedBytes=%d records=%d\n",
			category,
			index+1,
			record.Package,
			record.EncodedBytes,
			record.Records,
		); err != nil {
			return err
		}
	}
	for index, record := range metrics.LargestDefinitions() {
		if err := printSemanticRecord(
			output, category, "definition", index, record,
		); err != nil {
			return err
		}
	}
	for index, record := range metrics.LargestOperations() {
		if err := printSemanticRecord(
			output, category, "operation", index, record,
		); err != nil {
			return err
		}
	}
	for index, record := range metrics.LargestTypes() {
		if err := printSemanticRecord(
			output, category, "type", index, record,
		); err != nil {
			return err
		}
	}
	return nil
}

func printSemanticRecord(
	output io.Writer,
	category string,
	class string,
	index int,
	record semantic.RecordSize,
) error {
	_, err := fmt.Fprintf(
		output,
		"semantic-%s-%s-tail rank=%d package=%s identity=%s encodedBytes=%d\n",
		category,
		class,
		index+1,
		record.Package,
		record.Identity,
		record.EncodedBytes,
	)
	return err
}
