package naming

import (
	"go/token"
	"go/types"
	"testing"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
)

type facetOnlyProvider struct{}

func (facetOnlyProvider) Valid() bool { return true }

func (facetOnlyProvider) ToolchainKey() string { return "toolchain" }

func (facetOnlyProvider) Binding(string) (gostdlib.Binding, bool) {
	return gostdlib.Binding{}, false
}

func (facetOnlyProvider) Facet(
	string,
	gostdlib.FacetKind,
	gostdlib.FacetCapability,
) (gostdlib.Facet, bool) {
	return gostdlib.Facet{}, false
}

func (facetOnlyProvider) GenericCallableFacet(
	string,
	string,
) (gostdlib.Facet, bool) {
	return gostdlib.Facet{}, false
}

func TestPrivateProviderDeclarationCanOwnCertifiedFacetWithoutPublicBinding(
	t *testing.T,
) {
	selectedPackage := types.NewPackage("example.com/provider", "provider")
	privateType := types.NewTypeName(
		token.NoPos,
		selectedPackage,
		"privateType",
		types.NewStruct(nil, nil),
	)
	selectedPackage.Scope().Insert(privateType)
	registry := NewRegistry()
	registry.provider = facetOnlyProvider{}
	registry.byObject[privateType] = targetBinding{
		kind: targetBindingMissingProvider,
	}
	names := &File{owner: &Owner{registry: registry}}

	contract, providerOwned, err := names.providerFacetOwner(privateType)
	if err != nil {
		t.Fatal(err)
	}
	if !providerOwned ||
		contract.Identity() != "example.com/provider|kind=2|receiver=|name=privateType" {
		t.Fatalf("facet owner = %#v, provider=%t", contract, providerOwned)
	}
}
