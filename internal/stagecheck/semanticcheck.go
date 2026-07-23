package stagecheck

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
	"github.com/tsoniclang/gotots/internal/language/executable"
	"github.com/tsoniclang/gotots/internal/language/selectionfacts"
	"github.com/tsoniclang/gotots/internal/language/semantic"
	"github.com/tsoniclang/gotots/internal/language/structure"
	"github.com/tsoniclang/gotots/internal/scope"
	"github.com/tsoniclang/gotots/internal/scope/sourceplan"
	"github.com/tsoniclang/gotots/internal/source"
)

type semanticPackageExpectation struct {
	pkg         structure.PackageGraph
	loaded      *source.LoadedPackage
	definitions map[identity.DefinitionID]structure.ImplementationDefinition
	selections  map[identity.DefinitionID]scope.DefinitionSelection
	executable  map[identity.DefinitionID]bool
	occurrences map[identity.OccurrenceID]structure.Occurrence
	domains     map[identity.OccurrenceID]catalog.ResolutionDomain
	owners      map[identity.OccurrenceID]identity.DefinitionID
	localFiles  map[identity.FileID]bool
}

func VerifyStage2(
	universe *source.Universe,
	plan *sourceplan.Plan,
	graph *structure.Graph,
	facts *selectionfacts.Artifact,
	selections *scope.DefinitionSelections,
	executableInventory *executable.Inventory,
	model *semantic.Model,
	provider *semantic.ProviderArtifact,
) error {
	if universe == nil || !universe.Hydrated() ||
		plan == nil ||
		plan.Purpose() != sourceplan.PurposeCompilation ||
		graph == nil ||
		facts == nil ||
		selections == nil ||
		executableInventory == nil ||
		model == nil ||
		model.PackageCount() != graph.PackageCount() {
		return semanticVerificationError(
			"model", "Stage-2 inputs or package cardinality are invalid",
		)
	}
	loaded := loadedSemanticPackages(universe)
	selectionIndex := semanticSelections(selections)
	additional := semanticAdditionalOccurrences(executableInventory)
	expected := semanticDefinitionCensus(graph)
	expectations := map[identity.PackageID]semanticPackageExpectation{}
	before := graph.ProviderProjectionStats().ShardLoads
	err := graph.VisitResidentPackages(func(
		pkg structure.PackageGraph,
	) error {
		expectation, err := newSemanticPackageExpectation(
			pkg,
			loaded[pkg.ID()],
			selectionIndex,
			executableInventory,
			additional,
			plan,
			true,
		)
		if err != nil {
			return err
		}
		expectations[pkg.ID()] = expectation
		return nil
	})
	if err != nil {
		return err
	}
	visited := 0
	for _, packageID := range sortedSemanticPackages(expected) {
		expectation, present := expectations[packageID]
		if !present {
			return semanticVerificationError(
				"package",
				"resident package projection is absent for "+packageID.String(),
			)
		}
		err := model.VisitPackage(
			packageID,
			func(actual semantic.Package) error {
				visited++
				return verifySemanticPackage(
					expectation,
					expected[packageID],
					actual,
					universe,
					plan,
					facts,
					provider,
					true,
				)
			},
		)
		if err != nil {
			return err
		}
	}
	if after := graph.ProviderProjectionStats().ShardLoads; after != before {
		return semanticVerificationError(
			"provider-residency",
			fmt.Sprintf(
				"Stage 2 reopened %d structural provider shards",
				after-before,
			),
		)
	}
	if visited != graph.PackageCount() {
		return semanticVerificationError(
			"model",
			fmt.Sprintf(
				"visited %d packages, expected %d",
				visited, graph.PackageCount(),
			),
		)
	}
	stats := model.ProviderReadStats()
	if stats.MaxProviderPackagesResident > 1 {
		return semanticVerificationError(
			"provider-residency",
			fmt.Sprintf(
				"%d semantic provider packages were resident",
				stats.MaxProviderPackagesResident,
			),
		)
	}
	return nil
}

func VerifyProducedSemanticPackage(
	universe *source.Universe,
	plan *sourceplan.Plan,
	packageID identity.PackageID,
	graph *structure.Graph,
	_ *structure.TransientIndex,
	facts *selectionfacts.Artifact,
	selections *scope.DefinitionSelections,
	executableInventory *executable.Inventory,
	provider *semantic.ProviderArtifact,
) error {
	if universe == nil || !universe.Hydrated() ||
		plan == nil ||
		plan.Purpose() != sourceplan.PurposeProviderProduction ||
		graph == nil ||
		facts == nil ||
		selections == nil ||
		executableInventory == nil ||
		provider == nil {
		return semanticVerificationError(
			"provider", "provider semantic verification inputs are invalid",
		)
	}
	loaded := loadedSemanticPackages(universe)
	selectionIndex := semanticSelections(selections)
	additional := semanticAdditionalOccurrences(executableInventory)
	found := false
	err := graph.VisitResidentPackages(
		func(pkg structure.PackageGraph) error {
			if pkg.ID() != packageID {
				return nil
			}
			found = true
			expectation, err := newSemanticPackageExpectation(
				pkg,
				loaded[pkg.ID()],
				selectionIndex,
				executableInventory,
				additional,
				plan,
				false,
			)
			if err != nil {
				return err
			}
			return provider.VisitPackage(
				packageID,
				func(actual semantic.Package) error {
					return verifySemanticPackage(
						expectation,
						semanticDefinitionSet(expectation.definitions),
						actual,
						universe,
						plan,
						facts,
						provider,
						false,
					)
				},
			)
		},
	)
	if err != nil {
		return err
	}
	if !found {
		return semanticVerificationError(
			"provider", "derived semantic package is absent",
		)
	}
	return nil
}

