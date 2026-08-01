package gostdlib_test

import (
	"encoding/json"
	"testing"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
)

func TestManifestRoundTripIsCanonicalAndImmutable(t *testing.T) {
	document := gostdlib.Document{
		SchemaVersion:    gostdlib.SchemaVersion,
		PackageName:      gostdlib.PackageName,
		PackageVersion:   "0.0.0",
		Backend:          "node",
		GoVersion:        "go1.26.4",
		MinimumGoVersion: "go1.26.4",
		MaximumGoVersion: "go1.26.4",
		GOOS:             "linux",
		GOARCH:           "amd64",
		RuntimeDigest:    digest('a'),
		ProviderDigest:   digest('c'),
		Modules: []gostdlib.ModuleDocument{{
			GoImportPath: "strings",
			Specifier:    "@gotots/gostdlib/strings.js",
			SourcePath:   "src/strings.ts",
			Bindings: []gostdlib.BindingDocument{{
				Identity: "strings|kind=4|receiver=|name=Contains",
				Kind:     gostdlib.BindingFunction,
				Access:   gostdlib.AccessExport,
				Effect:   gostdlib.EffectSynchronous,
				Export:   "Contains",
				GenericTypeArguments: []gostdlib.GenericTypeArgumentDocument{{
					TypeParameter: 1,
					Facet:         gostdlib.GenericTypeArgumentLogical,
				}},
				GenericOperations: []gostdlib.GenericOperationDocument{{
					Kind: gostdlib.GenericOperationCopy,
					Parameters: []gostdlib.GenericOperationTypeDocument{
						gostdlib.GenericOperationTypeParameterReference(0),
					},
					Results: []gostdlib.GenericOperationTypeDocument{
						gostdlib.GenericOperationTypeParameterReference(0),
					},
				}},
				SourceSignature:     "func(s, substr string) bool|params=s,substr|results=",
				SourceLocation:      "strings/strings.go:1:1",
				ImplementationOwner: "src/internal/portable/strings/search.ts",
				TargetFingerprint:   digest('b'),
			}},
		}},
		FacetModules: []gostdlib.FacetModuleDocument{{
			Specifier:  "@gotots/gostdlib/internal/facets/strings.js",
			SourcePath: "src/internal/facets/strings.ts",
			Facets: []gostdlib.FacetDocument{{
				Kind:                gostdlib.FacetRecoveryCallable,
				SourceIdentity:      "strings|kind=4|receiver=|name=Contains",
				Capabilities:        []gostdlib.FacetCapability{gostdlib.FacetCapabilityRecovery},
				Export:              "ContainsRecovery",
				Effect:              gostdlib.EffectSynchronous,
				ImplementationOwner: "src/internal/facets/strings.ts",
				TargetFingerprint:   digest('d'),
			}},
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
	if manifest.Digest() == "" || manifest.GoVersion() != "go1.26.4" {
		t.Fatalf("manifest identity is incomplete: %#v", manifest)
	}
	binding, ok := manifest.Binding(
		"strings|kind=4|receiver=|name=Contains",
	)
	if !ok || binding.Export() != "Contains" ||
		binding.ModuleSpecifier() != "@gotots/gostdlib/strings.js" ||
		binding.Effect() != gostdlib.EffectSynchronous {
		t.Fatalf("binding = %#v, %t", binding, ok)
	}
	operations := binding.GenericOperations()
	if len(operations) != 1 ||
		operations[0].Kind != gostdlib.GenericOperationCopy ||
		operations[0].Parameters[0].TypeParameter == nil ||
		*operations[0].Parameters[0].TypeParameter != 0 {
		t.Fatalf("generic operations = %#v", operations)
	}
	*operations[0].Parameters[0].TypeParameter = 9
	nextOperation := binding.GenericOperations()[0].Parameters[0]
	if nextOperation.TypeParameter == nil || *nextOperation.TypeParameter != 0 {
		t.Fatal("binding exposed mutable generic-operation storage")
	}
	typeArguments := binding.GenericTypeArguments()
	typeArguments[0].TypeParameter = 9
	if binding.GenericTypeArguments()[0].TypeParameter != 1 {
		t.Fatal("binding exposed mutable generic-type-argument storage")
	}
	modules := manifest.Modules()
	modules[0] = gostdlib.Module{}
	if manifest.Modules()[0].GoImportPath() != "strings" {
		t.Fatal("manifest exposed mutable module storage")
	}
	facet, ok := manifest.Facet(
		"strings|kind=4|receiver=|name=Contains",
		gostdlib.FacetRecoveryCallable,
		gostdlib.FacetCapabilityRecovery,
	)
	if !ok || facet.Export() != "ContainsRecovery" ||
		facet.Effect() != gostdlib.EffectSynchronous {
		t.Fatalf("facet = %#v, %t", facet, ok)
	}
	facets := manifest.FacetModules()
	facets[0] = gostdlib.FacetModule{}
	if manifest.FacetModules()[0].Specifier() == "" {
		t.Fatal("manifest exposed mutable facet-module storage")
	}
	reencoded, err := gostdlib.Encode(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if string(reencoded) != string(payload) {
		t.Fatalf("round trip changed canonical bytes\nfirst=%s\nnext=%s", payload, reencoded)
	}
}

func TestManifestRejectsInvalidGenericOperationShape(t *testing.T) {
	document := validDocument()
	binding := &document.Modules[0].Bindings[0]
	binding.GenericOperations = []gostdlib.GenericOperationDocument{{
		Kind: gostdlib.GenericOperationZero,
		Parameters: []gostdlib.GenericOperationTypeDocument{
			gostdlib.GenericOperationTypeParameterReference(0),
		},
		Results: []gostdlib.GenericOperationTypeDocument{
			gostdlib.GenericOperationTypeParameterReference(0),
		},
	}}
	if _, err := gostdlib.Seal(document); err == nil {
		t.Fatal("zero operation with a parameter passed")
	}
	binding.GenericOperations = []gostdlib.GenericOperationDocument{{
		Kind: gostdlib.GenericOperationKind("invented"),
		Results: []gostdlib.GenericOperationTypeDocument{
			gostdlib.GenericOperationTypeParameterReference(0),
		},
	}}
	if _, err := gostdlib.Seal(document); err == nil {
		t.Fatal("open generic operation passed")
	}
}

func TestManifestOwnsCallableParameterGenericOperation(t *testing.T) {
	document := validDocument()
	binding := &document.Modules[0].Bindings[0]
	binding.GenericOperations = []gostdlib.GenericOperationDocument{{
		Kind: gostdlib.GenericOperationInterfaceAssertOK,
		Parameters: []gostdlib.GenericOperationTypeDocument{
			gostdlib.GenericOperationCallableParameterReference(0),
		},
		Results: []gostdlib.GenericOperationTypeDocument{
			gostdlib.GenericOperationTypeParameterReference(0),
			gostdlib.GenericOperationBoolReference(),
		},
	}}
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
		t.Fatal("callable-parameter operation binding is absent")
	}
	operations := selected.GenericOperations()
	if len(operations) != 1 ||
		operations[0].Parameters[0].CallableParameter == nil ||
		*operations[0].Parameters[0].CallableParameter != 0 {
		t.Fatalf("callable-parameter operation = %#v", operations)
	}
	*operations[0].Parameters[0].CallableParameter = 9
	next := selected.GenericOperations()[0].Parameters[0].CallableParameter
	if next == nil || *next != 0 {
		t.Fatal("callable-parameter operation exposed mutable storage")
	}
}

func TestManifestRequiresEffectsExactlyOnCallableBindings(t *testing.T) {
	document := validDocument()
	document.Modules[0].Bindings[0].Effect = gostdlib.EffectInvalid
	if _, err := gostdlib.Seal(document); err == nil {
		t.Fatal("function without an effect passed")
	}

	binding := &document.Modules[0].Bindings[0]
	binding.Kind = gostdlib.BindingType
	binding.Representation = gostdlib.RepresentationDirect
	binding.DefinedValue = gostdlib.DefinedValueRepresentationIdentity
	if _, err := gostdlib.Seal(document); err == nil {
		t.Fatal("callable identity without an effect passed")
	}
	binding.DefinedValue = gostdlib.DefinedValueRepresentationOperations
	binding.Effect = gostdlib.EffectSynchronous
	if _, err := gostdlib.Seal(document); err == nil {
		t.Fatal("operation-represented value with a callable effect passed")
	}
}

func TestManifestFailsClosedOnUnknownDuplicateAndResealedChanges(t *testing.T) {
	document := validDocument()
	payload, err := gostdlib.Seal(document)
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	if err := json.Unmarshal(payload, &raw); err != nil {
		t.Fatal(err)
	}
	raw["unexpected"] = true
	unknown, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := gostdlib.Parse(unknown); err == nil {
		t.Fatal("unknown manifest field was accepted")
	}

	document.Modules = append(document.Modules, document.Modules[0])
	if _, err := gostdlib.Seal(document); err == nil {
		t.Fatal("duplicate module was accepted")
	}

	document = validDocument()
	payload, err = gostdlib.Seal(document)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(payload, &raw); err != nil {
		t.Fatal(err)
	}
	raw["goVersion"] = "go1.27"
	changed, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := gostdlib.Parse(changed); err == nil {
		t.Fatal("changed manifest with stale digest was accepted")
	}
}

func TestManifestRejectsOpenOrMismatchedFacetCapabilities(t *testing.T) {
	document := validDocument()
	document.FacetModules = []gostdlib.FacetModuleDocument{{
		Specifier:  "@gotots/gostdlib/internal/facets/sync.js",
		SourcePath: "src/internal/facets/sync.ts",
		Facets: []gostdlib.FacetDocument{{
			Kind:                gostdlib.FacetNamedStructOperations,
			SourceIdentity:      "sync|kind=2|receiver=|name=Mutex",
			Capabilities:        []gostdlib.FacetCapability{gostdlib.FacetCapabilityZero},
			Export:              "MutexOperations",
			ImplementationOwner: "src/internal/facets/sync.ts",
			TargetFingerprint:   digest('d'),
		}},
	}}
	if _, err := gostdlib.Seal(document); err != nil {
		t.Fatal(err)
	}
	document.FacetModules[0].Facets[0].Capabilities =
		[]gostdlib.FacetCapability{"invented"}
	if _, err := gostdlib.Seal(document); err == nil {
		t.Fatal("open facet capability passed")
	}
	document = validDocument()
	document.FacetModules = []gostdlib.FacetModuleDocument{{
		Specifier:  "@gotots/gostdlib/internal/facets/sync.js",
		SourcePath: "src/internal/facets/sync.ts",
		Facets: []gostdlib.FacetDocument{{
			Kind:                gostdlib.FacetNamedStructOperations,
			SourceIdentity:      "sync|kind=2|receiver=|name=Mutex",
			Capabilities:        []gostdlib.FacetCapability{gostdlib.FacetCapabilityStorage},
			Export:              "MutexOperations",
			ImplementationOwner: "src/internal/facets/sync.ts",
			TargetFingerprint:   digest('d'),
		}},
	}}
	if _, err := gostdlib.Seal(document); err == nil {
		t.Fatal("storage facet without storage export passed")
	}
}

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
				SourceLocation:      "strings/reader.go:2:1",
				ImplementationOwner: "src/strings.ts",
				TargetFingerprint:   digest('e'),
			},
			{
				SourceIdentity:  "strings|kind=4|receiver=strings.Reader|name=private",
				Kind:            gostdlib.ProviderInterfaceMethodRuntimeOnly,
				SourceSignature: "func()|params=|results=",
				SourceLocation:  "strings/reader.go:1:1",
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
			SourceIdentity:  "strings|kind=4|receiver=strings.Reader|name=private",
			Kind:            gostdlib.ProviderInterfaceMethodRuntimeOnly,
			SourceSignature: "func()|params=|results=",
			SourceLocation:  "strings/reader.go:1:1",
		}},
	}
	if _, err := gostdlib.Seal(document); err == nil {
		t.Fatal("open bridge interface with a sealing method passed")
	}
}

