package naming

import (
	"go/types"
	"testing"
)

func TestPortableIdentifierEscapesNonASCIIWithoutChangingASCII(t *testing.T) {
	for source, expected := range map[string]string{
		"value":     "value",
		"π":         "__u3c0_",
		"Δelta":     "__u394_elta",
		"class":     "__go_class",
		"await":     "__go_await",
		"arguments": "__go_arguments",
		"__proto__": "__go___proto__",
	} {
		if actual := portableIdentifier(source); actual != expected {
			t.Fatalf("portableIdentifier(%q) = %q, want %q", source, actual, expected)
		}
	}
}

func TestPackageQualifiersAreGloballyUniqueAfterPortableEscaping(t *testing.T) {
	packages := []*types.Package{
		types.NewPackage("example.com/first", "π"),
		types.NewPackage("example.com/second", "__u3c0_"),
		types.NewPackage("example.com/third", "π"),
	}
	registry := NewRegistry()
	if err := registry.indexPackageQualifiers(packages); err != nil {
		t.Fatal(err)
	}
	seen := make(map[string]*types.Package)
	for _, sourcePackage := range packages {
		qualifier := registry.importQualifierByPackage[sourcePackage]
		if qualifier == "" {
			t.Fatalf("package %s has no qualifier", sourcePackage.Path())
		}
		if previous := seen[qualifier]; previous != nil {
			t.Fatalf(
				"packages %s and %s share qualifier %q",
				previous.Path(),
				sourcePackage.Path(),
				qualifier,
			)
		}
		seen[qualifier] = sourcePackage
	}

	reversed := []*types.Package{packages[2], packages[1], packages[0]}
	second := NewRegistry()
	if err := second.indexPackageQualifiers(reversed); err != nil {
		t.Fatal(err)
	}
	for _, sourcePackage := range packages {
		if registry.importQualifierByPackage[sourcePackage] !=
			second.importQualifierByPackage[sourcePackage] {
			t.Fatalf("package %s qualifier depends on input order", sourcePackage.Path())
		}
	}
}
