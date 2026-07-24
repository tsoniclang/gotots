package compiler

import (
	"slices"
	"testing"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/semantic"
)

func semanticPackageInput(pkg semantic.Package) semantic.PackageInput {
	input := semantic.PackageInput{
		ID: pkg.ID(), Provenance: pkg.Provenance(),
	}
	if err := pkg.VisitDefinitions(func(
		record semantic.DefinitionSemantics,
	) error {
		input.Definitions = append(input.Definitions, record)
		return nil
	}); err != nil {
		panic(err)
	}
	if err := pkg.VisitResolutions(func(
		record semantic.OccurrenceResolution,
	) error {
		input.Resolutions = append(input.Resolutions, record)
		return nil
	}); err != nil {
		panic(err)
	}
	if err := pkg.VisitDeclarations(func(record semantic.Declaration) error {
		input.Declarations = append(input.Declarations, record)
		return nil
	}); err != nil {
		panic(err)
	}
	if err := pkg.VisitBindings(func(record semantic.Binding) error {
		input.Bindings = append(input.Bindings, record)
		return nil
	}); err != nil {
		panic(err)
	}
	if err := pkg.VisitTypes(func(record semantic.Type) error {
		input.Types = append(input.Types, record)
		return nil
	}); err != nil {
		panic(err)
	}
	if err := pkg.VisitTypeWitnesses(func(record semantic.TypeWitness) error {
		input.TypeWitnesses = append(input.TypeWitnesses, record)
		return nil
	}); err != nil {
		panic(err)
	}
	if err := pkg.VisitOperations(func(record semantic.Operation) error {
		input.Operations = append(input.Operations, record)
		return nil
	}); err != nil {
		panic(err)
	}
	if err := pkg.VisitUnsupported(func(record semantic.Unsupported) error {
		input.Unsupported = append(input.Unsupported, record)
		return nil
	}); err != nil {
		panic(err)
	}
	return input
}

func semanticDefinitions(
	pkg semantic.Package,
) []semantic.DefinitionSemantics {
	var records []semantic.DefinitionSemantics
	if err := pkg.VisitDefinitions(func(
		record semantic.DefinitionSemantics,
	) error {
		records = append(records, record)
		return nil
	}); err != nil {
		panic(err)
	}
	return records
}

func semanticResolutions(
	pkg semantic.Package,
) []semantic.OccurrenceResolution {
	var records []semantic.OccurrenceResolution
	if err := pkg.VisitResolutions(func(
		record semantic.OccurrenceResolution,
	) error {
		records = append(records, record)
		return nil
	}); err != nil {
		panic(err)
	}
	return records
}

func semanticDeclarations(pkg semantic.Package) []semantic.Declaration {
	var records []semantic.Declaration
	if err := pkg.VisitDeclarations(func(
		record semantic.Declaration,
	) error {
		records = append(records, record)
		return nil
	}); err != nil {
		panic(err)
	}
	return records
}

func semanticBindings(pkg semantic.Package) []semantic.Binding {
	var records []semantic.Binding
	if err := pkg.VisitBindings(func(record semantic.Binding) error {
		records = append(records, record)
		return nil
	}); err != nil {
		panic(err)
	}
	return records
}

func semanticTypes(pkg semantic.Package) []semantic.Type {
	var records []semantic.Type
	if err := pkg.VisitTypes(func(record semantic.Type) error {
		records = append(records, record)
		return nil
	}); err != nil {
		panic(err)
	}
	return records
}

func semanticOperations(pkg semantic.Package) []semantic.Operation {
	var records []semantic.Operation
	if err := pkg.VisitOperations(func(record semantic.Operation) error {
		records = append(records, record)
		return nil
	}); err != nil {
		panic(err)
	}
	return records
}

func expectIdentity(
	value interface{ String() string },
) ([]string, error) {
	return []string{value.String()}, nil
}

func rejectAt(
	value interface{ String() string },
	err error,
) ([]string, error) {
	return []string{value.String()}, err
}

func mutateOperation(
	t *testing.T,
	harness *stage2MutationHarness,
	importPath string,
	match func(semantic.Operation) bool,
	mutate func(*semantic.OperationSpec) error,
) {
	t.Helper()
	harness.requireRejected(
		t, importPath, nil,
		func(input *semantic.PackageInput) ([]string, error) {
			index, record := sourceOperationBy(t, *input, match)
			spec := record.Spec()
			if err := mutate(&spec); err != nil {
				return rejectAt(record.ID(), err)
			}
			mutated, err := semantic.NewOperation(spec)
			if err != nil {
				return rejectAt(record.ID(), err)
			}
			input.Operations[index] = mutated
			return expectIdentity(record.ID())
		},
	)
}

