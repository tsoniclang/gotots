package semantic

import (
	"fmt"
	"sort"

	"github.com/tsoniclang/gotots/internal/identity"
)

type PackageInput struct {
	ID            identity.PackageID
	Provenance    PackageProvenance
	Definitions   []DefinitionSemantics
	Resolutions   []OccurrenceResolution
	Declarations  []Declaration
	Bindings      []Binding
	Types         []Type
	TypeWitnesses []TypeWitness
	Operations    []Operation
	Unsupported   []Unsupported
}

type Package struct {
	id            identity.PackageID
	provenance    PackageProvenance
	definitions   []DefinitionSemantics
	resolutions   []OccurrenceResolution
	declarations  []Declaration
	bindings      []Binding
	types         []Type
	typeWitnesses []TypeWitness
	operations    []Operation
	unsupported   []Unsupported
}

func NewPackage(input PackageInput) (Package, error) {
	if input.ID.IsZero() {
		return Package{}, fmt.Errorf(
			"semantic package requires package identity",
		)
	}
	if !input.Provenance.Valid() {
		return Package{}, fmt.Errorf(
			"semantic package requires closed provenance",
		)
	}
	out := Package{
		id: input.ID, provenance: input.Provenance,
		definitions: append(
			[]DefinitionSemantics(nil), input.Definitions...,
		),
		resolutions: append(
			[]OccurrenceResolution(nil), input.Resolutions...,
		),
		declarations: append(
			[]Declaration(nil), input.Declarations...,
		),
		bindings: append([]Binding(nil), input.Bindings...),
		types:    append([]Type(nil), input.Types...),
		typeWitnesses: append(
			[]TypeWitness(nil), input.TypeWitnesses...,
		),
		operations: append(
			[]Operation(nil), input.Operations...,
		),
		unsupported: append(
			[]Unsupported(nil), input.Unsupported...,
		),
	}
	out.sort()
	if err := validatePackage(out); err != nil {
		return Package{}, err
	}
	return out, nil
}

func (pkg Package) ID() identity.PackageID { return pkg.id }
func (pkg Package) Provenance() PackageProvenance {
	return pkg.provenance
}
func (pkg Package) Definitions() []DefinitionSemantics {
	return append([]DefinitionSemantics(nil), pkg.definitions...)
}
func (pkg Package) Resolutions() []OccurrenceResolution {
	return append([]OccurrenceResolution(nil), pkg.resolutions...)
}
func (pkg Package) Declarations() []Declaration {
	return append([]Declaration(nil), pkg.declarations...)
}
func (pkg Package) Bindings() []Binding {
	return append([]Binding(nil), pkg.bindings...)
}
func (pkg Package) Types() []Type {
	return append([]Type(nil), pkg.types...)
}
func (pkg Package) TypeWitnesses() []TypeWitness {
	return append([]TypeWitness(nil), pkg.typeWitnesses...)
}
func (pkg Package) Operations() []Operation {
	return append([]Operation(nil), pkg.operations...)
}
func (pkg Package) Unsupported() []Unsupported {
	return append([]Unsupported(nil), pkg.unsupported...)
}

func (pkg *Package) sort() {
	sort.Slice(pkg.definitions, func(left, right int) bool {
		return pkg.definitions[left].Definition().String() <
			pkg.definitions[right].Definition().String()
	})
	sort.Slice(pkg.resolutions, func(left, right int) bool {
		return pkg.resolutions[left].Occurrence().String() <
			pkg.resolutions[right].Occurrence().String()
	})
	sort.Slice(pkg.declarations, func(left, right int) bool {
		return pkg.declarations[left].ID().String() <
			pkg.declarations[right].ID().String()
	})
	sort.Slice(pkg.bindings, func(left, right int) bool {
		return pkg.bindings[left].ID().String() <
			pkg.bindings[right].ID().String()
	})
	sort.Slice(pkg.types, func(left, right int) bool {
		return pkg.types[left].ID().String() <
			pkg.types[right].ID().String()
	})
	sort.Slice(pkg.typeWitnesses, func(left, right int) bool {
		return pkg.typeWitnesses[left].Type().String() <
			pkg.typeWitnesses[right].Type().String()
	})
	sort.Slice(pkg.operations, func(left, right int) bool {
		return pkg.operations[left].ID().String() <
			pkg.operations[right].ID().String()
	})
	sort.Slice(pkg.unsupported, func(left, right int) bool {
		return pkg.unsupported[left].ID().String() <
			pkg.unsupported[right].ID().String()
	})
}

