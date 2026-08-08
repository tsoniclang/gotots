package storagefacet_test

import (
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/runtime/storagefacet"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestStorageFacetRuntimeOwnsTwoClosedAssociatedTypeFamilies(t *testing.T) {
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

func identifierText(name tsgo.EntityName) string {
	identifier, _ := name.(tsgo.Identifier)
	return identifier.Text()
}
