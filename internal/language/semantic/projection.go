package semantic

import (
	"fmt"
	"sort"

	"github.com/tsoniclang/gotots/internal/identity"
)

type PackageProjectionInput struct {
	ID                  identity.PackageID
	Provenance          PackageProvenance
	ExpectedDefinitions []identity.DefinitionID
	Local               bool
	LocalFiles          []identity.FileID
	LocalSynthetic      bool
	LocalDeclarations   []identity.SemanticDeclarationID
	Certified           bool
}

type packageProjection struct {
	id                  identity.PackageID
	provenance          PackageProvenance
	expectedDefinitions []identity.DefinitionID
	local               bool
	localDefinitions    map[identity.DefinitionID]bool
	localFiles          map[identity.FileID]bool
	localSynthetic      bool
	certified           bool
}

func newPackageProjection(
	input PackageProjectionInput,
	checker *CheckerStore,
) (packageProjection, error) {
	if input.ID.IsZero() || !input.Provenance.Valid() {
		return packageProjection{}, fmt.Errorf(
			"semantic package projection requires package and provenance",
		)
	}
	projection := packageProjection{
		id:         input.ID,
		provenance: input.Provenance,
		expectedDefinitions: append(
			[]identity.DefinitionID(nil),
			input.ExpectedDefinitions...,
		),
		local:            input.Local,
		localDefinitions: map[identity.DefinitionID]bool{},
		localFiles:       map[identity.FileID]bool{},
		localSynthetic:   input.LocalSynthetic,
		certified:        input.Certified,
	}
	if err := projection.validateExpectedDefinitions(); err != nil {
		return packageProjection{}, err
	}
	for _, file := range input.LocalFiles {
		if file.IsZero() || projection.localFiles[file] {
			return packageProjection{}, fmt.Errorf(
				"semantic projection has invalid local file %s", file,
			)
		}
		projection.localFiles[file] = true
	}
	if !input.Local {
		if len(input.LocalDeclarations) != 0 ||
			len(input.LocalFiles) != 0 ||
			input.LocalSynthetic {
			return packageProjection{}, fmt.Errorf(
				"semantic projection has local selections without local package",
			)
		}
		return projection, nil
	}
	if checker == nil {
		return packageProjection{}, fmt.Errorf(
			"semantic projection %s requires absent checker storage",
			input.ID,
		)
	}
	context, present, err := checker.PackageContext(input.ID)
	if err != nil {
		return packageProjection{}, err
	}
	if !present ||
		context.Package != input.ID ||
		context.Provenance != input.Provenance {
		return packageProjection{}, fmt.Errorf(
			"semantic checker package disagrees with projection",
		)
	}
	selectedDeclarations := append(
		[]identity.SemanticDeclarationID(nil),
		input.LocalDeclarations...,
	)
	sort.Slice(selectedDeclarations, func(left, right int) bool {
		return selectedDeclarations[left].Compare(
			selectedDeclarations[right],
		) < 0
	})
	if len(selectedDeclarations) != len(context.Declarations) {
		return packageProjection{}, fmt.Errorf(
			"semantic checker declaration census differs for %s",
			input.ID,
		)
	}
	for _, declaration := range input.LocalDeclarations {
		if declaration.IsZero() {
			return packageProjection{}, fmt.Errorf(
				"semantic projection has invalid local declaration %s",
				declaration,
			)
		}
	}
	for index, declaration := range context.Declarations {
		if declaration != selectedDeclarations[index] {
			return packageProjection{}, fmt.Errorf(
				"semantic checker declaration census differs at %s",
				declaration,
			)
		}
	}
	for _, definition := range context.Definitions {
		projection.localDefinitions[definition] = true
	}
	return projection, nil
}

func (projection *packageProjection) validateExpectedDefinitions() error {
	sort.Slice(
		projection.expectedDefinitions,
		func(left, right int) bool {
			return projection.expectedDefinitions[left].Compare(
				projection.expectedDefinitions[right],
			) < 0
		},
	)
	var previous identity.DefinitionID
	for _, definition := range projection.expectedDefinitions {
		if definition.IsZero() || definition == previous {
			return fmt.Errorf(
				"semantic projection has invalid expected definition %s",
				definition,
			)
		}
		previous = definition
	}
	return nil
}

func (projection packageProjection) completeLocal(pkg Package) (Package, error) {
	if projection.certified {
		return Package{}, fmt.Errorf(
			"certified semantic projection requires provider package",
		)
	}
	if pkg.ID() != projection.id ||
		pkg.Provenance() != projection.provenance ||
		pkg.DefinitionCount() !=
			len(projection.expectedDefinitions) {
		return Package{}, fmt.Errorf(
			"local semantic package %s has %d of %d definitions",
			projection.id,
			pkg.DefinitionCount(),
			len(projection.expectedDefinitions),
		)
	}
	return pkg, nil
}

func (projection packageProjection) completeProvider(
	provider Package,
) (Package, error) {
	if !projection.certified ||
		provider.ID() != projection.id ||
		provider.Provenance() != projection.provenance {
		return Package{}, fmt.Errorf(
			"semantic provider package disagrees with projection %s",
			projection.id,
		)
	}
	if err := projection.verifyProviderDefinitions(provider); err != nil {
		return Package{}, err
	}
	if projection.local {
		return Package{}, fmt.Errorf(
			"mixed semantic projection %s requires streaming authorities",
			projection.id,
		)
	}
	return provider, nil
}

func (projection packageProjection) verifyProviderDefinitions(
	provider Package,
) error {
	actual := make(
		[]identity.DefinitionID, 0, provider.DefinitionCount(),
	)
	if err := provider.VisitDefinitions(func(
		record DefinitionSemantics,
	) error {
		actual = append(actual, record.Definition())
		return nil
	}); err != nil {
		return err
	}
	sort.Slice(actual, func(left, right int) bool {
		return actual[left].Compare(actual[right]) < 0
	})
	if len(actual) != len(projection.expectedDefinitions) {
		return fmt.Errorf(
			"semantic provider package %s definition census differs",
			projection.id,
		)
	}
	for index := range actual {
		if actual[index] != projection.expectedDefinitions[index] {
			return fmt.Errorf(
				"semantic provider package %s definition census differs at %s",
				projection.id, actual[index],
			)
		}
	}
	return nil
}

func (projection packageProjection) definitionIsLocal(
	definition identity.DefinitionID,
) bool {
	if !definition.File().IsZero() {
		return projection.localFiles[definition.File()]
	}
	if definition.SyntheticRole().Valid() {
		return projection.localSynthetic
	}
	return projection.localDefinitions[definition]
}
