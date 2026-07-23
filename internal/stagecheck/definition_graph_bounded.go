package stagecheck

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/structure"
	"github.com/tsoniclang/gotots/internal/scope/sourceplan"
	"github.com/tsoniclang/gotots/internal/source"
)

func verifyDefinitionGraphPackagesBounded(
	universe *source.Universe,
	plan *sourceplan.Plan,
	graph *structure.Graph,
	certified *structure.ProviderArtifact,
	selectedPackages map[identity.PackageID]bool,
) error {
	authority, err := verifyStructuralAuthorityProjections(
		universe, plan, graph, selectedPackages,
	)
	if err != nil {
		return err
	}
	if stats := graph.ProviderProjectionStats(); stats.ShardLoads != 0 {
		return &VerificationError{
			Stage: "provider-residency",
			Reason: fmt.Sprintf(
				"%d structural provider shards opened before verification",
				stats.ShardLoads,
			),
		}
	}
	census := map[identity.PackageID][]structure.DefinitionCensusRecord{}
	for _, record := range graph.DefinitionCensus() {
		census[record.Package()] = append(
			census[record.Package()], record,
		)
	}
	visited := map[identity.PackageID]bool{}
	err = graph.VisitResidentPackages(func(
		pkg structure.PackageGraph,
	) error {
		projection, present := authority[pkg.ID()]
		if !present || visited[pkg.ID()] {
			return &VerificationError{
				Stage: "definition-graph",
				Reason: "resident package is unexpected or repeated " +
					pkg.ID().String(),
			}
		}
		visited[pkg.ID()] = true
		localCensus, certifiedCensus, err :=
			partitionStructuralDefinitionCensus(
				plan, pkg.ID(), census[pkg.ID()],
			)
		if err != nil {
			return err
		}
		if err := compareDefinitionCensus(
			pkg, localCensus,
		); err != nil {
			return err
		}
		if err := compareCertifiedDefinitionCensus(
			pkg.ID(),
			projection,
			certifiedCensus,
			certified,
		); err != nil {
			return err
		}
		expected, err := deriveExpectedResidentGraph(
			universe,
			plan,
			map[identity.PackageID]bool{pkg.ID(): true},
		)
		if err != nil {
			return &VerificationError{
				Stage:  "definition-graph-independent",
				Reason: pkg.ID().String() + ": " + err.Error(),
			}
		}
		return compareLedgers(
			"definition-graph/"+pkg.ID().String(),
			ledgerForPackage(pkg),
			expected,
		)
	})
	if err != nil {
		return err
	}
	for packageID := range authority {
		if !visited[packageID] {
			return &VerificationError{
				Stage: "definition-graph",
				Reason: "resident package is absent " +
					packageID.String(),
			}
		}
		delete(census, packageID)
	}
	for packageID := range census {
		return &VerificationError{
			Stage: "definition-graph",
			Reason: "definition census has unexpected package " +
				packageID.String(),
		}
	}
	if stats := graph.ProviderProjectionStats(); stats.ShardLoads != 0 {
		return &VerificationError{
			Stage: "provider-residency",
			Reason: fmt.Sprintf(
				"Stage-1 verification opened %d structural provider shards",
				stats.ShardLoads,
			),
		}
	}
	return nil
}

func partitionStructuralDefinitionCensus(
	plan *sourceplan.Plan,
	packageID identity.PackageID,
	records []structure.DefinitionCensusRecord,
) (
	[]structure.DefinitionCensusRecord,
	[]structure.DefinitionCensusRecord,
	error,
) {
	var local []structure.DefinitionCensusRecord
	var certified []structure.DefinitionCensusRecord
	for _, record := range records {
		selected, err := definitionUsesCertifiedStructure(
			plan, packageID, record.ID(),
		)
		if err != nil {
			return nil, nil, err
		}
		if selected {
			certified = append(certified, record)
		} else {
			local = append(local, record)
		}
	}
	return local, certified, nil
}

func definitionUsesCertifiedStructure(
	plan *sourceplan.Plan,
	packageID identity.PackageID,
	definition identity.DefinitionID,
) (bool, error) {
	if plan == nil || definition.ImplicitOp().Valid() {
		return false, nil
	}
	if file := definition.File(); !file.IsZero() {
		decision, present := plan.For(file)
		if !present {
			return false, &VerificationError{
				Stage: "definition-graph",
				Reason: "source plan omits definition " +
					definition.String(),
			}
		}
		return decision.Kind() == sourceplan.KindCertifiedGraph, nil
	}
	if definition.SyntheticRole().Valid() {
		decision, present := plan.SyntheticFor(packageID)
		if !present {
			return false, &VerificationError{
				Stage: "definition-graph",
				Reason: "source plan omits synthetic definition " +
					definition.String(),
			}
		}
		return decision.Kind() == sourceplan.KindCertifiedGraph, nil
	}
	return false, nil
}

func compareCertifiedDefinitionCensus(
	packageID identity.PackageID,
	projection structure.AuthorityProjection,
	records []structure.DefinitionCensusRecord,
	provider *structure.ProviderArtifact,
) error {
	expected := newStructuralLedger()
	if len(projection.CertifiedFiles()) != 0 ||
		projection.HasCertifiedSynthetic() {
		if provider == nil {
			return &VerificationError{
				Stage: "definition-census",
				Reason: "certified package has no provider " +
					packageID.String(),
			}
		}
		census, present := provider.PackageCensus(packageID)
		if !present {
			return &VerificationError{
				Stage:  "definition-census",
				Reason: "provider census omits " + packageID.String(),
			}
		}
		for _, definition := range census.Definitions() {
			expected.add(
				"definition-census",
				packageID.String()+"|"+definition.String(),
			)
		}
	}
	actual := newStructuralLedger()
	for _, record := range records {
		actual.add(
			"definition-census",
			record.Package().String()+"|"+record.ID().String(),
		)
	}
	return compareLedgers(
		"certified-definition-census/"+packageID.String(),
		actual,
		expected,
	)
}
