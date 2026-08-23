package gostdlib_test

import (
	"encoding/json"
	"slices"
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
		BuildTags:        []string{"noasm"},
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
					Parameters: []gostdlib.ContractTypeDocument{
						gostdlib.ContractTypeParameterReference(0),
					},
					Results: []gostdlib.ContractTypeDocument{
						gostdlib.ContractTypeParameterReference(0),
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
		InvocationTransport: &gostdlib.InvocationTransportContractDocument{
			SchemaVersion:   gostdlib.InvocationTransportSchemaVersion,
			DeclarationRoot: "..",
			Transports: []gostdlib.InvocationTransportDocument{{
				SourceIdentity: "strings|kind=4|receiver=|name=Contains",
				Target: gostdlib.InvocationTransportTargetDocument{
					Specifier:         "@gotots/gostdlib/strings.js",
					SourcePath:        "src/strings.ts",
					DeclarationPath:   "dist/src/strings.d.ts",
					Access:            gostdlib.InvocationTransportAccessStaticMethod,
					Export:            "Contains",
					Member:            "$forward",
					TargetType:        "(value: () => void) => () => void",
					TargetFingerprint: digest('e'),
				},
				InputParameters:        []int{0},
				ResultOriginParameters: []int{0},
			}},
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
	if manifest.Digest() == "" || manifest.GoVersion() != "go1.26.4" {
		t.Fatalf("manifest identity is incomplete: %#v", manifest)
	}
	profile, ok := manifest.BuildProfile()
	if !ok || profile.CgoEnabled() ||
		profile.GOOS() != "linux" || profile.GOARCH() != "amd64" ||
		!slices.Equal(profile.Tags(), []string{"noasm"}) {
		t.Fatalf("manifest build profile = %#v, %t", profile, ok)
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
	transports := manifest.InvocationTransports()
	if len(transports) != 1 || transports[0].Target.Member != "$forward" {
		t.Fatalf("invocation transports = %#v", transports)
	}
	transports[0].InputParameters[0] = 9
	if manifest.InvocationTransports()[0].InputParameters[0] != 0 {
		t.Fatal("manifest exposed mutable invocation-transport storage")
	}
	reencoded, err := gostdlib.Encode(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if string(reencoded) != string(payload) {
		t.Fatalf("round trip changed canonical bytes\nfirst=%s\nnext=%s", payload, reencoded)
	}
}

func TestManifestRejectsInvalidInvocationTransport(t *testing.T) {
	document := validDocument()
	document.InvocationTransport = &gostdlib.InvocationTransportContractDocument{
		SchemaVersion:   gostdlib.InvocationTransportSchemaVersion,
		DeclarationRoot: "..",
		Transports: []gostdlib.InvocationTransportDocument{{
			SourceIdentity: document.Modules[0].Bindings[0].Identity,
			Target: gostdlib.InvocationTransportTargetDocument{
				Specifier:         document.Modules[0].Specifier,
				SourcePath:        document.Modules[0].SourcePath,
				DeclarationPath:   "dist/src/strings.d.ts",
				Access:            gostdlib.InvocationTransportAccessStaticMethod,
				Export:            document.Modules[0].Bindings[0].Export,
				Member:            "$forward",
				TargetType:        "(value: () => void) => () => void",
				TargetFingerprint: digest('d'),
			},
			InputParameters:        []int{0},
			ResultOriginParameters: []int{0},
		}},
	}
	if _, err := gostdlib.Seal(document); err != nil {
		t.Fatal(err)
	}

	document.InvocationTransport.DeclarationRoot = ""
	if _, err := gostdlib.Seal(document); err == nil {
		t.Fatal("empty invocation declaration root passed")
	}
	document.InvocationTransport.DeclarationRoot = ".."
	document.InvocationTransport.Transports[0].Target.DeclarationPath = "../strings.d.ts"
	if _, err := gostdlib.Seal(document); err == nil {
		t.Fatal("escaping invocation declaration path passed")
	}
	document.InvocationTransport.Transports[0].Target.DeclarationPath = "dist/src/strings.d.ts"
	document.InvocationTransport.SchemaVersion++
	if _, err := gostdlib.Seal(document); err == nil {
		t.Fatal("unsupported invocation transport schema passed")
	}
	document.InvocationTransport.SchemaVersion = gostdlib.InvocationTransportSchemaVersion
	document.InvocationTransport.Transports[0].InputParameters = []int{0, 0}
	if _, err := gostdlib.Seal(document); err == nil {
		t.Fatal("duplicate invocation input index passed")
	}
	document.InvocationTransport.Transports[0].InputParameters = []int{0}
	document.InvocationTransport.Transports[0].State = &gostdlib.InvocationTransportStateDocument{
		Kind: gostdlib.InvocationTransportStateCreate,
		Read: true,
	}
	if _, err := gostdlib.Seal(document); err == nil {
		t.Fatal("reading create-state transport passed")
	}
	document.InvocationTransport.Transports[0].State = nil
	document.InvocationTransport.Transports[0].SourceIdentity = "absent"
	if _, err := gostdlib.Seal(document); err == nil {
		t.Fatal("transport without a source owner passed")
	}
}

func TestManifestOwnsConditionalInvocationTransport(t *testing.T) {
	document := validDocument()
	document.InvocationTransport = conditionalInvocationContract(document)
	payload, err := gostdlib.Seal(document)
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := gostdlib.Parse(payload)
	if err != nil {
		t.Fatal(err)
	}
	transports := manifest.InvocationTransports()
	if len(transports) != 1 || transports[0].Conditional == nil ||
		!slices.Equal(transports[0].Conditional.CallableParameters, []int{0}) ||
		transports[0].Conditional.Replacement.Export != "ContainsSynchronous" {
		t.Fatalf("conditional invocation transport = %#v", transports)
	}
	transports[0].Conditional.CallableParameters[0] = 9
	transports[0].Conditional.Replacement.Export = "changed"
	next := manifest.InvocationTransports()[0].Conditional
	if next == nil || next.CallableParameters[0] != 0 ||
		next.Replacement.Export != "ContainsSynchronous" {
		t.Fatal("manifest exposed mutable conditional transport storage")
	}
}

func TestManifestRejectsInvalidConditionalInvocationTransport(t *testing.T) {
	document := validDocument()
	document.InvocationTransport = conditionalInvocationContract(document)
	transport := &document.InvocationTransport.Transports[0]
	conditional := transport.Conditional
	if conditional == nil {
		t.Fatal("test conditional transport is absent")
	}
	if _, err := gostdlib.Seal(document); err != nil {
		t.Fatal(err)
	}

	conditional.CallableParameters = nil
	if _, err := gostdlib.Seal(document); err == nil {
		t.Fatal("conditional transport without callable parameters passed")
	}
	conditional.CallableParameters = []int{1}
	if _, err := gostdlib.Seal(document); err == nil {
		t.Fatal("conditional transport over an uncertified input passed")
	}
	conditional.CallableParameters = []int{0}
	conditional.Replacement = transport.Target
	if _, err := gostdlib.Seal(document); err == nil {
		t.Fatal("conditional transport replacing itself passed")
	}
	conditional.Replacement = conditionalInvocationReplacement(document)
	conditional.Replacement.Access = gostdlib.InvocationTransportAccessStaticMethod
	conditional.Replacement.Member = "$forward"
	if _, err := gostdlib.Seal(document); err == nil {
		t.Fatal("conditional transport with a static replacement passed")
	}
}

func conditionalInvocationContract(
	document gostdlib.Document,
) *gostdlib.InvocationTransportContractDocument {
	return &gostdlib.InvocationTransportContractDocument{
		SchemaVersion:   gostdlib.InvocationTransportSchemaVersion,
		DeclarationRoot: "..",
		Transports: []gostdlib.InvocationTransportDocument{{
			SourceIdentity: document.Modules[0].Bindings[0].Identity,
			Target: gostdlib.InvocationTransportTargetDocument{
				Specifier:         document.Modules[0].Specifier,
				SourcePath:        document.Modules[0].SourcePath,
				DeclarationPath:   "dist/src/strings.d.ts",
				Access:            gostdlib.InvocationTransportAccessExport,
				Export:            "ContainsKernel",
				TargetType:        "(value: () => Promise<void>) => Promise<void>",
				TargetFingerprint: digest('d'),
			},
			InputParameters: []int{0},
			Conditional: &gostdlib.InvocationTransportConditionalDocument{
				CallableParameters: []int{0},
				Replacement:        conditionalInvocationReplacement(document),
			},
		}},
	}
}

func conditionalInvocationReplacement(
	document gostdlib.Document,
) gostdlib.InvocationTransportTargetDocument {
	return gostdlib.InvocationTransportTargetDocument{
		Specifier:         document.Modules[0].Specifier,
		SourcePath:        document.Modules[0].SourcePath,
		DeclarationPath:   "dist/src/strings.d.ts",
		Access:            gostdlib.InvocationTransportAccessExport,
		Export:            "ContainsSynchronous",
		TargetType:        "(value: () => void) => void",
		TargetFingerprint: digest('e'),
	}
}

func TestManifestRejectsInvalidGenericOperationShape(t *testing.T) {
	document := validDocument()
	binding := &document.Modules[0].Bindings[0]
	binding.GenericOperations = []gostdlib.GenericOperationDocument{{
		Kind: gostdlib.GenericOperationZero,
		Parameters: []gostdlib.ContractTypeDocument{
			gostdlib.ContractTypeParameterReference(0),
		},
		Results: []gostdlib.ContractTypeDocument{
			gostdlib.ContractTypeParameterReference(0),
		},
	}}
	if _, err := gostdlib.Seal(document); err == nil {
		t.Fatal("zero operation with a parameter passed")
	}
	binding.GenericOperations = []gostdlib.GenericOperationDocument{{
		Kind: gostdlib.GenericOperationKind("invented"),
		Results: []gostdlib.ContractTypeDocument{
			gostdlib.ContractTypeParameterReference(0),
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
		Parameters: []gostdlib.ContractTypeDocument{
			gostdlib.ContractCallableParameterReference(0),
		},
		Results: []gostdlib.ContractTypeDocument{
			gostdlib.ContractTypeParameterReference(0),
			gostdlib.ContractBoolReference(),
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
	binding.DefinedValue = gostdlib.DefinedValueRepresentationCanonical
	if _, err := gostdlib.Seal(document); err != nil {
		t.Fatalf("canonical callable identity was rejected: %v", err)
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
