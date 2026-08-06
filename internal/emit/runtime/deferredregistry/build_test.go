package deferredregistry_test

import (
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/runtime/deferredregistry"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestRuntimeRegistryOwnsOneGenericInstanceImplementation(t *testing.T) {
	class := deferredregistry.Build(
		tsgo.NewFactory(),
		"GoDeferredRegistry",
		"GoInterfaceValue",
	)
	if class.Name().Text() != "GoDeferredRegistry" ||
		len(class.TypeParameters()) != 3 ||
		len(class.Members()) != 6 {
		t.Fatalf(
			"registry shape = %s/%d/%d",
			class.Name().Text(),
			len(class.TypeParameters()),
			len(class.Members()),
		)
	}
	constraint := class.TypeParameters()[0].Constraint()
	if constraint == nil ||
		constraint.Kind() != tsgo.SyntaxKindObjectKeyword {
		t.Fatalf("source callable constraint = %T", constraint)
	}
	for _, member := range class.Members() {
		var modifiers []tsgo.ModifierLike
		switch member := member.(type) {
		case tsgo.PropertyDeclaration:
			modifiers = member.Modifiers()
		case tsgo.MethodDeclaration:
			modifiers = member.Modifiers()
		default:
			t.Fatalf("registry member = %T", member)
		}
		for _, modifier := range modifiers {
			if modifier.Kind() == tsgo.SyntaxKindStaticKeyword {
				t.Fatalf("registry member %T is static", member)
			}
		}
	}
}
