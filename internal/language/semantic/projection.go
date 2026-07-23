package semantic

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/tsoniclang/gotots/internal/identity"
)

type PackageProjectionInput struct {
	ID                  identity.PackageID
	Provenance          PackageProvenance
	ExpectedDefinitions []identity.DefinitionID
	Local               *Package
	LocalFiles          []identity.FileID
	LocalSynthetic      bool
	LocalDeclarations   []identity.SemanticDeclarationID
	Certified           bool
}

type packageProjection struct {
	id                  identity.PackageID
	provenance          PackageProvenance
	expectedDefinitions []identity.DefinitionID
	local               PackageInput
	localDefinitions    map[identity.DefinitionID]bool
	localOccurrences    map[identity.OccurrenceID]bool
	localDeclarations   map[identity.SemanticDeclarationID]bool
	localFiles          map[identity.FileID]bool
	localSynthetic      bool
	certified           bool
}

func newPackageProjection(
	input PackageProjectionInput,
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
		localDefinitions:  map[identity.DefinitionID]bool{},
		localOccurrences:  map[identity.OccurrenceID]bool{},
		localDeclarations: map[identity.SemanticDeclarationID]bool{},
		localFiles:        map[identity.FileID]bool{},
		localSynthetic:    input.LocalSynthetic,
		certified:         input.Certified,
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
	if input.Local == nil {
		if len(input.LocalDeclarations) != 0 ||
			len(input.LocalFiles) != 0 ||
			input.LocalSynthetic {
			return packageProjection{}, fmt.Errorf(
				"semantic projection has local selections without local package",
			)
		}
		return projection, nil
	}
	if input.Local.ID() != input.ID ||
		input.Local.Provenance() != input.Provenance {
		return packageProjection{}, fmt.Errorf(
			"semantic local package disagrees with projection",
		)
	}
	selectedDeclarations := map[identity.SemanticDeclarationID]bool{}
	for _, declaration := range input.LocalDeclarations {
		if declaration.IsZero() || selectedDeclarations[declaration] {
			return packageProjection{}, fmt.Errorf(
				"semantic projection has invalid local declaration %s",
				declaration,
			)
		}
		selectedDeclarations[declaration] = true
	}
	local, err := projectLocalPackage(
		*input.Local, selectedDeclarations,
	)
	if err != nil {
		return packageProjection{}, err
	}
	projection.local = local
	for _, record := range local.Definitions {
		projection.localDefinitions[record.Definition()] = true
	}
	for _, record := range local.Resolutions {
		projection.localOccurrences[record.Occurrence()] = true
	}
	for _, record := range local.Declarations {
		projection.localDeclarations[record.ID()] = true
	}
	return projection, nil
}

