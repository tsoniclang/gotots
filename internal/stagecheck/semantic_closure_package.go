package stagecheck

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/semantic"
)

type semanticPackageClosure struct {
	packageID identity.PackageID
	owners    *semanticOwnerCensus
	pkg       semantic.Package
}

func verifySemanticPackageClosure(
	pkg semantic.Package,
	owners *semanticOwnerCensus,
) error {
	closure := semanticPackageClosure{
		packageID: pkg.ID(),
		owners:    owners,
		pkg:       pkg,
	}
	if err := closure.verifyMemberTargets(); err != nil {
		return err
	}
	if err := closure.verifyDefinitionOwners(); err != nil {
		return err
	}
	if err := closure.verifyDeclarationOwners(); err != nil {
		return err
	}
	return closure.verifyDeclarationIdentityClosure()
}

func (closure semanticPackageClosure) verifyMemberTargets() error {
	memberTargets, err := closure.pkg.MemberTargetCensus()
	if err != nil {
		return err
	}
	if memberTargets.Count() !=
		closure.owners.memberCounts[closure.packageID] ||
		memberTargets.Digest() !=
			closure.owners.memberDigests[closure.packageID] {
		return fmt.Errorf(
			"package %s member-target census disagrees with its manifest",
			closure.packageID,
		)
	}
	return nil
}

func (closure semanticPackageClosure) verifyDefinitionOwners() error {
	return closure.pkg.VisitDefinitions(func(
		record semantic.DefinitionSemantics,
	) error {
		owner, present :=
			closure.owners.definitions[record.Definition()]
		if !present || owner != closure.packageID {
			return fmt.Errorf(
				"definition %s has invalid package owner",
				record.Definition(),
			)
		}
		return nil
	})
}

func (closure semanticPackageClosure) verifyDeclarationOwners() error {
	return closure.pkg.VisitDeclarations(func(
		record semantic.Declaration,
	) error {
		owner, present :=
			closure.owners.declarations[record.ID()]
		if !present || owner != closure.packageID {
			return fmt.Errorf(
				"declaration %s has invalid package owner",
				record.ID(),
			)
		}
		return nil
	})
}

func (
	closure semanticPackageClosure,
) verifyDeclarationIdentityClosure() error {
	return closure.pkg.VisitDeclarationIdentities(func(
		id identity.SemanticDeclarationID,
	) error {
		if id.Form() == identity.SemanticDeclarationMember {
			return nil
		}
		if _, present := closure.owners.declarations[id]; !present {
			return fmt.Errorf(
				"declaration identity references absent semantic target %s",
				id,
			)
		}
		return nil
	})
}
