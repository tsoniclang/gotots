package compiler

import (
	"slices"
	"testing"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
	"github.com/tsoniclang/gotots/internal/language/semantic"
)

func TestStage2MutationMatrixRejectsSemanticCorruption(t *testing.T) {
	harness := newStage2MutationHarness(t)
	const application = "example.com/stage2"

	t.Run("missing-definition-semantics", func(t *testing.T) {
		harness.requireRejected(
			t, application, nil,
			func(input *semantic.PackageInput) ([]string, error) {
				index, record := definitionByName(t, *input, "Matrix")
				input.Definitions = slices.Delete(
					input.Definitions, index, index+1,
				)
				return expectIdentity(record.Definition())
			},
		)
	})

	t.Run("duplicate-occurrence-resolution", func(t *testing.T) {
		harness.requireRejected(
			t, application, nil,
			func(input *semantic.PackageInput) ([]string, error) {
				record := input.Resolutions[0]
				input.Resolutions = append(input.Resolutions, record)
				return expectIdentity(record.Occurrence())
			},
		)
	})

	t.Run("parent-assigned-arity", func(t *testing.T) {
		mutateOperation(
			t, harness, application,
			func(operation semantic.Operation) bool {
				return operation.Kind() == semantic.OperationMapLookup &&
					operation.Spec().Arity ==
						semantic.ResultArityCommaOk
			},
			func(spec *semantic.OperationSpec) error {
				spec.Arity = semantic.ResultArityOne
				spec.HasOk = false
				return nil
			},
		)
	})

	t.Run("ordered-operands", func(t *testing.T) {
		mutateOperation(
			t, harness, application,
			func(operation semantic.Operation) bool {
				return len(operation.Spec().Operands) >= 2
			},
			func(spec *semantic.OperationSpec) error {
				spec.Operands[0], spec.Operands[1] =
					spec.Operands[1], spec.Operands[0]
				return nil
			},
		)
	})

	t.Run("shadowed-bindings", func(t *testing.T) {
		mutateBindingResolution(
			t, harness, application, "value",
			func(records []semantic.Binding) (
				semantic.Binding,
				semantic.Binding,
			) {
				for _, left := range records {
					for _, right := range records {
						if left.ID() != right.ID() &&
							left.Definition() != right.Definition() {
							return left, right
						}
					}
				}
				t.Fatal("two definition-distinct value bindings are absent")
				return semantic.Binding{}, semantic.Binding{}
			},
		)
	})

	t.Run("alias-versus-defined-type", func(t *testing.T) {
		harness.requireRejected(
			t, application, nil,
			func(input *semantic.PackageInput) ([]string, error) {
				alias := declarationNamed(t, *input, "Alias")
				defined := declarationNamed(t, *input, "Number")
				index, record := resolutionForDeclaration(
					t, *input, alias.ID(),
				)
				spec := resolutionSpec(record)
				spec.Declaration = defined.ID()
				mutated, err := semantic.NewOccurrenceResolution(spec)
				if err != nil {
					return rejectAt(record.Occurrence(), err)
				}
				input.Resolutions[index] = mutated
				return expectIdentity(record.Occurrence())
			},
		)
	})

	t.Run("file-versus-definition-scope", func(t *testing.T) {
		harness.requireRejected(
			t, application, nil,
			func(input *semantic.PackageInput) ([]string, error) {
				var fileScoped, definitionScoped semantic.Binding
				for _, binding := range input.Bindings {
					if binding.Name() != "unsafe" {
						continue
					}
					if binding.Definition().IsZero() {
						fileScoped = binding
					} else {
						definitionScoped = binding
					}
				}
				if fileScoped.ID().IsZero() ||
					definitionScoped.ID().IsZero() {
					t.Fatal("file-scoped and definition-scoped unsafe bindings are absent")
				}
				for index, operation := range input.Operations {
					spec := operation.Spec()
					if spec.Object.Kind() !=
						semantic.ObjectReferenceBinding ||
						spec.Object.Binding() != fileScoped.ID() {
						continue
					}
					reference, err := semantic.BindingReference(
						definitionScoped.ID(),
					)
					if err != nil {
						return rejectAt(operation.ID(), err)
					}
					spec.Object = reference
					mutated, err := semantic.NewOperation(spec)
					if err != nil {
						return rejectAt(operation.ID(), err)
					}
					input.Operations[index] = mutated
					return expectIdentity(operation.ID())
				}
				t.Fatalf(
					"file-scoped binding %s has no operation reference",
					fileScoped.ID(),
				)
				return nil, nil
			},
		)
	})

	t.Run("universal-versus-empty-type-set", func(t *testing.T) {
		harness.requireRejected(
			t, application, nil,
			func(input *semantic.PackageInput) ([]string, error) {
				universal := namedTypeByDeclaration(
					t, *input, declarationNamed(
						t, *input, "Universal",
					).ID(),
				)
				emptyIndex, empty := namedTypeIndexByDeclaration(
					t, *input, declarationNamed(
						t, *input, "Empty",
					).ID(),
				)
				spec := empty.Spec()
				spec.Underlying = universal.Spec().Underlying
				mutated, err := semantic.NewType(spec)
				if err != nil {
					return rejectAt(empty.ID(), err)
				}
				if mutated.ID() != empty.ID() {
					t.Fatalf(
						"nominal identity changed: %s -> %s",
						empty.ID(), mutated.ID(),
					)
				}
				input.Types[emptyIndex] = mutated
				return expectIdentity(empty.ID())
			},
		)
	})

	t.Run("method-selection-index", func(t *testing.T) {
		mutateOperation(
			t, harness, application,
			func(operation semantic.Operation) bool {
				selection := operation.Spec().Selection
				return operation.Kind() ==
					semantic.OperationFieldSelect &&
					len(selection.Index()) > 1
			},
			func(spec *semantic.OperationSpec) error {
				selection := spec.Selection
				index := selection.Index()
				index[len(index)-1]++
				mutated, err := semantic.NewSelection(
					selection.Kind(),
					selection.Receiver(),
					selection.Object(),
					index,
					selection.Indirect(),
				)
				if err != nil {
					return err
				}
				spec.Selection = mutated
				return nil
			},
		)
	})

	t.Run("initializer-declaration-order", func(t *testing.T) {
		harness.requireRejected(
			t, application, nil,
			func(input *semantic.PackageInput) ([]string, error) {
				for index, record := range input.Definitions {
					spec := record.Spec()
					if record.Form() !=
						semantic.DefinitionFormInitializer ||
						len(spec.Declarations) != 2 {
						continue
					}
					spec.Declarations[0], spec.Declarations[1] =
						spec.Declarations[1], spec.Declarations[0]
					mutated, err := semantic.NewDefinitionSemantics(spec)
					if err != nil {
						return rejectAt(record.Definition(), err)
					}
					input.Definitions[index] = mutated
					return expectIdentity(record.Definition())
				}
				t.Fatal("two-declaration initializer is absent")
				return nil, nil
			},
		)
	})

	t.Run("branch-control-target", func(t *testing.T) {
		mutateOperation(
			t, harness, application,
			func(operation semantic.Operation) bool {
				spec := operation.Spec()
				return operation.Kind() == semantic.OperationBranch &&
					!spec.Label.IsZero() &&
					!spec.ControlTarget.IsZero()
			},
			func(spec *semantic.OperationSpec) error {
				spec.ControlTarget = spec.ID
				return nil
			},
		)
	})

	t.Run("compile-time-coverage-target", func(t *testing.T) {
		harness.requireRejected(
			t, application, nil,
			func(input *semantic.PackageInput) ([]string, error) {
				for index, record := range input.Resolutions {
					if record.Kind() !=
						semantic.ResolutionStructuralOnly ||
						record.Structural().Disposition() !=
							semantic.StructuralCompileTimeExpression {
						continue
					}
					spec := resolutionSpec(record)
					spec.Structural = semantic.StructuralEvidence{}
					mutated, err := semantic.NewOccurrenceResolution(spec)
					if err != nil {
						return rejectAt(record.Occurrence(), err)
					}
					input.Resolutions[index] = mutated
					return expectIdentity(record.Occurrence())
				}
				t.Fatal("compile-time structural resolution is absent")
				return nil, nil
			},
		)
	})

	t.Run("predeclared-duplicate-in-ordinary-package", func(t *testing.T) {
		builtin, _ := harness.packageInput(t, "builtin")
		harness.requireRejected(
			t, application, nil,
			func(input *semantic.PackageInput) ([]string, error) {
				record := builtin.Declarations[0]
				input.Declarations = append(
					input.Declarations, record,
				)
				return expectIdentity(record.ID())
			},
		)
	})

	t.Run("operation-in-boundary-domain", func(t *testing.T) {
		harness.requireRejected(
			t, application, nil,
			func(input *semantic.PackageInput) ([]string, error) {
				for index, record := range input.Resolutions {
					if record.Kind() != semantic.ResolutionOperation {
						continue
					}
					spec := resolutionSpec(record)
					spec.Domain = catalog.ResolutionDomainBoundary
					mutated, err := semantic.NewOccurrenceResolution(spec)
					if err != nil {
						return rejectAt(record.Occurrence(), err)
					}
					input.Resolutions[index] = mutated
					return expectIdentity(record.Occurrence())
				}
				t.Fatal("operation resolution is absent")
				return nil, nil
			},
		)
	})

	t.Run("fabricated-package-init-occurrence", func(t *testing.T) {
		harness.requireRejected(
			t, application, nil,
			func(input *semantic.PackageInput) ([]string, error) {
				_, operation := operationByKind(
					t, *input,
					semantic.OperationPackageInitialization,
				)
				source := input.Resolutions[0]
				_, err := semantic.NewOccurrenceResolution(
					semantic.ResolutionSpec{
						Occurrence: source.Occurrence(),
						Owner:      source.Owner(),
						Syntax:     source.Syntax(),
						Role:       source.Role(),
						Variant:    source.Variant(),
						Domain:     source.Domain(),
						Kind:       semantic.ResolutionOperation,
						Operation:  operation.ID(),
					},
				)
				if err == nil {
					t.Fatal("package initialization accepted a fabricated occurrence")
				}
				return rejectAt(source.Occurrence(), err)
			},
		)
	})

	t.Run("collapsed-same-kind-implicit-effects", func(t *testing.T) {
		mutateDuplicateImplicit(
			t, harness, application, true,
		)
	})

	t.Run("missing-required-implicit-effect", func(t *testing.T) {
		mutateDuplicateImplicit(
			t, harness, application, false,
		)
	})
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
