package structvalue_test

import (
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/output"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestAnonymousStructCapabilitiesAreDemandDriven(t *testing.T) {
	emission, err := compileTemporaryStructProgram(t, `package boundary

func Value(value struct{ Field int32 }) int32 {
	return value.Field
}
`)
	if err != nil {
		t.Fatal(err)
	}
	support := anonymousStructSupport(t, emission)
	var class tsgo.ClassDeclaration
	for _, statement := range support.Statements() {
		candidate, ok := statement.(tsgo.ClassDeclaration)
		if ok {
			class = candidate
			break
		}
	}
	if class == nil {
		t.Fatal("anonymous-struct definition capability emitted no class")
	}
	assertStaticOperationSequence(
		t,
		support,
		class.Name().Text(),
		nil,
	)
}

func TestEmptyAnonymousStructUsesOneRuntimeOwner(t *testing.T) {
	emission, err := compileTemporaryStructProgram(t, `package boundary

func Empty(value struct{}) (struct{}, bool) {
	copy := value
	return copy, copy == struct{}{}
}
`)
	if err != nil {
		t.Fatal(err)
	}
	foundRuntime := false
	for _, file := range emission.Files() {
		if file.OutputPath() == output.AnonymousStructSupportPath {
			t.Fatal("empty struct created an anonymous support artifact")
		}
		if file.OutputPath() != "runtime/struct.ts" {
			continue
		}
		foundRuntime = true
		classes := make([]tsgo.ClassDeclaration, 0, 1)
		for _, statement := range file.SourceFile().Statements() {
			class, ok := statement.(tsgo.ClassDeclaration)
			if ok {
				classes = append(classes, class)
			}
		}
		if len(classes) != 1 || classes[0].Name().Text() != "GoEmptyStruct" {
			t.Fatalf("empty struct runtime classes = %d, want sole GoEmptyStruct", len(classes))
		}
	}
	if !foundRuntime {
		t.Fatal("GoEmptyStruct runtime owner is absent")
	}
	source := structTargetSource(t, emission)
	encoded, err := tsgo.EncodeSourceFile(source)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "$goStruct$") {
		t.Fatal("empty struct source retained a generated anonymous-struct identity")
	}
}
