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

type independentPackageEvidence struct {
	ledger *structuralLedger
	files  map[identity.FileID]*derivedFile
}

func deriveExpectedResidentPackage(
	universe *source.Universe,
	plan *sourceplan.Plan,
	pkg *source.LoadedPackage,
) (*independentPackageEvidence, error) {
	return deriveExpectedPackageAuthority(
		universe, plan, nil, pkg, false,
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
	for _, pkg := range universe.Packages() {
		if selectedPackages != nil &&
			!selectedPackages[pkg.ID()] {
			continue
		}
		if pkg.Disposition() != source.DispositionOrdinarySource &&
			pkg.Disposition() != source.DispositionUnsafeIntrinsic {
			continue
		}
		evidence, err := deriveExpectedPackageAuthority(
			universe, plan, artifact, pkg, includeCertified,
		)
		if err != nil {
			return nil, err
		}
		expected.merge(evidence.ledger)
	}
	return expected, nil
}

func deriveExpectedPackageAuthority(
	universe *source.Universe,
	plan *sourceplan.Plan,
	artifact *structure.ProviderArtifact,
	pkg *source.LoadedPackage,
	includeCertified bool,
) (*independentPackageEvidence, error) {
	evidence := &independentPackageEvidence{
		ledger: newStructuralLedger(),
		files:  map[identity.FileID]*derivedFile{},
	}
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
			evidence.files[file.ID()] = derived
			evidence.ledger.merge(derived.ledger)
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
			if !found || digest != file.ByteDigest().String() {
				return nil, fmt.Errorf(
					"independent certified file %s is missing or stale",
					file.ID(),
				)
			}
			evidence.ledger.merge(ledgerForFile(certified))
		default:
			return nil, fmt.Errorf(
				"independent file %s has invalid source decision",
				file.ID(),
			)
		}
	}
	var synthetic sourceplan.SyntheticOwner
	planned := false
	if plan != nil {
		synthetic, planned = plan.SyntheticFor(pkg.ID())
	}
	localSynthetic := !planned ||
		synthetic.Kind() == sourceplan.KindLocalSyntax
	if planned &&
		synthetic.Kind() == sourceplan.KindCertifiedGraph &&
		includeCertified {
		if artifact == nil {
			return nil, fmt.Errorf(
				"independent synthetic owner %s lacks artifact", pkg.ID(),
			)
		}
		certified, present, err := artifact.SyntheticPackageGraph(pkg.ID())
		if err != nil {
			return nil, err
		}
		if !present {
			return nil, fmt.Errorf(
				"independent synthetic owner %s is absent", pkg.ID(),
			)
		}
		addDefinitionRecords(
			evidence.ledger,
			certified.Definitions(),
			certified.Sites(),
			certified.Headers(),
			certified.Boundaries(),
			true,
		)
		for _, owner := range certified.SyntheticOwners() {
			addRecord(&evidence.ledger.owners, owner.ID())
		}
	}
	if err := deriveCgoPackage(
		universe, pkg, evidence.files, localSynthetic, evidence.ledger,
	); err != nil {
		return nil, err
	}
	if pkg.Disposition() == source.DispositionOrdinarySource {
		if err := addExpectedPackageInitialization(
			evidence.ledger, pkg.ID(),
		); err != nil {
			return nil, err
		}
	}
	return evidence, nil
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
