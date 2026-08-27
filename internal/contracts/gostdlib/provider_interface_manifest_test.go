package gostdlib_test

import (
	"testing"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
)

func TestManifestOwnsProviderInterfaceSurface(t *testing.T) {
	document := validDocument()
	binding := &document.Modules[0].Bindings[0]
	binding.Kind = gostdlib.BindingType
	binding.Representation = gostdlib.RepresentationDirect
	binding.Effect = gostdlib.EffectInvalid
	binding.ProviderInterface = &gostdlib.ProviderInterfaceDocument{
		Mode: gostdlib.ProviderInterfaceModeSealedNative,
		Methods: []gostdlib.ProviderInterfaceMethodDocument{
			{
				SourceIdentity:      "strings|kind=4|receiver=strings.Reader|name=Read",
				Kind:                gostdlib.ProviderInterfaceMethodCallable,
				Member:              "Read",
				Effect:              gostdlib.EffectSynchronous,
				SourceSignature:     "func([]byte) (int, error)|params=|results=",
				ContractSignature:   "func([]byte) (int, error)",
				SourceLocation:      "strings/reader.go:2:1",
				ImplementationOwner: "src/strings.ts",
				TargetFingerprint:   digest('e'),
			},
			{
				SourceIdentity:    "strings|kind=4|receiver=strings.Reader|name=private",
				Kind:              gostdlib.ProviderInterfaceMethodRuntimeOnly,
				SourceSignature:   "func()|params=|results=",
				ContractSignature: "func()",
				SourceLocation:    "strings/reader.go:1:1",
			},
		},
	}
	payload, err := gostdlib.Seal(document)
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := gostdlib.Parse(payload)
	if err != nil {
		t.Fatal(err)
	}
	selected, ok := manifest.Binding(binding.Identity)
	if !ok {
		t.Fatal("provider interface binding is absent")
	}
	providerInterface, ok := selected.ProviderInterface()
	if !ok {
		t.Fatal("provider interface evidence is absent")
	}
	method, ok := providerInterface.Method(
		"strings|kind=4|receiver=strings.Reader|name=Read",
	)
	if !ok || method.Member() != "Read" ||
		method.Effect() != gostdlib.EffectSynchronous ||
		providerInterface.Mode() != gostdlib.ProviderInterfaceModeSealedNative {
		t.Fatalf("provider interface method = %#v, %t", method, ok)
	}
	methods := providerInterface.Methods()
	methods[0] = gostdlib.ProviderInterfaceMethod{}
	if len(providerInterface.Methods()) != 2 ||
		providerInterface.Methods()[0].SourceIdentity() == "" {
		t.Fatal("provider interface exposed mutable method storage")
	}

	document.Modules[0].Bindings[0].ProviderInterface.Methods[1].Member = "private"
	if _, err := gostdlib.Seal(document); err == nil {
		t.Fatal("runtime-only interface method with a public target passed")
	}

	document = validDocument()
	binding = &document.Modules[0].Bindings[0]
	binding.Kind = gostdlib.BindingType
	binding.Representation = gostdlib.RepresentationDirect
	binding.Effect = gostdlib.EffectInvalid
	binding.ProviderInterface = &gostdlib.ProviderInterfaceDocument{
		Mode: gostdlib.ProviderInterfaceModeBridge,
		Methods: []gostdlib.ProviderInterfaceMethodDocument{{
			SourceIdentity:    "strings|kind=4|receiver=strings.Reader|name=private",
			Kind:              gostdlib.ProviderInterfaceMethodRuntimeOnly,
			SourceSignature:   "func()|params=|results=",
			ContractSignature: "func()",
			SourceLocation:    "strings/reader.go:1:1",
		}},
	}
	if _, err := gostdlib.Seal(document); err == nil {
		t.Fatal("open bridge interface with a sealing method passed")
	}
}

func TestManifestOwnsLanguageProviderInterfaceBinding(t *testing.T) {
	document := validDocument()
	document.FacetModules = []gostdlib.FacetModuleDocument{{
		Specifier:  "@gotots/gostdlib/internal/facets/provider-error.js",
		SourcePath: "src/internal/facets/provider-error.ts",
		ProviderInterfaces: []gostdlib.ProviderInterfaceBindingDocument{{
			SourceIdentity: gostdlib.LanguageErrorInterfaceIdentity,
			Export:         "ProviderErrorInterface",
			ProviderInterface: gostdlib.ProviderInterfaceDocument{
				Mode: gostdlib.ProviderInterfaceModeBridge,
				Methods: []gostdlib.ProviderInterfaceMethodDocument{{
					SourceIdentity:      gostdlib.LanguageErrorMethodIdentity,
					Kind:                gostdlib.ProviderInterfaceMethodCallable,
					Member:              "Error",
					Effect:              gostdlib.EffectSynchronous,
					SourceSignature:     "func() string",
					ContractSignature:   "func() string",
					SourceLocation:      "builtin",
					ImplementationOwner: "src/internal/facets/provider-error.ts",
					TargetFingerprint:   digest('e'),
				}},
			},
			TargetFingerprint: digest('f'),
		}},
	}}
	payload, err := gostdlib.Seal(document)
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := gostdlib.Parse(payload)
	if err != nil {
		t.Fatal(err)
	}
	selected, ok := manifest.ProviderInterface(
		gostdlib.LanguageErrorInterfaceIdentity,
	)
	if !ok || selected.Export() != "ProviderErrorInterface" ||
		selected.ModuleSpecifier() !=
			"@gotots/gostdlib/internal/facets/provider-error.js" ||
		selected.TargetFingerprint() != digest('f') {
		t.Fatalf("language provider interface = %#v, %t", selected, ok)
	}
	providerInterface := selected.ProviderInterface()
	method, ok := providerInterface.Method(gostdlib.LanguageErrorMethodIdentity)
	if !ok || method.Member() != "Error" ||
		method.Effect() != gostdlib.EffectSynchronous {
		t.Fatalf("language provider method = %#v, %t", method, ok)
	}
	methods := providerInterface.Methods()
	methods[0] = gostdlib.ProviderInterfaceMethod{}
	if providerInterface.Methods()[0].SourceIdentity() == "" {
		t.Fatal("language provider interface exposed mutable method storage")
	}

	document.FacetModules = append(
		document.FacetModules,
		document.FacetModules[0],
	)
	if _, err := gostdlib.Seal(document); err == nil {
		t.Fatal("duplicate language provider-interface owner passed")
	}
}
