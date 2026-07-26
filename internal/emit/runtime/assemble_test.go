package runtime

import (
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestArrayRuntimeAssemblyExactJoinsRequestedDefinition(t *testing.T) {
	factory := tsgo.NewFactory()
	definitions, err := Build(
		factory,
		api.RuntimeModuleArray,
		[]api.RuntimeSymbol{api.RuntimeArray},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(definitions) != 1 {
		t.Fatalf("array runtime definitions = %d, want one", len(definitions))
	}
	class, ok := definitions[0].Statement().(tsgo.ClassDeclaration)
	if !ok || class.Name().Text() != "GoArray" {
		t.Fatalf("array runtime statement = %T", definitions[0].Statement())
	}
}

func TestArrayRuntimeAssemblyRejectsMissingDuplicateAndWrongDefinitions(
	t *testing.T,
) {
	factory := tsgo.NewFactory()
	if _, err := Build(
		factory,
		api.RuntimeModuleArray,
		nil,
	); err == nil {
		t.Fatal("missing RuntimeArray definition passed")
	}
	if _, err := Build(
		factory,
		api.RuntimeModuleArray,
		[]api.RuntimeSymbol{api.RuntimeArray, api.RuntimeArray},
	); err == nil {
		t.Fatal("duplicate RuntimeArray definition passed")
	}
	if _, err := Build(
		factory,
		api.RuntimeModuleArray,
		[]api.RuntimeSymbol{api.RuntimePointer},
	); err == nil {
		t.Fatal("wrong-module runtime symbol passed array assembly")
	}
}
