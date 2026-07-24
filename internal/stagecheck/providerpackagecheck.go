package stagecheck

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/selectionfacts"
	"github.com/tsoniclang/gotots/internal/language/structure"
	"github.com/tsoniclang/gotots/internal/scope/contract"
	"github.com/tsoniclang/gotots/internal/scope/sourceplan"
	"github.com/tsoniclang/gotots/internal/source"
)

// VerifyProviderManifest exact-joins the disk-backed manifest to the complete
// provider-production plan without loading any package shard.
func VerifyProviderManifest(
	universe *source.Universe,
	selected contract.Contract,
	plan *sourceplan.Plan,
	artifact *structure.ProviderArtifact,
) error {
	if err := structure.VerifyProviderArtifactContext(
		artifact, universe, selected,
	); err != nil {
		return providerManifestError(err.Error())
	}
	files, synthetic, contexts, packages, err :=
		expectedProviderSets(universe, plan, identity.PackageID{})
	if err != nil {
		return err
	}
	problems := newProblemSet()
	joinIdentitySets(
		"provider file", artifact.FileIDs(), files, problems,
	)
	joinIdentitySets(
		"provider synthetic package",
		artifact.PackageIDs(),
		synthetic,
		problems,
	)
	joinIdentitySets(
		"provider package context",
		artifact.ContextPackageIDs(),
		contexts,
		problems,
	)
	for packageID := range contexts {
		if _, present := artifact.PackageCensus(packageID); !present {
			problems.add(
				"provider package census missing " + packageID.String(),
			)
		}
		actual, present := artifact.PackageInputDigest(packageID)
		if !present {
			continue
		}
		if actual != independentProviderInputFingerprint(
			packages[packageID],
		) {
			problems.add(
				"provider package input mismatch " + packageID.String(),
			)
		}
	}
	if !problems.empty() {
		return providerManifestError(
			problems.summary("manifest exact join failed"),
		)
	}
	return nil
}

// VerifyProducedProviderPackageArtifact exact-joins one package shard to its
// independently verified local Stage-1 graph and selection facts.
func VerifyProducedProviderPackageArtifact(
	universe *source.Universe,
	selected contract.Contract,
	plan *sourceplan.Plan,
	packageID identity.PackageID,
	graph *structure.Graph,
	facts *selectionfacts.Artifact,
	artifact *structure.ProviderArtifact,
) error {
	if err := structure.VerifyProviderArtifactContext(
		artifact, universe, selected,
	); err != nil {
		return providerManifestError(err.Error())
	}
	files, synthetic, contexts, packages, err :=
		expectedProviderSets(universe, plan, packageID)
	if err != nil {
		return err
	}
	problems := newProblemSet()
	joinIdentitySets(
		"provider file",
		artifact.PackageFileIDs(packageID),
		files,
		problems,
	)
	actualSynthetic := map[identity.PackageID]bool{}
	if artifact.HasSyntheticPackage(packageID) {
		actualSynthetic[packageID] = true
	}
	joinIdentitySets(
		"provider synthetic package",
		actualSynthetic,
		synthetic,
		problems,
	)
	actualContext := map[identity.PackageID]bool{}
	if _, present := artifact.PackageInputDigest(packageID); present {
		actualContext[packageID] = true
	}
	joinIdentitySets(
		"provider package context",
		actualContext,
		contexts,
		problems,
	)
	actualDigest, present := artifact.PackageInputDigest(packageID)
	if !present ||
		actualDigest != independentProviderInputFingerprint(
			packages[packageID],
		) {
		problems.add(
			"provider package input mismatch " + packageID.String(),
		)
	}
	var graphPackage structure.PackageGraph
	packageCount := 0
	if visitErr := graph.VisitPackages(
		func(pkg structure.PackageGraph) error {
			packageCount++
			graphPackage = pkg
			return nil
		},
	); visitErr != nil {
		return visitErr
	}
	if packageCount != 1 ||
		graphPackage.ID() != packageID {
		problems.add(
			"local graph is not exactly " + packageID.String(),
		)
	} else {
		census, present := artifact.PackageCensus(packageID)
		if !present {
			problems.add(
				"provider package census missing " +
					packageID.String(),
			)
		} else if err := compareProviderPackageCensus(
			graphPackage,
			files,
			synthetic[packageID],
			census,
		); err != nil {
			problems.add(err.Error())
		}
		localFiles := map[identity.FileID]structure.FileGraph{}
		for _, file := range graphPackage.Files() {
			localFiles[file.Owner().ID().File()] = file
		}
		for fileID := range files {
			stored, _, found, loadErr := artifact.FileGraph(fileID)
			if loadErr != nil {
				return loadErr
			}
			if !found {
				continue
			}
			if err := compareLedgers(
				"provider-package-file",
				ledgerForFile(stored),
				ledgerForFile(localFiles[fileID]),
			); err != nil {
				problems.add(err.Error())
			}
		}
		if synthetic[packageID] {
			stored, found, loadErr :=
				artifact.SyntheticPackageGraph(packageID)
			if loadErr != nil {
				return loadErr
			}
			if found {
				if err := compareLedgers(
					"provider-package-synthetic",
					syntheticLedger(stored),
					syntheticLedger(graphPackage),
				); err != nil {
					problems.add(err.Error())
				}
			}
		}
	}
	expectedFacts := recordMultiset[certifiedFactLedgerRecord]{}
	for _, fact := range facts.CertifiedFacts() {
		definition := fact.Definition()
		if files[definition.File()] ||
			(definition.SyntheticRole().Valid() &&
				synthetic[packageID]) {
			expectedFacts[certifiedFactLedgerRecordFromFact(fact)]++
		}
	}
	actualFacts := recordMultiset[certifiedFactLedgerRecord]{}
	storedFacts := artifact.CertifiedFactsForPackage(packageID)
	for _, fact := range storedFacts {
		actualFacts[certifiedFactLedgerRecordFromFact(fact)]++
	}
	joinProviderFactCounts(expectedFacts, actualFacts, problems)
	if !problems.empty() {
		return providerManifestError(
			problems.summary("package shard exact join failed"),
		)
	}
	return nil
}

