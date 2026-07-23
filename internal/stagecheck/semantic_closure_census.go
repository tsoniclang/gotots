package stagecheck

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/semantic"
)

type semanticOwnerCensus struct {
	definitions  map[identity.DefinitionID]identity.PackageID
	declarations map[identity.SemanticDeclarationID]identity.PackageID
}

func verifySemanticModelClosure(
	model *semantic.Model,
	provider *semantic.ProviderArtifact,
	packageIDs []identity.PackageID,
) error {
	owners, err := censusSemanticOwners(
		model, provider, packageIDs,
	)
	if err != nil {
		return semanticVerificationError("closure", err.Error())
	}
	for _, packageID := range packageIDs {
		err := model.VisitPackage(
			packageID,
			func(pkg semantic.Package) error {
				return verifySemanticPackageClosure(pkg, owners)
			},
		)
		if err != nil {
			return semanticVerificationError(
				"closure", err.Error(),
			)
		}
	}
	return nil
}

func censusSemanticOwners(
	model *semantic.Model,
	provider *semantic.ProviderArtifact,
	packageIDs []identity.PackageID,
) (*semanticOwnerCensus, error) {
	owners := &semanticOwnerCensus{
		definitions:  map[identity.DefinitionID]identity.PackageID{},
		declarations: map[identity.SemanticDeclarationID]identity.PackageID{},
	}
	for _, packageID := range packageIDs {
		if provider != nil {
			context, present, err := provider.PackageContext(packageID)
			if err != nil {
				return nil, err
			}
			if present {
				if context.Package != packageID ||
					context.DefinitionCount != len(context.Definitions) ||
					context.DeclarationCount != len(context.Declarations) {
					return nil, fmt.Errorf(
						"provider census disagrees with package %s",
						packageID,
					)
				}
				if err := owners.add(
					packageID,
					context.Definitions,
					context.Declarations,
				); err != nil {
					return nil, err
				}
				continue
			}
		}
		err := model.VisitPackage(
			packageID,
			func(pkg semantic.Package) error {
				return owners.addPackage(pkg)
			},
		)
		if err != nil {
			return nil, err
		}
	}
	return owners, nil
}

func (owners *semanticOwnerCensus) addPackage(
	pkg semantic.Package,
) error {
	definitions := pkg.Definitions()
	declarations := pkg.Declarations()
	definitionIDs := make(
		[]identity.DefinitionID, 0, len(definitions),
	)
	declarationIDs := make(
		[]identity.SemanticDeclarationID, 0, len(declarations),
	)
	for _, record := range definitions {
		definitionIDs = append(
			definitionIDs, record.Definition(),
		)
	}
	for _, record := range declarations {
		declarationIDs = append(
			declarationIDs, record.ID(),
		)
	}
	return owners.add(pkg.ID(), definitionIDs, declarationIDs)
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
