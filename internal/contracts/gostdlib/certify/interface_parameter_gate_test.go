package certify

import (
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
)

func TestInterfaceParameterProfilesExactJoinProviderBoundaries(t *testing.T) {
	packageType := types.NewPackage("example.com/provider", "provider")
	methodSignature := types.NewSignatureType(
		nil,
		nil,
		nil,
		types.NewTuple(),
		types.NewTuple(types.NewVar(token.NoPos, nil, "ok", types.Typ[types.Bool])),
		false,
	)
	method := types.NewFunc(token.NoPos, packageType, "Ready", methodSignature)
	interfaceType := types.NewInterfaceType([]*types.Func{method}, nil).Complete()
	typeName := types.NewTypeName(token.NoPos, packageType, "Worker", nil)
	named := types.NewNamed(typeName, interfaceType, nil)
	signature := types.NewSignatureType(
		nil,
		nil,
		nil,
		types.NewTuple(types.NewVar(token.NoPos, nil, "worker", named)),
		types.NewTuple(),
		false,
	)
	callable := types.NewFunc(token.NoPos, packageType, "Use", signature)
	interfaceIdentity := "example.com/provider|kind=2|receiver=|name=Worker"
	callableIdentity := "example.com/provider|kind=4|receiver=|name=Use"
	source := goSurface{objects: map[string]goObject{
		interfaceIdentity: {object: typeName},
		callableIdentity:  {object: callable},
	}}
	ordinary := gostdlib.ProviderInterfaceDocument{
		Mode: gostdlib.ProviderInterfaceModeBridge,
		Methods: []gostdlib.ProviderInterfaceMethodDocument{{
			SourceIdentity:    "example.com/provider.Worker.Ready",
			Kind:              gostdlib.ProviderInterfaceMethodCallable,
			Effect:            gostdlib.EffectSynchronous,
			SourceSignature:   "func() bool",
			ContractSignature: "func() bool",
		}},
	}
	modules := []gostdlib.ModuleDocument{{Bindings: []gostdlib.BindingDocument{
		{
			Identity:          interfaceIdentity,
			Kind:              gostdlib.BindingType,
			ProviderInterface: &ordinary,
		},
		{
			Identity: callableIdentity,
			Kind:     gostdlib.BindingFunction,
		},
	}}}
	cooperative := ordinary
	cooperative.Methods = append(
		[]gostdlib.ProviderInterfaceMethodDocument(nil),
		ordinary.Methods...,
	)
	cooperative.Methods[0].Effect = gostdlib.EffectAwaitable
	profile := gostdlib.ProviderCallableProfileDocument{
		SourceIdentity:      callableIdentity,
		CanonicalParameters: []int{0},
		Interfaces: []gostdlib.ProviderCallableProfileInterfaceDocument{{
			SourceIdentity:    interfaceIdentity,
			ProviderInterface: cooperative,
		}},
	}
	facets := []gostdlib.FacetModuleDocument{{
		CallableProfiles: []gostdlib.ProviderCallableProfileDocument{profile},
	}}
	if err := verifyInterfaceParameterProfileCoverage(source, modules, facets); err != nil {
		t.Fatal(err)
	}
	if err := verifyInterfaceParameterProfileCoverage(source, modules, nil); err == nil ||
		!strings.Contains(err.Error(), "expected 1 profile") {
		t.Fatalf("missing interface profile error = %v", err)
	}
	synchronous := profile
	synchronous.Interfaces = append(
		[]gostdlib.ProviderCallableProfileInterfaceDocument(nil),
		profile.Interfaces...,
	)
	synchronous.Interfaces[0].ProviderInterface = ordinary
	facets[0].CallableProfiles = []gostdlib.ProviderCallableProfileDocument{synchronous}
	if err := verifyInterfaceParameterProfileCoverage(source, modules, facets); err == nil ||
		!strings.Contains(err.Error(), "expected 1 profile") {
		t.Fatalf("synchronous interface profile error = %v", err)
	}
	facets[0].CallableProfiles = []gostdlib.ProviderCallableProfileDocument{profile, profile}
	if err := verifyInterfaceParameterProfileCoverage(source, modules, facets); err == nil ||
		!strings.Contains(err.Error(), "certified 2 profile") {
		t.Fatalf("duplicate interface profile error = %v", err)
	}
}

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
	cooperativeParent := providerInterfaceForClosureTest(
		"example.com/provider.Parent.Child",
		"func() example.com/provider.Child",
		gostdlib.EffectAwaitable,
	)
	profile := gostdlib.ProviderCallableProfileDocument{
		SourceIdentity: "example.com/provider|kind=4|receiver=|name=Use",
		Interfaces: []gostdlib.ProviderCallableProfileInterfaceDocument{{
			SourceIdentity:    parentIdentity,
			ProviderInterface: cooperativeParent,
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
				gostdlib.EffectAwaitable,
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
