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

	t.Run("blank-function-declaration-cardinality", func(t *testing.T) {
		harness.requireRejected(
			t, application, nil,
			func(input *semantic.PackageInput) ([]string, error) {
				index, record := definitionByName(t, *input, "_")
				spec := record.Spec()
				spec.Name = "FabricatedBinding"
				mutated, err := semantic.NewDefinitionSemantics(spec)
				if err != nil {
					return rejectAt(record.Definition(), err)
				}
				input.Definitions[index] = mutated
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

	t.Run("explicit-implicit-binding-order", func(t *testing.T) {
		harness.requireRejected(
			t, application, nil,
			func(input *semantic.PackageInput) ([]string, error) {
				for index, binding := range input.Bindings {
					if binding.Name() != "other" ||
						binding.Role() != identity.SemanticBindingImport {
						continue
					}
					if binding.ID().Ordinal() == 0 {
						t.Fatal("explicit import does not follow the implicit import")
					}
					id, err := identity.NewSemanticBindingID(
						binding.ID().Owner(),
						binding.ID().Declaration(),
						binding.Role(),
						0,
					)
					if err != nil {
						return rejectAt(binding.ID(), err)
					}
					mutated, err := semantic.NewBinding(
						id,
						binding.Package(),
						binding.Definition(),
						binding.Role(),
						binding.Name(),
						binding.Type(),
						binding.Source(),
						binding.CapturedBy(),
						binding.Authority(),
					)
					if err != nil {
						return rejectAt(binding.ID(), err)
					}
					input.Bindings[index] = mutated
					for resolutionIndex, resolution := range input.Resolutions {
						if resolution.Kind() !=
							semantic.ResolutionBinding ||
							resolution.Binding() != binding.ID() {
							continue
						}
						spec := resolutionSpec(resolution)
						spec.Binding = mutated.ID()
						replacement, err :=
							semantic.NewOccurrenceResolution(spec)
						if err != nil {
							return rejectAt(
								resolution.Occurrence(),
								err,
							)
						}
						input.Resolutions[resolutionIndex] =
							replacement
					}
					for operationIndex, operation := range input.Operations {
						spec := operation.Spec()
						if spec.Object.Kind() !=
							semantic.ObjectReferenceBinding ||
							spec.Object.Binding() != binding.ID() {
							continue
						}
						reference, err := semantic.BindingReference(
							mutated.ID(),
						)
						if err != nil {
							return rejectAt(operation.ID(), err)
						}
						spec.Object = reference
						replacement, err :=
							semantic.NewOperation(spec)
						if err != nil {
							return rejectAt(operation.ID(), err)
						}
						input.Operations[operationIndex] =
							replacement
					}
					return []string{
						binding.ID().String(),
						mutated.ID().String(),
					}, nil
				}
				t.Fatal("explicit import binding other is absent")
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

	t.Run("inferred-array-ellipsis-type", func(t *testing.T) {
		harness.requireRejected(
			t, application, nil,
			func(input *semantic.PackageInput) ([]string, error) {
				number := namedTypeByDeclaration(
					t,
					*input,
					declarationNamed(t, *input, "Number").ID(),
				)
				for index, resolution := range input.Resolutions {
					if resolution.Syntax() != catalog.KindEllipsis ||
						resolution.Role() != catalog.RoleArrayLength ||
						resolution.Kind() != semantic.ResolutionType {
						continue
					}
					spec := resolutionSpec(resolution)
					spec.Type = number.ID()
					mutated, err :=
						semantic.NewOccurrenceResolution(spec)
					if err != nil {
						return rejectAt(
							resolution.Occurrence(),
							err,
						)
					}
					input.Resolutions[index] = mutated
					return expectIdentity(resolution.Occurrence())
				}
				t.Fatal("inferred array ellipsis resolution is absent")
				return nil, nil
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
