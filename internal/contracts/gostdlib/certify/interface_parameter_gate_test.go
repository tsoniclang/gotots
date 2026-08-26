package certify

import (
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
)

func TestProviderProfileIncludesTransitiveInterfaceClosure(t *testing.T) {
	packageType := types.NewPackage("example.com/provider", "provider")
	childMethod := types.NewFunc(
		token.NoPos,
		packageType,
		"Value",
		types.NewSignatureType(
			nil,
			nil,
			nil,
			types.NewTuple(),
			types.NewTuple(types.NewVar(
				token.NoPos,
				nil,
				"value",
				types.Typ[types.Bool],
			)),
			false,
		),
	)
	childName := types.NewTypeName(token.NoPos, packageType, "Child", nil)
	child := types.NewNamed(
		childName,
		types.NewInterfaceType([]*types.Func{childMethod}, nil).Complete(),
		nil,
	)
	parentMethod := types.NewFunc(
		token.NoPos,
		packageType,
		"Child",
		types.NewSignatureType(
			nil,
			nil,
			nil,
			types.NewTuple(),
			types.NewTuple(types.NewVar(token.NoPos, nil, "child", child)),
			false,
		),
	)
	parentName := types.NewTypeName(token.NoPos, packageType, "Parent", nil)
	_ = types.NewNamed(
		parentName,
		types.NewInterfaceType([]*types.Func{parentMethod}, nil).Complete(),
		nil,
	)
	parentIdentity := "example.com/provider|kind=2|receiver=|name=Parent"
	childIdentity := "example.com/provider|kind=2|receiver=|name=Child"
	source := goSurface{objects: map[string]goObject{
		parentIdentity: {object: parentName},
		childIdentity:  {object: childName},
	}}
	ordinaryParent := providerInterfaceForClosureTest(
		"example.com/provider.Parent.Child",
		"func() example.com/provider.Child",
		gostdlib.EffectSynchronous,
	)
	ordinaryChild := providerInterfaceForClosureTest(
		"example.com/provider.Child.Value",
		"func() bool",
		gostdlib.EffectSynchronous,
	)
	modules := []gostdlib.ModuleDocument{{Bindings: []gostdlib.BindingDocument{
		{Identity: parentIdentity, ProviderInterface: &ordinaryParent},
		{Identity: childIdentity, ProviderInterface: &ordinaryChild},
	}}}
	profileParent := providerInterfaceForClosureTest(
		"example.com/provider.Parent.Child",
		"func() example.com/provider.Child",
		gostdlib.EffectSynchronous,
	)
	profile := gostdlib.ProviderCallableProfileDocument{
		SourceIdentity: "example.com/provider|kind=4|receiver=|name=Use",
		Interfaces: []gostdlib.ProviderCallableProfileInterfaceDocument{{
			SourceIdentity:    parentIdentity,
			ProviderInterface: profileParent,
		}},
	}
	facets := []gostdlib.FacetModuleDocument{{
		CallableProfiles: []gostdlib.ProviderCallableProfileDocument{profile},
	}}
	err := verifyProviderProfileInterfaceClosure(source, modules, facets)
	if err == nil || !strings.Contains(err.Error(), childIdentity) {
		t.Fatalf("missing transitive interface error = %v", err)
	}
	profile.Interfaces = append(
		profile.Interfaces,
		gostdlib.ProviderCallableProfileInterfaceDocument{
			SourceIdentity: childIdentity,
			ProviderInterface: providerInterfaceForClosureTest(
				"example.com/provider.Child.Value",
				"func() bool",
				gostdlib.EffectSynchronous,
			),
		},
	)
	facets[0].CallableProfiles[0] = profile
	if err := verifyProviderProfileInterfaceClosure(source, modules, facets); err != nil {
		t.Fatal(err)
	}
}

func providerInterfaceForClosureTest(
	methodIdentity string,
	signature string,
	effect gostdlib.EffectKind,
) gostdlib.ProviderInterfaceDocument {
	return gostdlib.ProviderInterfaceDocument{
		Mode: gostdlib.ProviderInterfaceModeBridge,
		Methods: []gostdlib.ProviderInterfaceMethodDocument{{
			SourceIdentity:    methodIdentity,
			Kind:              gostdlib.ProviderInterfaceMethodCallable,
			Effect:            effect,
			SourceSignature:   signature,
			ContractSignature: signature,
		}},
	}
}