func validDocument() gostdlib.Document {
	return gostdlib.Document{
		SchemaVersion:    gostdlib.SchemaVersion,
		PackageName:      gostdlib.PackageName,
		PackageVersion:   "0.0.0",
		Backend:          "node",
		GoVersion:        "go1.26.4",
		MinimumGoVersion: "go1.26.4",
		MaximumGoVersion: "go1.26.4",
		GOOS:             "linux",
		GOARCH:           "amd64",
		RuntimeDigest:    digest('a'),
		ProviderDigest:   digest('c'),
		Modules: []gostdlib.ModuleDocument{{
			GoImportPath: "strings",
			Specifier:    "@gotots/gostdlib/strings.js",
			SourcePath:   "src/strings.ts",
			Bindings: []gostdlib.BindingDocument{{
				Identity:            "strings|kind=4|receiver=|name=Contains",
				Kind:                gostdlib.BindingFunction,
				Access:              gostdlib.AccessExport,
				Effect:              gostdlib.EffectSynchronous,
				Export:              "Contains",
				SourceSignature:     "func(s, substr string) bool|params=s,substr|results=",
				SourceLocation:      "strings/strings.go:1:1",
				ImplementationOwner: "src/strings.ts",
				TargetFingerprint:   digest('b'),
			}},
		}},
	}
}

func digest(character byte) string {
	value := make([]byte, 64)
	for index := range value {
		value[index] = character
	}
	return string(value)
}
