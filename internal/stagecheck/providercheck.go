package stagecheck

import "github.com/tsoniclang/gotots/internal/language/structure"

func syntheticLedger(pkg structure.PackageGraph) *structuralLedger {
	ledger := newStructuralLedger()
	for _, owner := range pkg.SyntheticOwners() {
		if owner.ID().SyntheticKind() ==
			structure.SyntheticOwnerCgoGenerated {
			addRecord(&ledger.owners, owner.ID())
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

type joinedIdentity[Self any] interface {
	comparable
	String() string
}

func joinIdentitySets[ID joinedIdentity[ID]](
	class string,
	actual, expected map[ID]bool,
	problems *problemSet,
) {
	for value := range actual {
		if !expected[value] {
			problems.add(
				"unexpected " + class + " " + value.String(),
			)
		}
	}
	for value := range expected {
		if !actual[value] {
			problems.add("missing " + class + " " + value.String())
		}
	}
}