type Model struct {
	packages    []Package
	projections []packageProjection
	provider    *ProviderArtifact
}

func NewModel(packages []Package) (*Model, error) {
	out := &Model{
		packages: append([]Package(nil), packages...),
	}
	sort.Slice(out.packages, func(left, right int) bool {
		return out.packages[left].ID().String() <
			out.packages[right].ID().String()
	})
	seenPackages := map[identity.PackageID]bool{}
	seenDefinitions := map[identity.DefinitionID]identity.PackageID{}
	seenDeclarations := map[identity.SemanticDeclarationID]identity.PackageID{}
	seenBindings := map[identity.SemanticBindingID]identity.PackageID{}
	seenOperations := map[identity.OperationID]identity.PackageID{}
	seenTypes := map[identity.SemanticTypeID]string{}
	for _, pkg := range out.packages {
		if seenPackages[pkg.ID()] {
			return nil, fmt.Errorf(
				"duplicate semantic package %s", pkg.ID(),
			)
		}
		seenPackages[pkg.ID()] = true
		for _, record := range pkg.Definitions() {
			if owner, duplicate := seenDefinitions[record.Definition()]; duplicate {
				return nil, fmt.Errorf(
					"semantic definition %s is owned by %s and %s",
					record.Definition(), owner, pkg.ID(),
				)
			}
			seenDefinitions[record.Definition()] = pkg.ID()
		}
		for _, record := range pkg.Declarations() {
			if owner, duplicate := seenDeclarations[record.ID()]; duplicate {
				return nil, fmt.Errorf(
					"semantic declaration %s is owned by %s and %s",
					record.ID(), owner, pkg.ID(),
				)
			}
			seenDeclarations[record.ID()] = pkg.ID()
		}
		for _, record := range pkg.Bindings() {
			if owner, duplicate := seenBindings[record.ID()]; duplicate {
				return nil, fmt.Errorf(
					"semantic binding %s is owned by %s and %s",
					record.ID(), owner, pkg.ID(),
				)
			}
			seenBindings[record.ID()] = pkg.ID()
		}
		for _, record := range pkg.Operations() {
			if owner, duplicate := seenOperations[record.ID()]; duplicate {
				return nil, fmt.Errorf(
					"semantic operation %s is owned by %s and %s",
					record.ID(), owner, pkg.ID(),
				)
			}
			seenOperations[record.ID()] = pkg.ID()
		}
		for _, record := range pkg.Types() {
			if canonical, present := seenTypes[record.ID()]; present &&
				canonical != record.Canonical() {
				return nil, fmt.Errorf(
					"semantic type collision %s", record.ID(),
				)
			}
			seenTypes[record.ID()] = record.Canonical()
		}
	}
	for _, pkg := range out.packages {
		for _, record := range pkg.Resolutions() {
			var declaration identity.SemanticDeclarationID
			switch record.Kind() {
			case ResolutionDeclaration:
				declaration = record.Declaration()
			case ResolutionStructuralOnly:
				declaration = record.Structural().Declaration()
			}
			if !declaration.IsZero() {
				if _, present := seenDeclarations[declaration]; !present {
					return nil, fmt.Errorf(
						"semantic resolution %s references absent declaration %s",
						record.Occurrence(), declaration,
					)
				}
			}
		}
	}
	return out, nil
}

func (model *Model) PackageCount() int {
	if model == nil {
		return 0
	}
	if len(model.projections) != 0 {
		return len(model.projections)
	}
	return len(model.packages)
}

