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
			synthetic.Kind() == sourceplan.KindCertifiedGraph {
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
				expected.add("owner", owner.ID().String())
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
	owner := ownerID.String()
	definition, err := identity.NewImplicitDefinitionID(
		pkg, identity.ImplicitDefinitionPackageInit,
	)
	if err != nil {
		return err
	}
	header, _ := identity.NewHeaderRegionID(definition)
	boundary, _ := identity.NewExecutionBoundaryID(definition)
	ledger.add("owner", owner)
	ledger.add(
		"definition",
		fmt.Sprintf(
			"%s|%s|%s|%s|package initialization",
			definition, owner, header, boundary,
		),
	)
	ledger.add(
		"definition-site",
		fmt.Sprintf(
			"%d|%s|%s||",
			uint8(structure.DefinitionSiteSynthetic),
			definition,
			owner,
		),
	)
	ledger.add(
		"header",
		fmt.Sprintf(
			"%s|%s",
			header,
			independentDigest(definition.String(), "header"),
		),
	)
	ledger.add(
		"execution-boundary",
		fmt.Sprintf(
			"%s|%d|%s|%d|0",
			boundary,
			uint8(structure.BoundaryImplicit),
			independentDigest(definition.String(), "execution"),
			uint8(identity.ImplicitDefinitionPackageInit),
		),
	)
	return nil
}
