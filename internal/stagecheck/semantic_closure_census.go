package stagecheck

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/semantic"
)

type semanticOwnerCensus struct {
	definitions   map[identity.DefinitionID]identity.PackageID
	declarations  map[identity.SemanticDeclarationID]identity.PackageID
	memberCounts  map[identity.PackageID]int
	memberDigests map[identity.PackageID]string
}

func censusSemanticOwners(
	model *semantic.Model,
	packageIDs []identity.PackageID,
) (*semanticOwnerCensus, error) {
	owners := &semanticOwnerCensus{
		definitions:   map[identity.DefinitionID]identity.PackageID{},
		declarations:  map[identity.SemanticDeclarationID]identity.PackageID{},
		memberCounts:  map[identity.PackageID]int{},
		memberDigests: map[identity.PackageID]string{},
	}
	for _, packageID := range packageIDs {
		census, present, err := model.PackageCensus(packageID)
		if err != nil {
			return nil, err
		}
		if !present || census.Package != packageID {
			return nil, fmt.Errorf(
				"semantic census omits package %s", packageID,
			)
		}
		if err := owners.add(
			packageID,
			census.Definitions,
			census.Declarations,
		); err != nil {
			return nil, err
		}
		owners.memberCounts[packageID] = census.MemberTargetCount
		owners.memberDigests[packageID] = census.MemberTargetDigest
	}
	return owners, nil
}

func (owners *semanticOwnerCensus) add(
	packageID identity.PackageID,
	definitions []identity.DefinitionID,
	declarations []identity.SemanticDeclarationID,
) error {
	for _, definition := range definitions {
		if err := admitSemanticOwner(
			owners.definitions,
			definition,
			packageID,
			"definition",
		); err != nil {
			return err
		}
	}
	for _, declaration := range declarations {
		if err := admitSemanticOwner(
			owners.declarations,
			declaration,
			packageID,
			"declaration",
		); err != nil {
			return err
		}
	}
	return nil
}

func admitSemanticOwner[ID comparable](
	owners map[ID]identity.PackageID,
	id ID,
	pkg identity.PackageID,
	class string,
) error {
	if prior, present := owners[id]; present {
		return fmt.Errorf(
			"%s is owned by packages %s and %s",
			class, prior, pkg,
		)
	}
	owners[id] = pkg
	return nil
}
