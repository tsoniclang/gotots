package unsafepointer

import (
	"testing"

	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestBuildCreatesUnconstructableNominalDeclaration(t *testing.T) {
	class := Build(tsgo.NewFactory(), "GoUnsafePointer", "GoPointer")
	if class.Name().Text() != "GoUnsafePointer" ||
		len(class.Modifiers()) != 1 ||
		class.Modifiers()[0].Kind() != tsgo.SyntaxKindExportKeyword ||
		len(class.Members()) != 4 {
		t.Fatalf("unsafe-pointer class has unexpected shape")
	}
	property, ok := class.Members()[0].(tsgo.PropertyDeclaration)
	if !ok ||
		property.Name().(tsgo.Identifier).Text() != brandName ||
		len(property.Modifiers()) != 3 {
		t.Fatal("unsafe-pointer class lacks its private nominal brand")
	}
	constructor, ok := class.Members()[1].(tsgo.ConstructorDeclaration)
	if !ok ||
		len(constructor.Modifiers()) != 1 ||
		constructor.Modifiers()[0].Kind() != tsgo.SyntaxKindPrivateKeyword ||
		constructor.Body() == nil {
		t.Fatal("unsafe-pointer declaration is constructable")
	}
	for index, name := range []string{FromName, ToName} {
		method, ok := class.Members()[index+2].(tsgo.MethodDeclaration)
		if !ok ||
			method.Name().(tsgo.Identifier).Text() != name ||
			len(method.Modifiers()) != 1 ||
			method.Modifiers()[0].Kind() != tsgo.SyntaxKindStaticKeyword ||
			len(method.TypeParameters()) != 2 ||
			method.Body() == nil {
			t.Fatalf("unsafe-pointer conversion method %q is invalid", name)
		}
	}
}
