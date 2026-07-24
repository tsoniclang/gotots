package compiler

import (
	"testing"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
	"github.com/tsoniclang/gotots/internal/language/semantic"
	"github.com/tsoniclang/gotots/internal/scope/contract"
	"github.com/tsoniclang/gotots/internal/source"
)

func TestLocalStructTagRemainsCompileTimeStructure(t *testing.T) {
	directory := t.TempDir()
	writeCompilerFile(
		t,
		directory,
		"go.mod",
		"module example.com/localtag\n\ngo 1.26.0\n",
	)
	writeCompilerFile(
		t,
		directory,
		"localtag.go",
		`package localtag

func Encode() any {
	type record struct {
		Value string `+"`json:\"value,omitempty\"`"+`
	}
	return record{Value: "ok"}
}
`,
	)
	inspection, err := inspectConstructsForTest(t, source.Request{
		Dir: directory, Patterns: []string{"."},
		ProviderContract: contract.DefaultID,
	})
	if err != nil {
		t.Fatal(err)
	}
	var fieldTags int
	if err := inspection.Semantic().VisitPackages(
		func(pkg semantic.Package) error {
			for _, resolution := range semanticResolutions(pkg) {
				if resolution.Role() != catalog.RoleFieldTag {
					continue
				}
				fieldTags++
				if resolution.Kind() != semantic.ResolutionStructuralOnly {
					t.Fatalf(
						"field tag resolution = %s",
						resolution.Kind(),
					)
				}
				for _, operation := range semanticOperations(pkg) {
					if operation.Occurrence() == resolution.Occurrence() {
						t.Fatal("field tag produced a runtime operation")
					}
				}
			}
			return nil
		},
	); err != nil {
		t.Fatal(err)
	}
	if fieldTags != 1 {
		t.Fatalf("field-tag resolutions = %d, want 1", fieldTags)
	}
}

func TestCompileTimeExpressionContextOverridesExecutableRegion(
	t *testing.T,
) {
	directory := t.TempDir()
	writeCompilerFile(
		t,
		directory,
		"go.mod",
		"module example.com/compiletime\n\ngo 1.26.0\n",
	)
	writeCompilerFile(
		t,
		directory,
		"compiletime.go",
		`package compiletime

const base = 2

var names = [base + 1]string{}

func Value(input int) int {
	const local = base + 1
	_ = [local + 1]int{}
	return input + 1
}
`,
	)
	inspection, err := inspectConstructsForTest(t, source.Request{
		Dir: directory, Patterns: []string{"."},
		ProviderContract: contract.DefaultID,
	})
	if err != nil {
		t.Fatal(err)
	}
	var arrayLengths, constInitializers, runtimeBinary int
	if err := inspection.Semantic().VisitPackages(
		func(pkg semantic.Package) error {
			operations := map[identity.OccurrenceID]semantic.Operation{}
			for _, operation := range semanticOperations(pkg) {
				operations[operation.Occurrence()] = operation
			}
			for _, resolution := range semanticResolutions(pkg) {
				if resolution.Syntax() != catalog.KindBinaryExpr {
					continue
				}
				switch resolution.Role() {
				case catalog.RoleArrayLength:
					arrayLengths++
					assertCompileTimeResolution(
						t, resolution, operations,
					)
				case catalog.RoleInitializerValue:
					constInitializers++
					assertCompileTimeResolution(
						t, resolution, operations,
					)
				case catalog.RoleReturnValue:
					runtimeBinary++
					if resolution.Kind() !=
						semantic.ResolutionOperation {
						t.Fatalf(
							"runtime binary resolution = %s",
							resolution.Kind(),
						)
					}
				}
			}
			return nil
		},
	); err != nil {
		t.Fatal(err)
	}
	if arrayLengths != 2 ||
		constInitializers != 1 ||
		runtimeBinary != 1 {
		t.Fatalf(
			"binary contexts array/const/runtime = %d/%d/%d",
			arrayLengths,
			constInitializers,
			runtimeBinary,
		)
	}
}

func assertCompileTimeResolution(
	t *testing.T,
	resolution semantic.OccurrenceResolution,
	operations map[identity.OccurrenceID]semantic.Operation,
) {
	t.Helper()
	if resolution.Kind() != semantic.ResolutionStructuralOnly ||
		resolution.Structural().Disposition() !=
			semantic.StructuralCompileTimeExpression {
		t.Fatalf(
			"compile-time resolution %s = %s/%v",
			resolution.Occurrence(),
			resolution.Kind(),
			resolution.Structural().Disposition(),
		)
	}
	if _, present := operations[resolution.Occurrence()]; present {
		t.Fatalf(
			"compile-time occurrence %s produced a runtime operation",
			resolution.Occurrence(),
		)
	}
}
