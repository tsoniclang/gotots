package compiler

import (
	"testing"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
	"github.com/tsoniclang/gotots/internal/language/semantic"
	"github.com/tsoniclang/gotots/internal/scope/contract"
	"github.com/tsoniclang/gotots/internal/source"
)

func TestStage2TypeSwitchBindingsAreOwnedByCaseScopes(t *testing.T) {
	directory := t.TempDir()
	writeCompilerFile(
		t,
		directory,
		"go.mod",
		"module example.com/typeswitch\n\ngo 1.26.0\n",
	)
	writeCompilerFile(t, directory, "typeswitch.go", `package typeswitch

func Empty(input any) {
	switch input.(type) {
	}
}

func One(input any) int {
	switch value := input.(type) {
	case int:
		return value
	}
	return 0
}

func Ordinary(input any) any {
	value := input
	return value
}

func LocalType() {
	type score struct {
		high int64
		low  int64
	}
	add := func(left, right score) score {
		return score{
			high: left.high + right.high,
			low:  left.low + right.low,
		}
	}
	_ = add
}

func Select(input any) int {
	switch value := input.(type) {
	case int:
		return value
	case string:
		return len(value)
	default:
		_ = value
		return 0
	}
}
`)
	inspection, err := inspectConstructsForTest(t, source.Request{
		Dir: directory, Patterns: []string{"."},
		ProviderContract: contract.DefaultID,
	})
	if err != nil {
		t.Fatal(err)
	}
	pkg := semanticPackageByImportPath(
		t, inspection.Semantic(), "example.com/typeswitch",
	)

	definitionNames := map[identity.DefinitionID]string{}
	for _, definition := range semanticDefinitions(pkg) {
		definitionNames[definition.Definition()] = definition.Spec().Name
	}
	bindings := map[identity.SemanticBindingID]semantic.Binding{}
	types := map[identity.SemanticTypeID]bool{}
	counts := map[string]int{}
	for _, binding := range semanticBindings(pkg) {
		if binding.Role() != identity.SemanticBindingTypeSwitch {
			continue
		}
		if binding.Name() != "value" ||
			!binding.Source().IsZero() ||
			binding.ID().Owner().KindID() !=
				uint16(catalog.KindCaseClause) {
			t.Fatalf(
				"type-switch binding has noncanonical identity: id=%s name=%q source=%s",
				binding.ID(), binding.Name(), binding.Source(),
			)
		}
		bindings[binding.ID()] = binding
		types[binding.Type()] = true
		counts[definitionNames[binding.Definition()]]++
	}
	if len(bindings) != 4 ||
		len(types) != 3 ||
		counts["Empty"] != 0 ||
		counts["One"] != 1 ||
		counts["Select"] != 3 {
		t.Fatalf(
			"type-switch bindings=%d types=%d counts=%v",
			len(bindings), len(types), counts,
		)
	}

	uses := map[identity.SemanticBindingID]int{}
	anchors := 0
	guards := 0
	for _, resolution := range semanticResolutions(pkg) {
		if resolution.Kind() == semantic.ResolutionStructuralOnly &&
			resolution.Structural().Disposition() ==
				semantic.StructuralTypeSwitchBindingAnchor {
			anchors++
		}
	}
	if anchors != 2 {
		t.Fatalf("type-switch anchors=%d, want 2", anchors)
	}
	var resultType identity.SemanticTypeID
	for _, binding := range bindings {
		resultType = binding.Type()
		break
	}
	for _, operation := range semanticOperations(pkg) {
		spec := operation.Spec()
		if spec.Object.Kind() ==
			semantic.ObjectReferenceBinding {
			binding := spec.Object.Binding()
			if _, present := bindings[binding]; present {
				uses[binding]++
			}
		}
		if operation.Kind() != semantic.OperationTypeAssert ||
			operation.Variant() != catalog.VariantTypeSwitchGuard {
			continue
		}
		guards++
		if spec.Mode != semantic.ValueModeNone ||
			spec.Arity != semantic.ResultArityZero ||
			!spec.ResultType.IsZero() ||
			len(spec.Operands) != 1 {
			t.Fatalf(
				"type-switch guard has noncanonical value shape: %+v",
				spec,
			)
		}
		spec.Mode = semantic.ValueModeValue
		spec.Arity = semantic.ResultArityOne
		spec.ResultType = resultType
		if _, err := semantic.NewOperation(spec); err == nil {
			t.Fatal("type-switch guard accepted a fabricated result")
		}
	}
	if guards != 3 {
		t.Fatalf("type-switch guard operations=%d, want 3", guards)
	}
	if len(uses) != len(bindings) {
		t.Fatalf(
			"type-switch use bindings=%d, want %d",
			len(uses), len(bindings),
		)
	}
	for binding := range bindings {
		if uses[binding] != 1 {
			t.Fatalf(
				"type-switch binding %s has %d uses, want 1",
				binding, uses[binding],
			)
		}
	}
}
