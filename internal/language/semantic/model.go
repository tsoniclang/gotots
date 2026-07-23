package semantic

import (
	"fmt"
	"sort"

	"github.com/tsoniclang/gotots/internal/identity"
)

type PackageInput struct {
	ID            identity.PackageID
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
	out := Package{
		id: input.ID,
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

func validatePackage(pkg Package) error {
	definitions := map[identity.DefinitionID]bool{}
	for _, record := range pkg.definitions {
		if record.Package() != pkg.id ||
			definitions[record.Definition()] {
			return fmt.Errorf(
				"semantic package %s has invalid definition %s",
				pkg.id, record.Definition(),
			)
		}
		definitions[record.Definition()] = true
	}
	declarations := map[identity.SemanticDeclarationID]bool{}
	for _, record := range pkg.declarations {
		if record.ID().IsZero() ||
			declarations[record.ID()] {
			return fmt.Errorf(
				"semantic package %s has duplicate declaration %s",
				pkg.id, record.ID(),
			)
		}
		declarations[record.ID()] = true
	}
	bindings := map[identity.SemanticBindingID]bool{}
	for _, record := range pkg.bindings {
		if record.Package() != pkg.id ||
			(!record.Definition().IsZero() &&
				!definitions[record.Definition()]) ||
			bindings[record.ID()] {
			return fmt.Errorf(
				"semantic package %s has invalid binding %s",
				pkg.id, record.ID(),
			)
		}
		bindings[record.ID()] = true
	}
	types := map[identity.SemanticTypeID]string{}
	for _, record := range pkg.types {
		if existing, duplicate := types[record.ID()]; duplicate {
			if existing != record.Canonical() {
				return fmt.Errorf(
					"semantic package %s has type collision %s",
					pkg.id, record.ID(),
				)
			}
			return fmt.Errorf(
				"semantic package %s duplicates type %s",
				pkg.id, record.ID(),
			)
		}
		types[record.ID()] = record.Canonical()
	}
	witnesses := map[identity.SemanticTypeID]bool{}
	for _, record := range pkg.typeWitnesses {
		if record.Package() != pkg.id ||
			types[record.Type()] == "" ||
			witnesses[record.Type()] {
			return fmt.Errorf(
				"semantic package %s has invalid type witness %s",
				pkg.id, record.Type(),
			)
		}
		witnesses[record.Type()] = true
	}
	if len(witnesses) != len(types) {
		return fmt.Errorf(
			"semantic package %s has %d types and %d witnesses",
			pkg.id, len(types), len(witnesses),
		)
	}
	operations := map[identity.OperationID]bool{}
	for _, record := range pkg.operations {
		if !definitions[record.Definition()] ||
			operations[record.ID()] {
			return fmt.Errorf(
				"semantic package %s has invalid operation %s",
				pkg.id, record.ID(),
			)
		}
		operations[record.ID()] = true
	}
	unsupported := map[identity.UnsupportedID]bool{}
	for _, record := range pkg.unsupported {
		if !definitions[record.ID().Definition()] ||
			unsupported[record.ID()] {
			return fmt.Errorf(
				"semantic package %s has invalid unsupported record %s",
				pkg.id, record.ID(),
			)
		}
		unsupported[record.ID()] = true
	}
	resolutions := map[identity.OccurrenceID]bool{}
	for _, record := range pkg.resolutions {
		if resolutions[record.Occurrence()] {
			return fmt.Errorf(
				"semantic package %s has duplicate resolution %s",
				pkg.id, record.Occurrence(),
			)
		}
		resolutions[record.Occurrence()] = true
		switch record.Kind() {
		case ResolutionDefinitionComponent:
			if !definitions[record.Definition()] {
				return fmt.Errorf(
					"resolution %s names absent definition",
					record.Occurrence(),
				)
			}
		case ResolutionDeclaration:
			if !declarations[record.Declaration()] {
				return fmt.Errorf(
					"resolution %s names absent declaration",
					record.Occurrence(),
				)
			}
		case ResolutionBinding:
			if !bindings[record.Binding()] {
				return fmt.Errorf(
					"resolution %s names absent binding",
					record.Occurrence(),
				)
			}
		case ResolutionOperation:
			if !operations[record.Operation()] {
				return fmt.Errorf(
					"resolution %s names absent operation",
					record.Occurrence(),
				)
			}
		case ResolutionUnsupported:
			if !unsupported[record.Unsupported()] {
				return fmt.Errorf(
					"resolution %s names absent unsupported record",
					record.Occurrence(),
				)
			}
		}
	}
	if err := validateTypeRecords(pkg.types, types); err != nil {
		return err
	}
	if err := validateTypeClosure([]Package{pkg}, types); err != nil {
		return err
	}
	return nil
}

type Model struct {
	packages []Package
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
	return out, nil
}

func (model *Model) Packages() []Package {
	return append([]Package(nil), model.packages...)
}
func validateTypeRecords(
	records []Type,
	types map[identity.SemanticTypeID]string,
) error {
	for _, record := range records {
		spec := record.Spec()
		var references []identity.SemanticTypeID
		references = append(references, spec.Arguments...)
		references = append(
			references,
			spec.Underlying,
			spec.Target,
			spec.Constraint,
			spec.Element,
			spec.Key,
			spec.Signature.Receiver,
		)
		references = append(
			references, spec.Signature.TypeParameters...,
		)
		references = append(
			references, spec.Signature.Parameters...,
		)
		references = append(
			references, spec.Signature.Results...,
		)
		for _, field := range spec.Fields {
			references = append(references, field.Type)
		}
		for _, method := range spec.Methods {
			references = append(references, method.Signature)
		}
		references = append(references, spec.Embeddeds...)
		for _, term := range spec.Terms {
			references = append(references, term.Type)
		}
		references = append(references, spec.Elements...)
		for _, reference := range references {
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
		for _, record := range pkg.Definitions() {
			spec := record.Spec()
			if err := require(
				record.Definition().String(), spec.Signature,
			); err != nil {
				return err
			}
			for _, typeID := range spec.Types {
				if err := require(
					record.Definition().String(), typeID,
				); err != nil {
					return err
				}
			}
		}
		for _, record := range pkg.Declarations() {
			if err := require(
				record.ID().String(), record.Type(),
			); err != nil {
				return err
			}
		}
		for _, record := range pkg.Bindings() {
			if err := require(
				record.ID().String(), record.Type(),
			); err != nil {
				return err
			}
		}
		for _, record := range pkg.Operations() {
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
		for _, record := range pkg.Resolutions() {
			if err := require(
				record.Occurrence().String(), record.Type(),
			); err != nil {
				return err
			}
		}
	}
	return nil
}