func newSemanticPackageExpectation(
	pkg structure.PackageGraph,
	loaded *source.LoadedPackage,
	selections map[identity.DefinitionID]scope.DefinitionSelection,
	executableInventory *executable.Inventory,
	additional map[identity.OccurrenceID]structure.Occurrence,
	plan *sourceplan.Plan,
	localOnly bool,
) (semanticPackageExpectation, error) {
	if loaded == nil {
		return semanticPackageExpectation{}, semanticVerificationError(
			"package", "structural package is absent from source universe",
		)
	}
	out := semanticPackageExpectation{
		pkg: pkg, loaded: loaded,
		definitions: map[identity.DefinitionID]structure.ImplementationDefinition{},
		selections:  map[identity.DefinitionID]scope.DefinitionSelection{},
		executable:  map[identity.DefinitionID]bool{},
		occurrences: map[identity.OccurrenceID]structure.Occurrence{},
		domains:     map[identity.OccurrenceID]catalog.ResolutionDomain{},
		owners:      map[identity.OccurrenceID]identity.DefinitionID{},
		localFiles:  map[identity.FileID]bool{},
	}
	for _, file := range pkg.Files() {
		out.localFiles[file.Owner().ID().File()] = true
		for _, occurrence := range file.Occurrences() {
			if err := out.addOccurrence(occurrence); err != nil {
				return semanticPackageExpectation{}, err
			}
		}
		if err := out.assign(
			file.Owner().Members(),
			catalog.ResolutionDomainOwner,
			identity.DefinitionID{},
		); err != nil {
			return semanticPackageExpectation{}, err
		}
	}
	for _, owner := range pkg.SyntheticOwners() {
		if err := out.assign(
			owner.Members(),
			catalog.ResolutionDomainOwner,
			identity.DefinitionID{},
		); err != nil {
			return semanticPackageExpectation{}, err
		}
	}
	for _, definition := range pkg.Definitions() {
		if localOnly &&
			!semanticDefinitionUsesLocal(
				plan, loaded, definition.ID(),
			) {
			continue
		}
		if _, duplicate := out.definitions[definition.ID()]; duplicate {
			return semanticPackageExpectation{}, semanticVerificationError(
				"definition", "duplicate structural definition "+definition.ID().String(),
			)
		}
		selection, present := selections[definition.ID()]
		if !present {
			return semanticPackageExpectation{}, semanticVerificationError(
				"definition", "missing selection "+definition.ID().String(),
			)
		}
		out.definitions[definition.ID()] = definition
		out.selections[definition.ID()] = selection
	}
	for _, header := range pkg.Headers() {
		_, local := out.definitions[header.ID().Definition()]
		if localOnly && !local {
			continue
		}
		if err := out.assign(
			header.Members(),
			catalog.ResolutionDomainHeader,
			header.ID().Definition(),
		); err != nil {
			return semanticPackageExpectation{}, err
		}
	}
	for _, boundary := range pkg.Boundaries() {
		_, local := out.definitions[boundary.ID().Definition()]
		if localOnly && !local {
			continue
		}
		var members []identity.OccurrenceID
		for _, entry := range boundary.Entries() {
			members = append(members, entry.ID())
		}
		if err := out.assign(
			members,
			catalog.ResolutionDomainBoundary,
			boundary.ID().Definition(),
		); err != nil {
			return semanticPackageExpectation{}, err
		}
	}
	for definition := range out.definitions {
		region, present := executableInventory.For(definition)
		if !present {
			continue
		}
		out.executable[definition] = true
		for _, member := range region.Members() {
			if occurrence, present := additional[member]; present {
				if err := out.addOccurrence(occurrence); err != nil {
					return semanticPackageExpectation{}, err
				}
			}
		}
		if err := out.assign(
			region.Members(),
			catalog.ResolutionDomainExecutable,
			definition,
		); err != nil {
			return semanticPackageExpectation{}, err
		}
	}
	for occurrence := range out.domains {
		if _, present := out.occurrences[occurrence]; !present {
			return semanticPackageExpectation{}, semanticVerificationError(
				"occurrence", "region member lacks payload "+occurrence.String(),
			)
		}
	}
	return out, nil
}

func (expected *semanticPackageExpectation) addOccurrence(
	occurrence structure.Occurrence,
) error {
	if existing, present := expected.occurrences[occurrence.ID()]; present &&
		existing != occurrence {
		return semanticVerificationError(
			"occurrence", "conflicting payload "+occurrence.ID().String(),
		)
	}
	expected.occurrences[occurrence.ID()] = occurrence
	return nil
}

func (expected *semanticPackageExpectation) assign(
	occurrences []identity.OccurrenceID,
	domain catalog.ResolutionDomain,
	owner identity.DefinitionID,
) error {
	for _, occurrence := range occurrences {
		existing := expected.domains[occurrence]
		if existing >= domain {
			if existing == domain &&
				expected.owners[occurrence] != owner {
				return semanticVerificationError(
					"domain", "occurrence has two owners "+occurrence.String(),
				)
			}
			continue
		}
		expected.domains[occurrence] = domain
		expected.owners[occurrence] = owner
	}
	return nil
}
