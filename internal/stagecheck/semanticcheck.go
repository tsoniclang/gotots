package stagecheck

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/executable"
	"github.com/tsoniclang/gotots/internal/language/selectionfacts"
	"github.com/tsoniclang/gotots/internal/language/semantic"
	"github.com/tsoniclang/gotots/internal/language/structure"
	"github.com/tsoniclang/gotots/internal/scope"
	"github.com/tsoniclang/gotots/internal/scope/sourceplan"
	"github.com/tsoniclang/gotots/internal/source"
)

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
	checkerBefore := model.CheckerReadStats()
	projectionBefore := model.ProjectionReadStats()
	if checkerBefore.ShardLoads != 0 ||
		checkerBefore.MaxPackagesResident != 0 ||
		projectionBefore.PackageLoads != 0 ||
		projectionBefore.MixedPackageLoads != 0 ||
		projectionBefore.MaxPackagesResident != 0 {
		return semanticVerificationError(
			"checker-residency",
			"semantic checker detail was opened before Stage-2 verification",
		)
	}
	packageIDs := sortedSemanticPackages(expected)
	owners, err := censusSemanticOwners(
		model, packageIDs,
	)
	if err != nil {
		return err
	}
	before := graph.ProviderProjectionStats().ShardLoads
	seenStructural := map[identity.PackageID]bool{}
	visited := 0
	expectedVisits := 0
	err = graph.VisitResidentPackages(func(
		pkg structure.PackageGraph,
	) error {
		packageID := pkg.ID()
		if seenStructural[packageID] {
			return semanticVerificationError(
				"package",
				"resident structural package repeats "+packageID.String(),
			)
		}
		seenStructural[packageID] = true
		definitions, present := expected[packageID]
		if !present {
			return semanticVerificationError(
				"package",
				"resident structural package is unexpected "+packageID.String(),
			)
		}
		projection, present := authority[packageID]
		if !present {
			return semanticVerificationError(
				"authority",
				"resident structural package has no authority "+packageID.String(),
			)
		}
		if !projection.HasLocalAuthority() {
			return nil
		}
		expectation, err := newSemanticPackageExpectation(
			pkg,
			loaded[packageID],
			selectionIndex,
			executableInventory,
			plan,
			true,
		)
		if err != nil {
			return err
		}
		expectedVisits++
		return model.VisitPackage(
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
					definitions,
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
	})
	if err != nil {
		return err
	}
	for _, packageID := range packageIDs {
		if seenStructural[packageID] {
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
		projection, present := authority[packageID]
		if !present {
			return semanticVerificationError(
				"authority",
				"builtin semantic package has no authority "+packageID.String(),
			)
		}
		if !projection.HasLocalAuthority() {
			continue
		}
		expectation := builtinSemanticExpectation(loadedPackage)
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
	checkerStats := model.CheckerReadStats()
	if checkerStats.ShardLoads != expectedVisits {
		return semanticVerificationError(
			"checker-residency",
			fmt.Sprintf(
				"opened %d semantic checker shards, expected %d local projections",
				checkerStats.ShardLoads, expectedVisits,
			),
		)
	}
	if checkerStats.MaxPackagesResident > 1 {
		return semanticVerificationError(
			"checker-residency",
			fmt.Sprintf(
				"%d semantic checker packages were resident",
				checkerStats.MaxPackagesResident,
			),
		)
	}
	projectionStats := model.ProjectionReadStats()
	if projectionStats.PackageLoads != expectedVisits ||
		projectionStats.MixedPackageLoads != mixedPackages ||
		projectionStats.MaxPackagesResident > 1 {
		return semanticVerificationError(
			"semantic-residency",
			fmt.Sprintf(
				"logical projections loads=%d mixed=%d resident=%d, expected %d/%d/1",
				projectionStats.PackageLoads,
				projectionStats.MixedPackageLoads,
				projectionStats.MaxPackagesResident,
				expectedVisits,
				mixedPackages,
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
