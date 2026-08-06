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
		[]string{"$make"},
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
		statements := file.SourceFile().Statements()
		if len(statements) != 1 {
			t.Fatalf("empty struct runtime statements = %d, want one", len(statements))
		}
		class, ok := statements[0].(tsgo.ClassDeclaration)
		if !ok || class.Name().Text() != "GoEmptyStruct" {
			t.Fatalf("empty struct runtime owner = %T", statements[0])
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
	if strings.Contains(string(encoded), "$goStruct_") {
		t.Fatal("empty struct source retained a generated anonymous-struct identity")
	}
}
