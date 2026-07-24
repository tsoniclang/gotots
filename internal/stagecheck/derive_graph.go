package stagecheck

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/structure"
	"github.com/tsoniclang/gotots/internal/scope/sourceplan"
	"github.com/tsoniclang/gotots/internal/source"
)

func deriveExpectedGraph(
	universe *source.Universe,
	plan *sourceplan.Plan,
	artifact *structure.ProviderArtifact,
	selectedPackages map[identity.PackageID]bool,
) (*structuralLedger, error) {
	return deriveExpectedGraphAuthority(
		universe, plan, artifact, selectedPackages, true,
	)
}

func deriveExpectedResidentGraph(
	universe *source.Universe,
	plan *sourceplan.Plan,
	selectedPackages map[identity.PackageID]bool,
) (*structuralLedger, error) {
	return deriveExpectedGraphAuthority(
		universe, plan, nil, selectedPackages, false,
	)
}

func deriveExpectedGraphAuthority(
	universe *source.Universe,
	plan *sourceplan.Plan,
	artifact *structure.ProviderArtifact,
	selectedPackages map[identity.PackageID]bool,
	includeCertified bool,
) (*structuralLedger, error) {
	expected := newStructuralLedger()
	loadedByPackage := map[identity.PackageID]*source.LoadedPackage{}
	derivedByPackage := map[identity.PackageID]map[identity.FileID]*derivedFile{}
	for _, pkg := range universe.Packages() {
		loadedByPackage[pkg.ID()] = pkg
		if selectedPackages != nil &&
			!selectedPackages[pkg.ID()] {
			continue
		}
		if pkg.Disposition() != source.DispositionOrdinarySource &&
			pkg.Disposition() != source.DispositionUnsafeIntrinsic {
			continue
		}
		derivedByPackage[pkg.ID()] = map[identity.FileID]*derivedFile{}
		for _, file := range pkg.Files() {
			decisionKind := sourceplan.KindLocalSyntax
			if plan != nil {
				decision, present := plan.For(file.ID())
				if !present {
					return nil, fmt.Errorf(
						"independent source plan omits %s", file.ID(),
					)
				}
				decisionKind = decision.Kind()
			}
			switch decisionKind {
			case sourceplan.KindLocalSyntax:
				derived, err := deriveFile(file)
				if err != nil {
					return nil, err
				}
				derivedByPackage[pkg.ID()][file.ID()] = derived
				expected.merge(derived.ledger)
			case sourceplan.KindCertifiedGraph:
				if !includeCertified {
					continue
				}
				if artifact == nil {
					return nil, fmt.Errorf(
						"independent certified file %s lacks artifact",
						file.ID(),
					)
				}
				certified, digest, found, err := artifact.FileGraph(file.ID())
				if err != nil {
					return nil, err
				}
				if !found ||
					digest != file.ByteDigest().String() {
					return nil, fmt.Errorf(
						"independent certified file %s is missing or stale",
						file.ID(),
					)
				}
				expected.merge(ledgerForFile(certified))
			default:
				return nil, fmt.Errorf(
					"independent file %s has invalid source decision",
					file.ID(),
				)
			}
		}
	}
	for pkgID, files := range derivedByPackage {
		pkg := loadedByPackage[pkgID]
		var synthetic sourceplan.SyntheticOwner
		planned := false
		if plan != nil {
			synthetic, planned = plan.SyntheticFor(pkgID)
		}
		localSynthetic := !planned ||
			synthetic.Kind() == sourceplan.KindLocalSyntax
		if planned &&
			synthetic.Kind() == sourceplan.KindCertifiedGraph &&
			includeCertified {
			if artifact == nil {
				return nil, fmt.Errorf(
					"independent synthetic owner %s lacks artifact", pkgID,
				)
			}
			certified, present, err := artifact.SyntheticPackageGraph(pkgID)
			if err != nil {
				return nil, err
			}
			if !present {
				return nil, fmt.Errorf(
					"independent synthetic owner %s is absent", pkgID,
				)
			}
			addDefinitionRecords(
				expected,
				certified.Definitions(),
				certified.Sites(),
				certified.Headers(),
				certified.Boundaries(),
				true,
			)
			for _, owner := range certified.SyntheticOwners() {
				addRecord(&expected.owners, owner.ID())
			}
		}
		if err := deriveCgoPackage(
			universe, pkg, files, localSynthetic, expected,
		); err != nil {
			return nil, err
		}
		if pkg.Disposition() == source.DispositionOrdinarySource {
			if err := addExpectedPackageInitialization(expected, pkgID); err != nil {
				return nil, err
			}
		}
	}
	return expected, nil
}

func addExpectedPackageInitialization(
	ledger *structuralLedger,
	pkg identity.PackageID,
) error {
	ownerID, err := structure.SyntheticOwner(
		pkg, structure.SyntheticOwnerPackageInitialization,
	)
	if err != nil {
		return err
	}
	definition, err := identity.NewImplicitDefinitionID(
		pkg, identity.ImplicitDefinitionPackageInit,
	)
	if err != nil {
		return err
	}
	header, _ := identity.NewHeaderRegionID(definition)
	boundary, _ := identity.NewExecutionBoundaryID(definition)
	addRecord(&ledger.owners, ownerID)
	addRecord(&ledger.definitions, definitionLedgerRecord{
		id:       definition,
		owner:    ownerID,
		header:   header,
		boundary: boundary,
		name:     "package initialization",
	})
	addRecord(&ledger.definitionSites, definitionSiteLedgerRecord{
		kind:       structure.DefinitionSiteSynthetic,
		definition: definition,
		owner:      ownerID,
	})
	addRecord(&ledger.headers, headerLedgerRecord{
		id:     header,
		digest: independentDigest(definition.String(), "header"),
	})
	addRecord(
		&ledger.executionBoundaries,
		executionBoundaryLedgerRecord{
			id:   boundary,
			kind: structure.BoundaryImplicit,
			digest: independentDigest(
				definition.String(), "execution",
			),
			implicit: identity.ImplicitDefinitionPackageInit,
		},
	)
	return nil
}