func compareProviderPackageCensus(
	pkg structure.PackageGraph,
	files map[identity.FileID]bool,
	includeSynthetic bool,
	actual structure.ProviderPackageCensus,
) error {
	expectedDefinitions := newStructuralLedger()
	headerOccurrences := 0
	boundaryEntries := 0
	for _, file := range pkg.Files() {
		if !files[file.Owner().ID().File()] {
			continue
		}
		for _, definition := range file.Definitions() {
			addRecord(
				&expectedDefinitions.providerDefinitionCensus,
				definitionCensusLedgerRecord{
					pkg: pkg.ID(), definition: definition.ID(),
				},
			)
		}
		for _, header := range file.Headers() {
			headerOccurrences += len(header.Members())
		}
		for _, boundary := range file.Boundaries() {
			boundaryEntries += len(boundary.Entries())
		}
	}
	if includeSynthetic {
		for _, definition := range pkg.Definitions() {
			if definition.ID().SyntheticRole().Valid() {
				addRecord(
					&expectedDefinitions.providerDefinitionCensus,
					definitionCensusLedgerRecord{
						pkg: pkg.ID(), definition: definition.ID(),
					},
				)
			}
		}
		for _, header := range pkg.Headers() {
			if header.ID().Definition().SyntheticRole().Valid() {
				headerOccurrences += len(header.Members())
			}
		}
		for _, boundary := range pkg.Boundaries() {
			if boundary.ID().Definition().SyntheticRole().Valid() {
				boundaryEntries += len(boundary.Entries())
			}
		}
	}
	actualDefinitions := newStructuralLedger()
	for _, definition := range actual.Definitions() {
		addRecord(
			&actualDefinitions.providerDefinitionCensus,
			definitionCensusLedgerRecord{
				pkg: actual.Package(), definition: definition,
			},
		)
	}
	if err := compareLedgers(
		"provider-definition-census",
		actualDefinitions,
		expectedDefinitions,
	); err != nil {
		return err
	}
	if actual.Package() != pkg.ID() ||
		actual.HeaderOccurrenceCount() != headerOccurrences ||
		actual.BoundaryEntryCount() != boundaryEntries {
		return &VerificationError{
			Stage: "provider-definition-census",
			Reason: fmt.Sprintf(
				"package/count mismatch package=%s/%s headers=%d/%d boundaries=%d/%d",
				actual.Package(),
				pkg.ID(),
				actual.HeaderOccurrenceCount(),
				headerOccurrences,
				actual.BoundaryEntryCount(),
				boundaryEntries,
			),
		}
	}
	return nil
}

func expectedProviderSets(
	universe *source.Universe,
	plan *sourceplan.Plan,
	only identity.PackageID,
) (
	map[identity.FileID]bool,
	map[identity.PackageID]bool,
	map[identity.PackageID]bool,
	map[identity.PackageID]*source.LoadedPackage,
	error,
) {
	if universe == nil || plan == nil ||
		plan.Purpose() != sourceplan.PurposeProviderProduction {
		return nil, nil, nil, nil, providerManifestError(
			"provider-production plan is absent",
		)
	}
	packages := map[identity.PackageID]*source.LoadedPackage{}
	filePackages := map[identity.FileID]identity.PackageID{}
	for _, pkg := range universe.Packages() {
		packages[pkg.ID()] = pkg
		for _, file := range pkg.Files() {
			filePackages[file.ID()] = pkg.ID()
		}
	}
	files := map[identity.FileID]bool{}
	synthetic := map[identity.PackageID]bool{}
	contexts := map[identity.PackageID]bool{}
	for _, decision := range plan.Files() {
		if decision.Kind() != sourceplan.KindCertifiedGraph {
			continue
		}
		packageID := filePackages[decision.ID()]
		if packageID.IsZero() {
			return nil, nil, nil, nil, providerManifestError(
				"provider plan file has no package " +
					decision.ID().String(),
			)
		}
		if !only.IsZero() && packageID != only {
			continue
		}
		files[decision.ID()] = true
		contexts[packageID] = true
	}
	for _, decision := range plan.SyntheticOwners() {
		if decision.Kind() != sourceplan.KindCertifiedGraph ||
			(!only.IsZero() && decision.Package() != only) {
			continue
		}
		synthetic[decision.Package()] = true
		contexts[decision.Package()] = true
	}
	if !only.IsZero() && !contexts[only] {
		return nil, nil, nil, nil, providerManifestError(
			"package has no certified provider records " + only.String(),
		)
	}
	return files, synthetic, contexts, packages, nil
}

func joinProviderFactCounts(
	expected recordMultiset[certifiedFactLedgerRecord],
	actual recordMultiset[certifiedFactLedgerRecord],
	problems *problemSet,
) {
	for record, count := range expected {
		if actual[record] != count {
			problems.addf(
				"provider fact expected %s x%d, actual x%d",
				renderCertifiedFactLedgerRecord(record),
				count,
				actual[record],
			)
		}
	}
	for record, count := range actual {
		if expected[record] != count {
			problems.addf(
				"provider fact actual %s x%d, expected x%d",
				renderCertifiedFactLedgerRecord(record),
				count,
				expected[record],
			)
		}
	}
}

func providerManifestError(reason string) error {
	return &VerificationError{
		Stage: "provider-artifact", Reason: reason,
	}
}
