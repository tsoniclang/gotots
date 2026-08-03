package storagefacet_test

import (
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/runtime/storagefacet"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestStorageFacetRuntimeOwnsThreeClosedAssociatedTypes(t *testing.T) {
	factory := tsgo.NewFactory()
	for _, testCase := range []struct {
		symbol api.RuntimeSymbol
		kind   string
	}{
		{api.RuntimeStorageTypeToken, "token"},
		{api.RuntimeStoredValue, "contract"},
		{api.RuntimeStorageType, "projection"},
		{api.RuntimeContainerStorageToken, "token"},
		{api.RuntimeContainerStoredValue, "contract"},
		{api.RuntimeContainerStorageType, "projection"},
		{api.RuntimePointerTypeToken, "token"},
		{api.RuntimePointerRepresentedValue, "contract"},
		{api.RuntimePointerType, "projection"},
	} {
		statement, err := storagefacet.Build(factory, testCase.symbol)
		if err != nil {
			t.Fatal(err)
		}
		switch testCase.kind {
		case "token":
			if _, ok := statement.(tsgo.VariableStatement); !ok {
				t.Fatalf("runtime symbol %d = %T, want token", testCase.symbol, statement)
			}
		case "contract":
			if _, ok := statement.(tsgo.InterfaceDeclaration); !ok {
				t.Fatalf("runtime symbol %d = %T, want contract", testCase.symbol, statement)
			}
		case "projection":
			if _, ok := statement.(tsgo.TypeAliasDeclaration); !ok {
				t.Fatalf("runtime symbol %d = %T, want projection", testCase.symbol, statement)
			}
		}
	}
}

func TestPointerProjectionFallsBackToCanonicalLogicalCarrier(t *testing.T) {
	statement, err := storagefacet.Build(
		tsgo.NewFactory(),
		api.RuntimePointerType,
	)
	if err != nil {
		t.Fatal(err)
	}
	alias := statement.(tsgo.TypeAliasDeclaration)
	if alias.Name().Text() != "GoPointerType" || len(alias.TypeParameters()) != 1 {
		t.Fatalf("pointer projection declaration = %#v", alias)
	}
	conditional, ok := alias.Type().(tsgo.ConditionalTypeNode)
	if !ok {
		t.Fatalf("pointer projection = %T, want conditional type", alias.Type())
	}
	check, ok := conditional.CheckType().(tsgo.TupleTypeNode)
	if !ok || len(check.Elements()) != 1 ||
		typeReferenceName(check.Elements()[0]) != "T" {
		t.Fatalf(
			"pointer projection check = %#v, want non-distributive [T]",
			conditional.CheckType(),
		)
	}
	extends, ok := conditional.ExtendsType().(tsgo.TupleTypeNode)
	if !ok || len(extends.Elements()) != 1 {
		t.Fatalf(
			"pointer projection extends = %#v, want one-element tuple",
			conditional.ExtendsType(),
		)
	}
	contract, ok := extends.Elements()[0].(tsgo.TypeReferenceNode)
	if !ok || identifierText(contract.TypeName()) != "GoPointerRepresentedValue" ||
		len(contract.TypeArguments()) != 1 {
		t.Fatalf("pointer projection contract = %#v", conditional.ExtendsType())
	}
	if _, ok := contract.TypeArguments()[0].(tsgo.InferTypeNode); !ok {
		t.Fatalf("pointer projection inference = %T", contract.TypeArguments()[0])
	}
	fallback, ok := conditional.FalseType().(tsgo.TypeReferenceNode)
	if !ok || identifierText(fallback.TypeName()) != "GoPointer" ||
		len(fallback.TypeArguments()) != 2 ||
		typeReferenceName(fallback.TypeArguments()[0]) != "T" ||
		typeReferenceName(fallback.TypeArguments()[1]) != "T" {
		t.Fatalf("pointer projection fallback = %#v", conditional.FalseType())
	}
}

func identifierText(name tsgo.EntityName) string {
	identifier, _ := name.(tsgo.Identifier)
	return identifier.Text()
}

func typeReferenceName(node tsgo.TypeNode) string {
	reference, _ := node.(tsgo.TypeReferenceNode)
	return identifierText(reference.TypeName())
}