func (model *Model) VisitPackage(
	packageID identity.PackageID,
	visit func(Package) error,
) error {
	if model == nil || packageID.IsZero() || visit == nil {
		return fmt.Errorf(
			"semantic model package visit requires model, package, and visitor",
		)
	}
	if len(model.projections) == 0 {
		index := sort.Search(len(model.packages), func(index int) bool {
			return model.packages[index].ID().String() >=
				packageID.String()
		})
		if index == len(model.packages) ||
			model.packages[index].ID() != packageID {
			return fmt.Errorf(
				"semantic model package %s is absent", packageID,
			)
		}
		return visit(model.packages[index])
	}
	index := sort.Search(len(model.projections), func(index int) bool {
		return model.projections[index].id.String() >=
			packageID.String()
	})
	if index == len(model.projections) ||
		model.projections[index].id != packageID {
		return fmt.Errorf(
			"semantic model package %s is absent", packageID,
		)
	}
	projection := model.projections[index]
	if !projection.certified {
		pkg, err := projection.completeLocal()
		if err != nil {
			return err
		}
		return visit(pkg)
	}
	return model.provider.VisitPackage(
		projection.id,
		func(provider Package) error {
			pkg, err := projection.merge(provider)
			if err != nil {
				return err
			}
			return visit(pkg)
		},
	)
}

func (model *Model) VisitPackages(
	visit func(Package) error,
) error {
	if model == nil || visit == nil {
		return fmt.Errorf(
			"semantic model package visit requires model and visitor",
		)
	}
	if len(model.projections) == 0 {
		for _, pkg := range model.packages {
			if err := visit(pkg); err != nil {
				return err
			}
		}
		return nil
	}
	for _, projection := range model.projections {
		if err := model.VisitPackage(projection.id, visit); err != nil {
			return err
		}
	}
	return nil
}

func (model *Model) ProviderReadStats() ProviderReadStats {
	if model == nil || model.provider == nil {
		return ProviderReadStats{}
	}
	return model.provider.ReadStats()
}
func validateTypeRecords(
	records []Type,
	types map[identity.SemanticTypeID]string,
) error {
	for _, record := range records {
		for _, reference := range referencedTypeIDs(record) {
			if reference.IsZero() {
				continue
			}
			if _, present := types[reference]; !present {
				return fmt.Errorf(
					"semantic type %s references absent type %s",
					record.ID(), reference,
				)
			}
		}
	}
	return nil
}

func validateTypeClosure(
	packages []Package,
	types map[identity.SemanticTypeID]string,
) error {
	require := func(
		owner string,
		typeID identity.SemanticTypeID,
	) error {
		if typeID.IsZero() {
			return nil
		}
		if _, present := types[typeID]; !present {
			return fmt.Errorf(
				"%s references absent semantic type %s",
				owner, typeID,
			)
		}
		return nil
	}
	for _, pkg := range packages {
		for _, record := range pkg.definitions {
			spec := record.Spec()
			if err := require(
				record.Definition().String(), spec.Signature,
			); err != nil {
				return err
			}
		}
		for _, record := range pkg.declarations {
			if err := require(
				record.ID().String(), record.Type(),
			); err != nil {
				return err
			}
		}
		for _, record := range pkg.bindings {
			if err := require(
				record.ID().String(), record.Type(),
			); err != nil {
				return err
			}
		}
		for _, record := range pkg.operations {
			spec := record.Spec()
			for _, typeID := range []identity.SemanticTypeID{
				spec.ResultType,
				spec.ExpectedType,
			} {
				if err := require(
					record.ID().String(), typeID,
				); err != nil {
					return err
				}
			}
		}
		for _, record := range pkg.resolutions {
			if err := require(
				record.Occurrence().String(), record.Type(),
			); err != nil {
				return err
			}
			if err := require(
				record.Occurrence().String(),
				record.Structural().Type(),
			); err != nil {
				return err
			}
		}
	}
	return nil
}
