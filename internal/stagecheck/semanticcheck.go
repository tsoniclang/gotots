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
	id               identity.PackageID
	pkg              structure.PackageGraph
	loaded           *source.LoadedPackage
	definitions      map[identity.DefinitionID]structure.ImplementationDefinition
	selections       map[identity.DefinitionID]scope.DefinitionSelection
	executable       map[identity.DefinitionID]bool
	regions          map[identity.DefinitionID]executable.Region
	parents          map[identity.DefinitionID]identity.DefinitionID
	initializers     map[identity.DefinitionID][]identity.OccurrenceID
	occurrences      map[identity.OccurrenceID]structure.Occurrence
	order            []identity.OccurrenceID
	domains          map[identity.OccurrenceID]catalog.ResolutionDomain
	owners           map[identity.OccurrenceID]identity.DefinitionID
	structuralOwners map[identity.OccurrenceID]identity.DefinitionID
	localFiles       map[identity.FileID]bool
}

func VerifyStage2(
	universe *source.Universe,
	plan *sourceplan.Plan,
	graph *structure.Graph,
	index *structure.TransientIndex,
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
		index == nil ||
		facts == nil ||
		selections == nil ||
		executableInventory == nil ||
		model == nil {
		return semanticVerificationError(
			"model", "Stage-2 inputs are invalid",
		)
	}
	loaded := loadedSemanticPackages(universe)
	selectionIndex := semanticSelections(selections)
	additional := semanticAdditionalOccurrences(executableInventory)
	expected, err := semanticDefinitionCensus(graph, loaded)
	if err != nil {
		return err
	}
	if model.PackageCount() != len(expected) {
		return semanticVerificationError(
			"model",
			fmt.Sprintf(
				"model has %d packages, expected %d",
				model.PackageCount(), len(expected),
			),
		)
	}
	authority, mixedPackages, err :=
		verifySemanticAuthorityProjections(
			model, expected, loaded, plan,
		)
	if err != nil {
		return err
	}
	semanticBefore := model.ProviderReadStats()
	if semanticBefore.ShardLoads != 0 ||
		semanticBefore.MaxProviderPackagesResident != 0 {
		return semanticVerificationError(
			"provider-residency",
			"semantic provider detail was opened before Stage-2 verification",
		)
	}
	packageIDs := sortedSemanticPackages(expected)
	owners, err := censusSemanticOwners(
		model, provider, packageIDs,
	)
	if err != nil {
		return err
	}
	expectations := map[identity.PackageID]semanticPackageExpectation{}
	before := graph.ProviderProjectionStats().ShardLoads
	err = graph.VisitResidentPackages(func(
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
	for packageID := range expected {
		if _, present := expectations[packageID]; present {
			continue
		}
		loadedPackage := loaded[packageID]
		if loadedPackage == nil ||
			loadedPackage.Disposition() !=
				source.DispositionBuiltinUniverse {
			return semanticVerificationError(
				"package",
				"semantic package has no structural or intrinsic owner "+
					packageID.String(),
			)
		}
		expectations[packageID] = builtinSemanticExpectation(
			loadedPackage,
		)
	}
	visited := 0
	expectedVisits := 0
	for _, packageID := range packageIDs {
		expectation, present := expectations[packageID]
		if !present {
			return semanticVerificationError(
				"package",
				"resident package projection is absent for "+packageID.String(),
			)
		}
		projection := authority[packageID]
		if !projection.HasLocalAuthority() {
			continue
		}
		expectedVisits++
		err := model.VisitPackage(
			packageID,
			func(actual semantic.Package) error {
				visited++
				if err := verifySemanticPackageClosure(
					actual, owners,
				); err != nil {
					return semanticVerificationError(
						"closure", err.Error(),
					)
				}
				return verifySemanticPackage(
					expectation,
					expected[packageID],
					actual,
					universe,
					plan,
					facts,
					provider,
					index,
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
	if visited != expectedVisits {
		return semanticVerificationError(
			"model",
			fmt.Sprintf(
				"visited %d packages, expected %d",
				visited, expectedVisits,
			),
		)
	}
	stats := model.ProviderReadStats()
	if stats.ShardLoads != mixedPackages {
		return semanticVerificationError(
			"provider-residency",
			fmt.Sprintf(
				"opened %d semantic provider shards, expected %d mixed projections",
				stats.ShardLoads, mixedPackages,
			),
		)
	}
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
	index *structure.TransientIndex,
	facts *selectionfacts.Artifact,
	selections *scope.DefinitionSelections,
	executableInventory *executable.Inventory,
	provider *semantic.ProviderArtifact,
) error {
	if universe == nil || !universe.Hydrated() ||
		plan == nil ||
		plan.Purpose() != sourceplan.PurposeProviderProduction ||
		graph == nil ||
		index == nil ||
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
						index,
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
		id: pkg.ID(), pkg: pkg, loaded: loaded,
		definitions:      map[identity.DefinitionID]structure.ImplementationDefinition{},
		selections:       map[identity.DefinitionID]scope.DefinitionSelection{},
		executable:       map[identity.DefinitionID]bool{},
		regions:          map[identity.DefinitionID]executable.Region{},
		parents:          map[identity.DefinitionID]identity.DefinitionID{},
		initializers:     map[identity.DefinitionID][]identity.OccurrenceID{},
		occurrences:      map[identity.OccurrenceID]structure.Occurrence{},
		domains:          map[identity.OccurrenceID]catalog.ResolutionDomain{},
		owners:           map[identity.OccurrenceID]identity.DefinitionID{},
		structuralOwners: map[identity.OccurrenceID]identity.DefinitionID{},
		localFiles:       map[identity.FileID]bool{},
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
	for _, site := range pkg.Sites() {
		if existing, present := out.parents[site.Definition()]; present && existing != site.ParentDefinition() {
			return semanticPackageExpectation{},
				semanticVerificationError(
					"definition",
					"definition site has two parents "+
						site.Definition().String(),
				)
		}
		out.parents[site.Definition()] = site.ParentDefinition()
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
		if err := out.assignStructuralOwner(
			header.Members(), header.ID().Definition(),
		); err != nil {
			return semanticPackageExpectation{}, err
		}
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
		var members []identity.OccurrenceID
		for _, entry := range boundary.Entries() {
			members = append(members, entry.ID())
		}
		_, local := out.definitions[boundary.ID().Definition()]
		if localOnly && !local {
			continue
		}
		out.initializers[boundary.ID().Definition()] = append(
			[]identity.OccurrenceID(nil), members...,
		)
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
		out.regions[definition] = region
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

func builtinSemanticExpectation(
	loaded *source.LoadedPackage,
) semanticPackageExpectation {
	return semanticPackageExpectation{
		id: loaded.ID(), loaded: loaded,
		definitions:      map[identity.DefinitionID]structure.ImplementationDefinition{},
		selections:       map[identity.DefinitionID]scope.DefinitionSelection{},
		executable:       map[identity.DefinitionID]bool{},
		regions:          map[identity.DefinitionID]executable.Region{},
		parents:          map[identity.DefinitionID]identity.DefinitionID{},
		initializers:     map[identity.DefinitionID][]identity.OccurrenceID{},
		occurrences:      map[identity.OccurrenceID]structure.Occurrence{},
		domains:          map[identity.OccurrenceID]catalog.ResolutionDomain{},
		owners:           map[identity.OccurrenceID]identity.DefinitionID{},
		structuralOwners: map[identity.OccurrenceID]identity.DefinitionID{},
		localFiles:       map[identity.FileID]bool{},
	}
}

func (expected *semanticPackageExpectation) assignStructuralOwner(
	occurrences []identity.OccurrenceID,
	owner identity.DefinitionID,
) error {
	for _, occurrence := range occurrences {
		if existing := expected.structuralOwners[occurrence]; !existing.IsZero() && existing != owner {
			return semanticVerificationError(
				"occurrence",
				"structural occurrence has two definition owners "+
					occurrence.String(),
			)
		}
		expected.structuralOwners[occurrence] = owner
	}
	return nil
}

func (expected *semanticPackageExpectation) addOccurrence(
	occurrence structure.Occurrence,
) error {
	if existing, present := expected.occurrences[occurrence.ID()]; present {
		if existing != occurrence {
			return semanticVerificationError(
				"occurrence", "conflicting payload "+occurrence.ID().String(),
			)
		}
		return nil
	}
	expected.occurrences[occurrence.ID()] = occurrence
	expected.order = append(expected.order, occurrence.ID())
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
