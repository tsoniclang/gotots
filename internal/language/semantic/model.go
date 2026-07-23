package semantic

import (
	"fmt"

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
	return newPackage(input, true)
}

func newPackage(input PackageInput, clone bool) (Package, error) {
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
		definitions:   input.Definitions,
		resolutions:   input.Resolutions,
		declarations:  input.Declarations,
		bindings:      input.Bindings,
		types:         input.Types,
		typeWitnesses: input.TypeWitnesses,
		operations:    input.Operations,
		unsupported:   input.Unsupported,
	}
	if clone {
		out.definitions = append(
			[]DefinitionSemantics(nil), out.definitions...,
		)
		out.resolutions = append(
			[]OccurrenceResolution(nil), out.resolutions...,
		)
		out.declarations = append(
			[]Declaration(nil), out.declarations...,
		)
		out.bindings = append([]Binding(nil), out.bindings...)
		out.types = append([]Type(nil), out.types...)
		out.typeWitnesses = append(
			[]TypeWitness(nil), out.typeWitnesses...,
		)
		out.operations = append(
			[]Operation(nil), out.operations...,
		)
		out.unsupported = append(
			[]Unsupported(nil), out.unsupported...,
		)
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
	sortCanonical(pkg.definitions, func(record DefinitionSemantics) string {
		return record.Definition().String()
	})
	sortCanonical(pkg.resolutions, func(record OccurrenceResolution) string {
		return record.Occurrence().String()
	})
	sortCanonical(pkg.declarations, func(record Declaration) string {
		return record.ID().String()
	})
	sortCanonical(pkg.bindings, func(record Binding) string {
		return record.ID().String()
	})
	sortCanonical(pkg.types, func(record Type) string {
		return record.ID().String()
	})
	sortCanonical(pkg.typeWitnesses, func(record TypeWitness) string {
		return record.Type().String()
	})
	sortCanonical(pkg.operations, func(record Operation) string {
		return record.ID().String()
	})
	sortCanonical(pkg.unsupported, func(record Unsupported) string {
		return record.ID().String()
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
	sortCanonical(out.packages, func(pkg Package) string {
		return pkg.ID().String()
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
		for _, record := range pkg.definitions {
			if owner, duplicate := seenDefinitions[record.Definition()]; duplicate {
				return nil, fmt.Errorf(
					"semantic definition %s is owned by %s and %s",
					record.Definition(), owner, pkg.ID(),
				)
			}
			seenDefinitions[record.Definition()] = pkg.ID()
		}
		for _, record := range pkg.declarations {
			if owner, duplicate := seenDeclarations[record.ID()]; duplicate {
				return nil, fmt.Errorf(
					"semantic declaration %s is owned by %s and %s",
					record.ID(), owner, pkg.ID(),
				)
			}
			seenDeclarations[record.ID()] = pkg.ID()
		}
		for _, record := range pkg.bindings {
			if owner, duplicate := seenBindings[record.ID()]; duplicate {
				return nil, fmt.Errorf(
					"semantic binding %s is owned by %s and %s",
					record.ID(), owner, pkg.ID(),
				)
			}
			seenBindings[record.ID()] = pkg.ID()
		}
		for _, record := range pkg.operations {
			if owner, duplicate := seenOperations[record.ID()]; duplicate {
				return nil, fmt.Errorf(
					"semantic operation %s is owned by %s and %s",
					record.ID(), owner, pkg.ID(),
				)
			}
			seenOperations[record.ID()] = pkg.ID()
		}
		for _, record := range pkg.types {
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
		for _, record := range pkg.resolutions {
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
		index := searchCanonical(
			len(model.packages),
			func(index int) string {
				return model.packages[index].ID().String()
			},
			packageID.String(),
		)
		if index == len(model.packages) ||
			model.packages[index].ID() != packageID {
			return fmt.Errorf(
				"semantic model package %s is absent", packageID,
			)
		}
		return visit(model.packages[index])
	}
	index := searchCanonical(
		len(model.projections),
		func(index int) string {
			return model.projections[index].id.String()
		},
		packageID.String(),
	)
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

func (model *Model) ProviderManifestMetrics() Metrics {
	if model == nil || model.provider == nil {
		return Metrics{}
	}
	return model.provider.ManifestMetrics()
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
