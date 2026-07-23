package semantic

import (
	"fmt"
	"sort"

	"github.com/tsoniclang/gotots/internal/identity"
)

type PackageInput struct {
	ID           identity.PackageID
	Definitions  []DefinitionSemantics
	Resolutions  []OccurrenceResolution
	Declarations []Declaration
	Bindings     []Binding
	Operations   []Operation
	Unsupported  []Unsupported
}

type Package struct {
	id           identity.PackageID
	definitions  []DefinitionSemantics
	resolutions  []OccurrenceResolution
	declarations []Declaration
	bindings     []Binding
	operations   []Operation
	unsupported  []Unsupported
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
		if !definitions[record.Definition()] ||
			bindings[record.ID()] {
			return fmt.Errorf(
				"semantic package %s has invalid binding %s",
				pkg.id, record.ID(),
			)
		}
		bindings[record.ID()] = true
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
	return nil
}

type Model struct {
	packages []Package
	types    []Type
}

func NewModel(
	packages []Package,
	types []Type,
) (*Model, error) {
	out := &Model{
		packages: append([]Package(nil), packages...),
		types:    append([]Type(nil), types...),
	}
	sort.Slice(out.packages, func(left, right int) bool {
		return out.packages[left].ID().String() <
			out.packages[right].ID().String()
	})
	sort.Slice(out.types, func(left, right int) bool {
		return out.types[left].ID().String() <
			out.types[right].ID().String()
	})
	seenPackages := map[identity.PackageID]bool{}
	for _, pkg := range out.packages {
		if seenPackages[pkg.ID()] {
			return nil, fmt.Errorf(
				"duplicate semantic package %s", pkg.ID(),
			)
		}
		seenPackages[pkg.ID()] = true
	}
	seenTypes := map[identity.SemanticTypeID]string{}
	for _, record := range out.types {
		if existing, present := seenTypes[record.ID()]; present {
			if existing != record.Canonical() {
				return nil, fmt.Errorf(
					"semantic type collision %s", record.ID(),
				)
			}
			return nil, fmt.Errorf(
				"duplicate semantic type %s", record.ID(),
			)
		}
		seenTypes[record.ID()] = record.Canonical()
	}
	if err := validateTypeRecords(out.types, seenTypes); err != nil {
		return nil, err
	}
	if err := validateTypeClosure(out.packages, seenTypes); err != nil {
		return nil, err
	}
	return out, nil
}

func (model *Model) Packages() []Package {
	return append([]Package(nil), model.packages...)
}
func (model *Model) Types() []Type {
	return append([]Type(nil), model.types...)
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