func mutateBindingResolution(
	t *testing.T,
	harness *stage2MutationHarness,
	importPath string,
	name string,
	choose func([]semantic.Binding) (semantic.Binding, semantic.Binding),
) {
	t.Helper()
	harness.requireRejected(
		t, importPath, nil,
		func(input *semantic.PackageInput) ([]string, error) {
			var matching []semantic.Binding
			for _, binding := range input.Bindings {
				if binding.Name() == name {
					matching = append(matching, binding)
				}
			}
			from, to := choose(matching)
			for index, record := range input.Resolutions {
				if record.Kind() != semantic.ResolutionBinding ||
					record.Binding() != from.ID() {
					continue
				}
				spec := resolutionSpec(record)
				spec.Binding = to.ID()
				mutated, err := semantic.NewOccurrenceResolution(spec)
				if err != nil {
					return rejectAt(record.Occurrence(), err)
				}
				input.Resolutions[index] = mutated
				return expectIdentity(record.Occurrence())
			}
			t.Fatalf("binding %s has no semantic reference", from.ID())
			return nil, nil
		},
	)
}

func mutateDuplicateImplicit(
	t *testing.T,
	harness *stage2MutationHarness,
	importPath string,
	collapse bool,
) {
	t.Helper()
	mutateOperation(
		t, harness, importPath,
		func(operation semantic.Operation) bool {
			effects := operation.Spec().Implicit
			for left := range effects {
				for right := left + 1; right < len(effects); right++ {
					if effects[left].Kind() == effects[right].Kind() {
						return true
					}
				}
			}
			return false
		},
		func(spec *semantic.OperationSpec) error {
			for left := range spec.Implicit {
				for right := left + 1; right < len(spec.Implicit); right++ {
					if spec.Implicit[left].Kind() !=
						spec.Implicit[right].Kind() {
						continue
					}
					if !collapse {
						spec.Implicit = slices.Delete(
							spec.Implicit, right, right+1,
						)
						return nil
					}
					first := spec.Implicit[left]
					mutated, err := semantic.NewImplicitOperation(
						first.Kind(),
						first.Site(),
						first.Ordinal(),
						spec.Implicit[right].Source(),
						spec.Implicit[right].Target(),
					)
					if err != nil {
						return err
					}
					spec.Implicit[right] = mutated
					return nil
				}
			}
			t.Fatal("same-kind implicit effects are absent")
			return nil
		},
	)
}

func resolutionSpec(
	record semantic.OccurrenceResolution,
) semantic.ResolutionSpec {
	return semantic.ResolutionSpec{
		Occurrence:  record.Occurrence(),
		Owner:       record.Owner(),
		Syntax:      record.Syntax(),
		Role:        record.Role(),
		Variant:     record.Variant(),
		Domain:      record.Domain(),
		Kind:        record.Kind(),
		Structural:  record.Structural(),
		Component:   record.Component(),
		Definition:  record.Definition(),
		Declaration: record.Declaration(),
		Binding:     record.Binding(),
		Type:        record.Type(),
		Operation:   record.Operation(),
		Unsupported: record.Unsupported(),
	}
}

func declarationNamed(
	t *testing.T,
	input semantic.PackageInput,
	name string,
) semantic.Declaration {
	t.Helper()
	for _, record := range input.Declarations {
		if record.Name() == name {
			return record
		}
	}
	t.Fatalf("semantic declaration %s is absent", name)
	return semantic.Declaration{}
}

func resolutionForDeclaration(
	t *testing.T,
	input semantic.PackageInput,
	declaration identity.SemanticDeclarationID,
) (int, semantic.OccurrenceResolution) {
	t.Helper()
	for index, record := range input.Resolutions {
		if record.Kind() == semantic.ResolutionDeclaration &&
			record.Declaration() == declaration {
			return index, record
		}
	}
	t.Fatalf("declaration %s has no semantic reference", declaration)
	return -1, semantic.OccurrenceResolution{}
}

func namedTypeByDeclaration(
	t *testing.T,
	input semantic.PackageInput,
	declaration identity.SemanticDeclarationID,
) semantic.Type {
	t.Helper()
	_, record := namedTypeIndexByDeclaration(t, input, declaration)
	return record
}

func namedTypeIndexByDeclaration(
	t *testing.T,
	input semantic.PackageInput,
	declaration identity.SemanticDeclarationID,
) (int, semantic.Type) {
	t.Helper()
	for index, record := range input.Types {
		if record.Kind() == semantic.TypeNamed &&
			record.Spec().Declaration == declaration {
			return index, record
		}
	}
	t.Fatalf("named semantic type for %s is absent", declaration)
	return -1, semantic.Type{}
}