func (projection *packageProjection) validateExpectedDefinitions() error {
	sort.Slice(
		projection.expectedDefinitions,
		func(left, right int) bool {
			return projection.expectedDefinitions[left].String() <
				projection.expectedDefinitions[right].String()
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

func projectLocalPackage(
	pkg Package,
	selectedDeclarations map[identity.SemanticDeclarationID]bool,
) (PackageInput, error) {
	input := PackageInput{
		ID: pkg.ID(), Provenance: pkg.Provenance(),
		Definitions: pkg.Definitions(),
		Resolutions: pkg.Resolutions(),
		Bindings:    pkg.Bindings(),
		Operations:  pkg.Operations(),
		Unsupported: pkg.Unsupported(),
	}
	foundDeclarations := map[identity.SemanticDeclarationID]bool{}
	for _, record := range pkg.Declarations() {
		if !selectedDeclarations[record.ID()] {
			continue
		}
		input.Declarations = append(input.Declarations, record)
		foundDeclarations[record.ID()] = true
	}
	if len(foundDeclarations) != len(selectedDeclarations) {
		return PackageInput{}, fmt.Errorf(
			"semantic local declaration selection is not a subset of package %s",
			pkg.ID(),
		)
	}
	types, witnesses, err := projectTypeClosure(pkg, input)
	if err != nil {
		return PackageInput{}, err
	}
	input.Types = types
	input.TypeWitnesses = witnesses
	return input, nil
}

func projectTypeClosure(
	pkg Package,
	input PackageInput,
) ([]Type, []TypeWitness, error) {
	records := map[identity.SemanticTypeID]Type{}
	for _, record := range pkg.Types() {
		records[record.ID()] = record
	}
	witnesses := map[identity.SemanticTypeID]TypeWitness{}
	for _, witness := range pkg.TypeWitnesses() {
		witnesses[witness.Type()] = witness
	}
	selected := map[identity.SemanticTypeID]bool{}
	queue := packageInputTypeRoots(input)
	for len(queue) != 0 {
		typeID := queue[len(queue)-1]
		queue = queue[:len(queue)-1]
		if typeID.IsZero() || selected[typeID] {
			continue
		}
		record, present := records[typeID]
		if !present {
			return nil, nil, fmt.Errorf(
				"semantic local projection references absent type %s",
				typeID,
			)
		}
		selected[typeID] = true
		queue = append(queue, referencedTypeIDs(record)...)
	}
	typeIDs := make([]identity.SemanticTypeID, 0, len(selected))
	for typeID := range selected {
		typeIDs = append(typeIDs, typeID)
	}
	sort.Slice(typeIDs, func(left, right int) bool {
		return typeIDs[left].String() < typeIDs[right].String()
	})
	out := make([]Type, 0, len(typeIDs))
	outWitnesses := make([]TypeWitness, 0, len(typeIDs))
	for _, typeID := range typeIDs {
		witness, present := witnesses[typeID]
		if !present {
			return nil, nil, fmt.Errorf(
				"semantic local projection has no type witness %s",
				typeID,
			)
		}
		out = append(out, records[typeID])
		outWitnesses = append(outWitnesses, witness)
	}
	return out, outWitnesses, nil
}

func (projection packageProjection) completeLocal() (Package, error) {
	if projection.certified {
		return Package{}, fmt.Errorf(
			"certified semantic projection requires provider package",
		)
	}
	if len(projection.local.Definitions) !=
		len(projection.expectedDefinitions) {
		return Package{}, fmt.Errorf(
			"local semantic package %s has %d of %d definitions",
			projection.id,
			len(projection.local.Definitions),
			len(projection.expectedDefinitions),
		)
	}
	return NewPackage(projection.local)
}

func (projection packageProjection) merge(
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
	if err := projection.verifySelectedAuthority(provider); err != nil {
		return Package{}, err
	}
	input := PackageInput{
		ID:            provider.ID(),
		Provenance:    provider.Provenance(),
		Definitions:   provider.Definitions(),
		Resolutions:   provider.Resolutions(),
		Declarations:  provider.Declarations(),
		Bindings:      provider.Bindings(),
		Types:         provider.Types(),
		TypeWitnesses: provider.TypeWitnesses(),
		Operations:    provider.Operations(),
		Unsupported:   provider.Unsupported(),
	}
	if err := overlayLocalPackage(&input, projection.local); err != nil {
		return Package{}, err
	}
	return NewPackage(input)
}

func (projection packageProjection) verifyProviderDefinitions(
	provider Package,
) error {
	actual := make(
		[]identity.DefinitionID, 0, len(provider.Definitions()),
	)
	for _, record := range provider.Definitions() {
		actual = append(actual, record.Definition())
	}
	sort.Slice(actual, func(left, right int) bool {
		return actual[left].String() < actual[right].String()
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

func (projection packageProjection) verifySelectedAuthority(
	provider Package,
) error {
	for _, record := range provider.Definitions() {
		if projection.definitionIsLocal(record.Definition()) &&
			!projection.localDefinitions[record.Definition()] {
			return fmt.Errorf(
				"local definition %s has no checker semantic record",
				record.Definition(),
			)
		}
	}
	for _, record := range provider.Resolutions() {
		if projection.localFiles[record.Occurrence().Span().File()] &&
			!projection.localOccurrences[record.Occurrence()] {
			return fmt.Errorf(
				"local occurrence %s has no checker semantic resolution",
				record.Occurrence(),
			)
		}
	}
	for _, record := range provider.Declarations() {
		if !record.Source().IsZero() &&
			projection.localFiles[record.Source().Span().File()] &&
			!projection.localDeclarations[record.ID()] {
			return fmt.Errorf(
				"local declaration %s has no checker semantic record",
				record.ID(),
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

func overlayLocalPackage(
	target *PackageInput,
	local PackageInput,
) error {
	var err error
	target.Definitions, err = overlayDefinitions(
		target.Definitions, local.Definitions,
	)
	if err != nil {
		return err
	}
	target.Resolutions, err = overlayResolutions(
		target.Resolutions, local.Resolutions,
	)
	if err != nil {
		return err
	}
	target.Declarations, err = overlayDeclarations(
		target.Declarations, local.Declarations,
	)
	if err != nil {
		return err
	}
	target.Bindings, err = overlayBindings(
		target.Bindings, local.Bindings,
	)
	if err != nil {
		return err
	}
	target.Types, target.TypeWitnesses, err = overlayTypes(
		target.Types,
		target.TypeWitnesses,
		local.Types,
		local.TypeWitnesses,
	)
	if err != nil {
		return err
	}
	target.Operations, err = overlayOperations(
		target.Operations, local.Operations,
	)
	if err != nil {
		return err
	}
	target.Unsupported, err = overlayUnsupported(
		target.Unsupported, local.Unsupported,
	)
	return err
}

func canonicalWireEqual[T any](left, right T) (bool, error) {
	leftBytes, err := json.Marshal(left)
	if err != nil {
		return false, err
	}
	rightBytes, err := json.Marshal(right)
	if err != nil {
		return false, err
	}
	return bytes.Equal(leftBytes, rightBytes), nil
}
