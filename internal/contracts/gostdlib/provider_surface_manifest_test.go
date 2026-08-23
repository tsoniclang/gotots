package gostdlib_test

import (
	"testing"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
)

func TestManifestOwnsUnsafeBuiltinsAsDirectExports(t *testing.T) {
	document := validDocument()
	document.Modules[0] = gostdlib.ModuleDocument{
		GoImportPath: "unsafe",
		Specifier:    "@gotots/gostdlib/unsafe.js",
		SourcePath:   "src/unsafe.ts",
		Bindings: []gostdlib.BindingDocument{{
			Identity:            "unsafe|kind=5|receiver=|name=String",
			Kind:                gostdlib.BindingBuiltin,
			Access:              gostdlib.AccessExport,
			Export:              "String",
			SourceSignature:     "builtin",
			SourceLocation:      "builtin",
			ImplementationOwner: "src/unsafe.ts",
			TargetFingerprint:   digest('b'),
		}},
	}
	payload, err := gostdlib.Seal(document)
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := gostdlib.Parse(payload)
	if err != nil {
		t.Fatal(err)
	}
	binding, ok := manifest.Binding("unsafe|kind=5|receiver=|name=String")
	if !ok || binding.Kind() != gostdlib.BindingBuiltin ||
		binding.Access() != gostdlib.AccessExport {
		t.Fatalf("builtin binding = %#v, %t", binding, ok)
	}
}

func TestManifestOwnsClosedPrivateProviderRepresentation(t *testing.T) {
	document := validDocument()
	document.FacetModules = []gostdlib.FacetModuleDocument{{
		Specifier:  "@gotots/gostdlib/internal/facets/binary.js",
		SourcePath: "src/internal/facets/binary.ts",
		Representations: []gostdlib.ProviderRepresentationDocument{{
			Export: "BinaryEndianRepresentation",
			SourceTypes: []string{
				"encoding/binary|kind=2|receiver=|name=bigEndian",
			},
			SourceInterfaces: []string{
				"encoding/binary|kind=2|receiver=|name=ByteOrder",
			},
			Methods: []gostdlib.ProviderRepresentationMethodDocument{{
				SourceIdentity:      "encoding/binary|kind=4|receiver=encoding/binary.bigEndian|name=Uint16",
				Member:              "Uint16",
				Effect:              gostdlib.EffectSynchronous,
				SourceSignature:     "func([]byte) uint16|params=|results=",
				SourceLocation:      "encoding/binary/binary.go:1:1",
				ImplementationOwner: "src/internal/facets/binary.ts",
				TargetFingerprint:   digest('e'),
			}},
			ImplementationOwner: "src/internal/facets/binary.ts",
			TargetFingerprint:   digest('f'),
		}},
		Facets: []gostdlib.FacetDocument{{
			Kind:           gostdlib.FacetNamedStructOperations,
			SourceIdentity: "encoding/binary|kind=2|receiver=|name=bigEndian",
			Capabilities: []gostdlib.FacetCapability{
				gostdlib.FacetCapabilityRepresentation,
			},
			Export:               "BinaryBigEndianOperations",
			RepresentationExport: "BinaryEndianRepresentation",
			ImplementationOwner:  "src/internal/facets/binary.ts",
			TargetFingerprint:    digest('d'),
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
	facet, ok := manifest.Facet(
		"encoding/binary|kind=2|receiver=|name=bigEndian",
		gostdlib.FacetNamedStructOperations,
		gostdlib.FacetCapabilityRepresentation,
	)
	if !ok {
		t.Fatal("representation facet is absent")
	}
	representation, ok := facet.Representation()
	if !ok || representation.Export() != "BinaryEndianRepresentation" {
		t.Fatalf("representation = %#v, %t", representation, ok)
	}
	method, ok := representation.Method(
		"encoding/binary|kind=4|receiver=encoding/binary.bigEndian|name=Uint16",
	)
	if !ok || method.Member() != "Uint16" ||
		method.Effect() != gostdlib.EffectSynchronous {
		t.Fatalf("method = %#v, %t", method, ok)
	}
	types := representation.SourceTypes()
	types[0] = "changed"
	if representation.SourceTypes()[0] == "changed" {
		t.Fatal("representation exposed mutable source types")
	}

	document.FacetModules[0].Representations[0].Methods = nil
	if _, err := gostdlib.Seal(document); err == nil {
		t.Fatal("representation without methods passed")
	}
	document.FacetModules[0].Representations[0].Methods = []gostdlib.ProviderRepresentationMethodDocument{{
		SourceIdentity:      "encoding/binary|kind=4|receiver=encoding/binary.bigEndian|name=Uint16",
		Member:              "Uint16",
		Effect:              gostdlib.EffectSynchronous,
		SourceSignature:     "func([]byte) uint16|params=|results=",
		SourceLocation:      "encoding/binary/binary.go:1:1",
		ImplementationOwner: "src/internal/facets/binary.ts",
		TargetFingerprint:   digest('e'),
	}}
	document.FacetModules[0].Representations[0].Methods[0].Effect =
		gostdlib.EffectInvalid
	if _, err := gostdlib.Seal(document); err == nil {
		t.Fatal("representation method without an effect passed")
	}
	document.FacetModules[0].Representations[0].Methods[0].Effect =
		gostdlib.EffectSynchronous
	document.FacetModules[0].Facets[0].RepresentationExport = "Missing"
	if _, err := gostdlib.Seal(document); err == nil {
		t.Fatal("facet with a missing representation passed")
	}
}

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
