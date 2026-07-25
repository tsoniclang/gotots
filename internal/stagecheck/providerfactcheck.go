package stagecheck

import (
	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/structure"
	"github.com/tsoniclang/gotots/internal/scope/contract"
	"github.com/tsoniclang/gotots/internal/scope/sourceplan"
)

func verifyCertifiedSelectionFacts(
	plan *sourceplan.Plan,
	graph *structure.Graph,
	selected contract.Contract,
	artifact *structure.ProviderArtifact,
) error {
	if plan == nil {
		return nil
	}
	expected := newStructuralLedger()
	actual := newStructuralLedger()
	packages := map[identity.PackageID]bool{}
	for _, record := range graph.DefinitionCensus() {
		packages[record.Package()] = true
		if !definitionUsesCertifiedGraph(
			plan, record.ID(), record.Package(),
		) {
			continue
		}
		for _, kind := range selected.RequestedFacts(
			record.ID(), record.Package(),
		) {
			addRecord(
				&expected.certifiedSelectionFacts,
				certifiedSelectionFactLedgerRecord{
					definition: record.ID(), kind: kind,
				},
			)
		}
	}
	for packageID := range packages {
		if artifact == nil {
			continue
		}
		if _, present := artifact.PackageInputDigest(packageID); !present {
			continue
		}
		facts := artifact.CertifiedFactsForPackage(packageID)
		for _, fact := range facts {
			if !definitionUsesCertifiedGraph(
				plan, fact.Definition(), packageID,
			) {
				continue
			}
			addRecord(
				&actual.certifiedSelectionFacts,
				certifiedSelectionFactLedgerRecord{
					definition: fact.Definition(), kind: fact.Kind(),
				},
			)
		}
	}
	return compareLedgers(
		"certified-selection-fact",
		actual,
		expected,
	)
}

func definitionUsesCertifiedGraph(
	plan *sourceplan.Plan,
	definition identity.DefinitionID,
	pkg identity.PackageID,
) bool {
	if definition.ImplicitOp().Valid() {
		return false
	}
	if definition.SyntheticRole().Valid() {
		decision, present := plan.SyntheticFor(pkg)
		return present &&
			decision.Kind() == sourceplan.KindCertifiedGraph
	}
	decision, present := plan.For(definition.File())
	return present && decision.Kind() == sourceplan.KindCertifiedGraph
}
