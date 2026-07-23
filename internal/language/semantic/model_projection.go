package semantic

import (
	"fmt"
	"sort"

	"github.com/tsoniclang/gotots/internal/identity"
)

func NewProjectedModel(
	inputs []PackageProjectionInput,
	provider *ProviderArtifact,
) (*Model, error) {
	out := &Model{provider: provider}
	for _, input := range inputs {
		projection, err := newPackageProjection(input)
		if err != nil {
			return nil, err
		}
		out.projections = append(out.projections, projection)
	}
	sort.Slice(out.projections, func(left, right int) bool {
		return out.projections[left].id.String() <
			out.projections[right].id.String()
	})
	if err := validateProjectedModel(out); err != nil {
		return nil, err
	}
	return out, nil
}

func validateProjectedModel(model *Model) error {
	if model == nil || len(model.projections) == 0 {
		return fmt.Errorf(
			"projected semantic model requires package projections",
		)
	}
	expectedProvider := map[identity.PackageID]bool{}
	seenPackages := map[identity.PackageID]bool{}
	seenDefinitions := map[identity.DefinitionID]identity.PackageID{}
	seenDeclarations := map[identity.SemanticDeclarationID]identity.PackageID{}
	for _, projection := range model.projections {
		if seenPackages[projection.id] {
			return fmt.Errorf(
				"semantic package projection repeats %s", projection.id,
			)
		}
		seenPackages[projection.id] = true
		for _, definition := range projection.expectedDefinitions {
			if owner, duplicate := seenDefinitions[definition]; duplicate {
				return fmt.Errorf(
					"semantic definition %s belongs to %s and %s",
					definition, owner, projection.id,
				)
			}
			seenDefinitions[definition] = projection.id
		}
		if projection.certified {
			expectedProvider[projection.id] = true
			if err := validateProviderProjectionManifest(
				model.provider, projection,
			); err != nil {
				return err
			}
			context, _, _ := model.provider.PackageContext(
				projection.id,
			)
			for _, declaration := range context.Declarations {
				if owner, duplicate := seenDeclarations[declaration]; duplicate {
					return fmt.Errorf(
						"semantic declaration %s belongs to %s and %s",
						declaration, owner, projection.id,
					)
				}
				seenDeclarations[declaration] = projection.id
			}
			continue
		}
		pkg, err := projection.completeLocal()
		if err != nil {
			return err
		}
		for _, declaration := range pkg.declarations {
			if owner, duplicate := seenDeclarations[declaration.ID()]; duplicate {
				return fmt.Errorf(
					"semantic declaration %s belongs to %s and %s",
					declaration.ID(), owner, projection.id,
				)
			}
			seenDeclarations[declaration.ID()] = projection.id
		}
	}
	if err := validateProviderPackageSet(
		model.provider, expectedProvider,
	); err != nil {
		return err
	}
	return nil
}

func validateProviderProjectionManifest(
	provider *ProviderArtifact,
	projection packageProjection,
) error {
	if provider == nil {
		return fmt.Errorf(
			"semantic package %s requires absent provider artifact",
			projection.id,
		)
	}
	context, present, err := provider.PackageContext(projection.id)
	if err != nil {
		return err
	}
	if !present ||
		context.Package != projection.id ||
		context.Provenance != projection.provenance ||
		len(context.Definitions) != len(projection.expectedDefinitions) {
		return fmt.Errorf(
			"semantic provider manifest disagrees with package %s",
			projection.id,
		)
	}
	for index, definition := range context.Definitions {
		if definition != projection.expectedDefinitions[index] {
			return fmt.Errorf(
				"semantic provider manifest definition differs at %s",
				definition,
			)
		}
	}
	return nil
}

func validateProviderPackageSet(
	provider *ProviderArtifact,
	expected map[identity.PackageID]bool,
) error {
	if len(expected) == 0 {
		if provider != nil && len(provider.PackageIDs()) != 0 {
			return fmt.Errorf(
				"semantic provider artifact is bound but no package selects it",
			)
		}
		return nil
	}
	if provider == nil {
		return fmt.Errorf(
			"semantic provider package set requires an artifact",
		)
	}
	actual := provider.PackageIDs()
	if len(actual) != len(expected) {
		return fmt.Errorf(
			"semantic provider package set has %d entries, expected %d",
			len(actual), len(expected),
		)
	}
	for _, packageID := range actual {
		if !expected[packageID] {
			return fmt.Errorf(
				"semantic provider artifact has unselected package %s",
				packageID,
			)
		}
	}
	return nil
}
