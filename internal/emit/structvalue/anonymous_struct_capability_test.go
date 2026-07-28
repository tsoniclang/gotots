package structvalue_test

import (
	"testing"

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
	assertStaticOperationSequence(t, support, class.Name().Text(), nil)
}
