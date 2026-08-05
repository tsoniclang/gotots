package naming

import (
	"errors"
	"go/ast"
	"go/token"
	"go/types"
	"testing"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type facetOnlyProvider struct{}

func (facetOnlyProvider) Valid() bool { return true }

func (facetOnlyProvider) ToolchainKey() string { return "toolchain" }

func (facetOnlyProvider) ProviderModules() []string    { return nil }
func (facetOnlyProvider) ProviderScalarModule() string { return "" }

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

func (facetOnlyProvider) ProviderRepresentation(
	string,
	string,
) (gostdlib.ProviderRepresentation, bool) {
	return gostdlib.ProviderRepresentation{}, false
}

func (facetOnlyProvider) ProviderInterface(
	string,
) (gostdlib.ProviderInterfaceBinding, bool) {
	return gostdlib.ProviderInterfaceBinding{}, false
}

func (facetOnlyProvider) ProviderInterfaceCapabilities(
	string,
) []gostdlib.ProviderInterfaceCapability {
	return nil
}

func (facetOnlyProvider) ProviderCallableProfile(
	string,
	string,
) (gostdlib.ProviderCallableProfile, bool) {
	return gostdlib.ProviderCallableProfile{}, false
}

func (facetOnlyProvider) ProviderCallableProfiles(
	string,
) []gostdlib.ProviderCallableProfile {
	return nil
}

func (facetOnlyProvider) ProviderStatefulProfile(
	string,
	string,
) (gostdlib.ProviderStatefulProfile, bool) {
	return gostdlib.ProviderStatefulProfile{}, false
}

func (facetOnlyProvider) ProviderStatefulProfiles(
	string,
) []gostdlib.ProviderStatefulProfile {
	return nil
}

func TestPredeclaredErrorRequiresLanguageProviderInterfaceCertificate(
	t *testing.T,
) {
	registry := NewRegistry()
	registry.provider = facetOnlyProvider{}
	typeName, ok := types.Universe.Lookup("error").(*types.TypeName)
	if !ok {
		t.Fatal("predeclared error type is absent")
	}
	if _, providerOwned, err := registry.ProviderInterface(typeName); err == nil || !providerOwned {
		t.Fatalf("missing language interface = provider %t, error %v", providerOwned, err)
	}
}

func TestPredeclaredErrorIsLanguageOwnedWithoutProvider(t *testing.T) {
	registry := NewRegistry()
	typeName, ok := types.Universe.Lookup("error").(*types.TypeName)
	if !ok {
		t.Fatal("predeclared error type is absent")
	}
	if _, providerOwned, err := registry.ProviderInterface(typeName); err != nil || providerOwned {
		t.Fatalf("source-only error = provider %t, error %v", providerOwned, err)
	}
}

func TestMissingProviderMethodReportsCanonicalSourceIdentity(t *testing.T) {
	selectedPackage := types.NewPackage("encoding/base64", "base64")
	typeName := types.NewTypeName(
		token.NoPos,
		selectedPackage,
		"Encoding",
		nil,
	)
	named := types.NewNamed(typeName, types.NewStruct(nil, nil), nil)
	receiver := types.NewVar(
		token.NoPos,
		selectedPackage,
		"encoding",
		types.NewPointer(named),
	)
	method := types.NewFunc(
		token.NoPos,
		selectedPackage,
		"AppendEncode",
		types.NewSignatureType(
			receiver,
			nil,
			nil,
			types.NewTuple(),
			types.NewTuple(),
			false,
		),
	)
	named.AddMethod(method)
	registry := NewRegistry()
	registry.byObject[method] = targetBinding{kind: targetBindingMissingProvider}
	names := &File{owner: &Owner{registry: registry}}

	_, err := names.MethodTarget(method)
	var nameError *api.NameError
	if !errors.As(err, &nameError) {
		t.Fatalf("error = %#v, want NameError", err)
	}
	want := "encoding/base64|kind=4|receiver=*encoding/base64.Encoding|name=AppendEncode"
	if nameError.Name != want {
		t.Fatalf("error identity = %q, want %q", nameError.Name, want)
	}
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

func TestLexicalSourceTypeIsNotAProviderFacetOwner(t *testing.T) {
	selectedPackage := types.NewPackage("example.com/application", "application")
	functionScope := types.NewScope(
		selectedPackage.Scope(),
		token.NoPos,
		token.NoPos,
		"function",
	)
	localType := types.NewTypeName(
		token.NoPos,
		selectedPackage,
		"MemberInfo",
		types.NewStruct(nil, nil),
	)
	if existing := functionScope.Insert(localType); existing != nil {
		t.Fatal("local type identity was duplicated")
	}
	names := &File{owner: &Owner{registry: NewRegistry()}}

	_, providerOwned, err := names.providerFacetOwner(localType)
	if err != nil {
		t.Fatal(err)
	}
	if providerOwned {
		t.Fatal("lexical source type was classified as provider-owned")
	}
}

func TestUnindexedPackageTypeFailsProviderFacetOwnership(t *testing.T) {
	selectedPackage := types.NewPackage("example.com/application", "application")
	packageType := types.NewTypeName(
		token.NoPos,
		selectedPackage,
		"PackageInfo",
		types.NewStruct(nil, nil),
	)
	if existing := selectedPackage.Scope().Insert(packageType); existing != nil {
		t.Fatal("package type identity was duplicated")
	}
	names := &File{owner: &Owner{registry: NewRegistry()}}

	if _, _, err := names.providerFacetOwner(packageType); err == nil {
		t.Fatal("unindexed package type was accepted")
	}
}

func TestProviderNamespaceIdentityIsTheCertifiedModule(t *testing.T) {
	const (
		recoveryModule = "@gotots/gostdlib/internal/facets/recovery-sync.js"
		firstShared    = "@first/provider/shared.js"
		secondShared   = "@second/provider/shared.js"
	)
	modules := []string{secondShared, recoveryModule, firstShared}
	first := NewRegistry()
	if err := first.indexProviderImportNames(modules); err != nil {
		t.Fatal(err)
	}
	second := NewRegistry()
	if err := second.indexProviderImportNames([]string{
		firstShared,
		recoveryModule,
		secondShared,
	}); err != nil {
		t.Fatal(err)
	}
	for _, module := range modules {
		if first.providerImportNameByModule[module] !=
			second.providerImportNameByModule[module] {
			t.Fatalf(
				"module %q name depends on discovery order: %q != %q",
				module,
				first.providerImportNameByModule[module],
				second.providerImportNameByModule[module],
			)
		}
	}
	if first.providerImportNameByModule[recoveryModule] != "recovery_sync" ||
		first.providerImportNameByModule[firstShared] != "shared" ||
		first.providerImportNameByModule[secondShared] != "shared__provider_1" {
		t.Fatalf("provider module names = %#v", first.providerImportNameByModule)
	}

	owner := NewOwner(nil, nil, first)
	names := testFileNames(
		t,
		owner,
		&ast.File{},
		nil,
		tsgo.NewFactory(),
		"modules/application/source.ts",
		nil,
	)
	firstLocal, firstRequest, err := names.providerImport(
		recoveryModule,
		api.ImportPhaseType,
	)
	if err != nil {
		t.Fatal(err)
	}
	secondLocal, secondRequest, err := names.providerImport(
		recoveryModule,
		api.ImportPhaseValue,
	)
	if err != nil {
		t.Fatal(err)
	}
	if firstLocal != "recovery_sync" || secondLocal != firstLocal ||
		firstRequest.Owner() != secondRequest.Owner() {
		t.Fatalf(
			"shared provider imports = %q/%#v and %q/%#v",
			firstLocal,
			firstRequest.Owner(),
			secondLocal,
			secondRequest.Owner(),
		)
	}
}
