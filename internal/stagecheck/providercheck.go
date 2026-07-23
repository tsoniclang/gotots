package stagecheck

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/language/structure"
)

func syntheticLedger(pkg structure.PackageGraph) *structuralLedger {
	ledger := newStructuralLedger()
	for _, owner := range pkg.SyntheticOwners() {
		if owner.ID().SyntheticKind() ==
			structure.SyntheticOwnerCgoGenerated {
			ledger.add("owner", owner.ID().String())
		}
	}
	var definitions []structure.ImplementationDefinition
	var sites []structure.DefinitionSite
	var headers []structure.HeaderRegion
	var boundaries []structure.ExecutionBoundary
	for _, definition := range pkg.Definitions() {
		if definition.ID().SyntheticRole().Valid() {
			definitions = append(definitions, definition)
		}
	}
	for _, site := range pkg.Sites() {
		if site.Definition().SyntheticRole().Valid() {
			sites = append(sites, site)
		}
	}
	for _, header := range pkg.Headers() {
		if header.ID().Definition().SyntheticRole().Valid() {
			headers = append(headers, header)
		}
	}
	for _, boundary := range pkg.Boundaries() {
		if boundary.ID().Definition().SyntheticRole().Valid() {
			boundaries = append(boundaries, boundary)
		}
	}
	addDefinitionRecords(
		ledger, definitions, sites, headers, boundaries, false,
	)
	return ledger
}

func certifiedFactKey(fact structure.CertifiedFact) string {
	return fmt.Sprintf(
		"%s|%d|%t|%s|%s",
		fact.Definition(),
		fact.Kind(),
		fact.Value(),
		fact.ProducerDigest(),
		fact.EvidenceDigest(),
	)
}

func joinStringSets(
	class string,
	actual, expected map[string]bool,
	problems *problemSet,
) {
	for value := range actual {
		if !expected[value] {
			problems.add("unexpected " + class + " " + value)
		}
	}
	for value := range expected {
		if !actual[value] {
			problems.add("missing " + class + " " + value)
		}
	}
}
