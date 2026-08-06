package gostdlib_test

import (
	"go/token"
	"go/types"
	"testing"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	gostdlibsource "github.com/tsoniclang/gotots/internal/contracts/gostdlib/sourcecontract"
)

func TestManifestOwnsCertifiedProviderInterfaceCapability(t *testing.T) {
	document := providerCapabilityDocument(t)
	payload, err := gostdlib.Seal(document)
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := gostdlib.Parse(payload)
	if err != nil {
		t.Fatal(err)
	}
	capabilities := manifest.ProviderInterfaceCapabilities(
		gostdlib.LanguageErrorInterfaceIdentity,
	)
	if len(capabilities) != 1 || !capabilities[0].Valid() ||
		capabilities[0].Usage() != gostdlib.ProviderInterfaceCapabilityUsageGeneratedBridge ||
		capabilities[0].BaseExport() != "CanonicalError" ||
		capabilities[0].TargetExport() != "ProviderErrorUnwrapDirect" ||
		capabilities[0].ViewExport() != "AsProviderErrorUnwrapDirect" {
		t.Fatalf("provider capability = %#v", capabilities)
	}
	target := capabilities[0].TargetInterface()
	base := capabilities[0].BaseInterface()
	wantTargetIdentity := document.FacetModules[0].
		ProviderInterfaceCapabilities[0].TargetSourceIdentity
	if target.SourceIdentity() != wantTargetIdentity ||
		base.SourceIdentity() != gostdlib.LanguageErrorInterfaceIdentity ||
		base.Export() != "CanonicalError" ||
		target.ProviderInterface().Mode() != gostdlib.ProviderInterfaceModeBridge {
		t.Fatalf("provider capability target = %#v", target)
	}
	profile := capabilities[0].ProfileInterfaces()
	if len(profile) != 2 || !profile[0].Valid() || !profile[1].Valid() {
		t.Fatalf("provider capability profile = %#v", profile)
	}
	profile[0] = gostdlib.ProviderCallableProfileInterface{}
	profile = capabilities[0].ProfileInterfaces()
	if len(profile) != 2 || !profile[0].Valid() {
		t.Fatal("provider capability exposed mutable profile-interface storage")
	}
	capabilities[0] = gostdlib.ProviderInterfaceCapability{}
	if len(manifest.ProviderInterfaceCapabilities(
		gostdlib.LanguageErrorInterfaceIdentity,
	)) != 1 {
		t.Fatal("manifest exposed mutable provider-capability storage")
	}
	directDocument := providerCapabilityDocument(t)
	directDocument.FacetModules[0].ProviderInterfaceCapabilities[0].BaseExport =
		"ProviderErrorInterface"
	directPayload, err := gostdlib.Seal(directDocument)
	if err != nil {
		t.Fatal(err)
	}
	directManifest, err := gostdlib.Parse(directPayload)
	if err != nil {
		t.Fatal(err)
	}
	directCapabilities := directManifest.ProviderInterfaceCapabilities(
		gostdlib.LanguageErrorInterfaceIdentity,
	)
	if len(directCapabilities) != 1 || !directCapabilities[0].Valid() ||
		directCapabilities[0].BaseInterface().Export() != "ProviderErrorInterface" {
		t.Fatalf("direct provider capability = %#v", directCapabilities)
	}
	modules := directManifest.FacetModules()
	if len(modules) != 1 {
		t.Fatalf("facet modules = %d, want 1", len(modules))
	}
	moduleCapabilities := modules[0].ProviderInterfaceCapabilities()
	if len(moduleCapabilities) != 1 || !moduleCapabilities[0].Valid() ||
		moduleCapabilities[0].BaseInterface().Export() != "ProviderErrorInterface" {
		t.Fatalf("facet-module provider capability = %#v", moduleCapabilities)
	}
}

func TestManifestRejectsUnjoinedProviderInterfaceCapability(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*gostdlib.ProviderInterfaceCapabilityDocument)
	}{
		{
			name: "usage",
			mutate: func(capability *gostdlib.ProviderInterfaceCapabilityDocument) {
				capability.Usage = gostdlib.ProviderInterfaceCapabilityUsageInvalid
			},
		},
		{
			name: "base export",
			mutate: func(capability *gostdlib.ProviderInterfaceCapabilityDocument) {
				capability.BaseExport = "MissingBase"
			},
		},
		{
			name: "profile",
			mutate: func(capability *gostdlib.ProviderInterfaceCapabilityDocument) {
				capability.ProfileKey = digest('9')
			},
		},
		{
			name: "target",
			mutate: func(capability *gostdlib.ProviderInterfaceCapabilityDocument) {
				capability.TargetExport = "MissingTarget"
			},
		},
		{
			name: "view fingerprint",
			mutate: func(capability *gostdlib.ProviderInterfaceCapabilityDocument) {
				capability.ViewFingerprint = ""
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			document := providerCapabilityDocument(t)
			test.mutate(&document.FacetModules[0].ProviderInterfaceCapabilities[0])
			if _, err := gostdlib.Seal(document); err == nil {
				t.Fatal("unjoined provider capability passed")
			}
		})
	}
	document := providerCapabilityDocument(t)
	document.FacetModules[0].ProviderInterfaceCapabilities = append(
		document.FacetModules[0].ProviderInterfaceCapabilities,
		document.FacetModules[0].ProviderInterfaceCapabilities[0],
	)
	if _, err := gostdlib.Seal(document); err == nil {
		t.Fatal("duplicate provider capability passed")
	}
}

func TestProviderCapabilityMethodContractExactJoinRejectsDrift(t *testing.T) {
	contract := types.NewInterfaceType(
		[]*types.Func{types.NewFunc(
			token.NoPos,
			nil,
			"Unwrap",
			types.NewSignatureType(
				nil,
				nil,
				nil,
				types.NewTuple(),
				types.NewTuple(types.NewVar(
					token.NoPos,
					nil,
					"",
					types.Universe.Lookup("error").Type(),
				)),
				false,
			),
		)},
		nil,
	).Complete()
	document := providerCapabilityDocument(t)
	payload, err := gostdlib.Seal(document)
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := gostdlib.Parse(payload)
	if err != nil {
		t.Fatal(err)
	}
	capability := manifest.ProviderInterfaceCapabilities(
		gostdlib.LanguageErrorInterfaceIdentity,
	)[0]
	methods, matched, err := gostdlibsource.SelectProviderInterfaceMethods(
		capability.TargetInterface(),
		contract,
	)
	if err != nil || !matched || len(methods) != 1 {
		t.Fatalf("exact provider capability join = %d, %t, %v", len(methods), matched, err)
	}
	for index := range document.FacetModules[0].CallableProfiles[0].Interfaces {
		selected := &document.FacetModules[0].CallableProfiles[0].Interfaces[index]
		if selected.Export == "ProviderErrorUnwrapDirect" {
			selected.ProviderInterface.Methods[0].ContractSignature = "func() string"
		}
	}
	payload, err = gostdlib.Seal(document)
	if err != nil {
		t.Fatal(err)
	}
	manifest, err = gostdlib.Parse(payload)
	if err != nil {
		t.Fatal(err)
	}
	capability = manifest.ProviderInterfaceCapabilities(
		gostdlib.LanguageErrorInterfaceIdentity,
	)[0]
	methods, matched, err = gostdlibsource.SelectProviderInterfaceMethods(
		capability.TargetInterface(),
		contract,
	)
	if err != nil || matched || len(methods) != 0 {
		t.Fatalf("drifted provider capability join = %d, %t, %v", len(methods), matched, err)
	}
}

func providerCapabilityDocument(t *testing.T) gostdlib.Document {
	t.Helper()
	baseMethod := gostdlib.ProviderInterfaceMethodDocument{
		SourceIdentity:      gostdlib.LanguageErrorMethodIdentity,
		Kind:                gostdlib.ProviderInterfaceMethodCallable,
		Member:              "Error",
		Effect:              gostdlib.EffectSynchronous,
		SourceSignature:     "func() string",
		ContractSignature:   "func() string",
		SourceLocation:      "builtin",
		ImplementationOwner: "src/internal/facets/provider-error.ts",
		TargetFingerprint:   digest('1'),
	}
	protocol := gostdlib.ProviderProtocolInterfaceDocument{
		Methods: []gostdlib.ProviderProtocolMethodDocument{{
			Name: "Unwrap",
			Results: []gostdlib.ContractTypeDocument{
				gostdlib.ContractCallableParameterReference(0),
			},
		}},
	}
	targetIdentity, err := gostdlib.BuildProviderProtocolInterfaceIdentity(protocol)
	if err != nil {
		t.Fatal(err)
	}
	targetMethodIdentity, targetMethodSignature, err :=
		gostdlib.ProviderProtocolMethodSource(targetIdentity, protocol.Methods[0])
	if err != nil {
		t.Fatal(err)
	}
	targetMethod := gostdlib.ProviderInterfaceMethodDocument{
		SourceIdentity:      targetMethodIdentity,
		Kind:                gostdlib.ProviderInterfaceMethodCallable,
		Member:              "Unwrap",
		Effect:              gostdlib.EffectSynchronous,
		SourceSignature:     targetMethodSignature,
		ContractSignature:   "func() error",
		SourceLocation:      "protocol",
		ImplementationOwner: "src/internal/facets/provider-error.ts",
		TargetFingerprint:   digest('2'),
	}
	key, err := gostdlib.BuildProviderCallableProfileKey(
		[]gostdlib.ProviderCallableProfileKeyInterface{
			{
				SourceIdentity: gostdlib.LanguageErrorInterfaceIdentity,
				Methods: []gostdlib.ProviderCallableProfileKeyMethod{{
					SourceIdentity: baseMethod.SourceIdentity,
					Effect:         baseMethod.Effect,
				}},
			},
			{
				SourceIdentity: targetIdentity,
				Methods: []gostdlib.ProviderCallableProfileKeyMethod{{
					SourceIdentity: targetMethod.SourceIdentity,
					Effect:         targetMethod.Effect,
				}},
			},
		},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	protocolParameter := 0
	document := validDocument()
	document.FacetModules = []gostdlib.FacetModuleDocument{{
		Specifier:  "@gotots/gostdlib/internal/facets/provider-error.js",
		SourcePath: "src/internal/facets/provider-error.ts",
		ProviderInterfaces: []gostdlib.ProviderInterfaceBindingDocument{{
			SourceIdentity: gostdlib.LanguageErrorInterfaceIdentity,
			Export:         "ProviderErrorInterface",
			ProviderInterface: gostdlib.ProviderInterfaceDocument{
				Mode:    gostdlib.ProviderInterfaceModeBridge,
				Methods: []gostdlib.ProviderInterfaceMethodDocument{baseMethod},
			},
			TargetFingerprint: digest('3'),
		}},
		CallableProfiles: []gostdlib.ProviderCallableProfileDocument{{
			SourceIdentity:      "errors|kind=4|receiver=|name=Is",
			ProfileKey:          key,
			Export:              "ErrorsIsDirect",
			Required:            true,
			CanonicalParameters: []int{0},
			GuardInterfaces:     []string{targetIdentity},
			Interfaces: []gostdlib.ProviderCallableProfileInterfaceDocument{
				{
					SourceIdentity:         targetIdentity,
					Export:                 "ProviderErrorUnwrapDirect",
					Protocol:               &protocol,
					ProtocolValueParameter: &protocolParameter,
					ProviderInterface: gostdlib.ProviderInterfaceDocument{
						Mode:    gostdlib.ProviderInterfaceModeBridge,
						Methods: []gostdlib.ProviderInterfaceMethodDocument{targetMethod},
					},
					TargetFingerprint: digest('4'),
				},
				{
					SourceIdentity: gostdlib.LanguageErrorInterfaceIdentity,
					Export:         "CanonicalError",
					ProviderInterface: gostdlib.ProviderInterfaceDocument{
						Mode:    gostdlib.ProviderInterfaceModeBridge,
						Methods: []gostdlib.ProviderInterfaceMethodDocument{baseMethod},
					},
					TargetFingerprint: digest('7'),
				},
			},
			Effect:              gostdlib.EffectSynchronous,
			ImplementationOwner: "src/internal/facets/provider-error.ts",
			TargetFingerprint:   digest('5'),
		}},
		ProviderInterfaceCapabilities: []gostdlib.ProviderInterfaceCapabilityDocument{{
			Usage:                 gostdlib.ProviderInterfaceCapabilityUsageGeneratedBridge,
			BaseSourceIdentity:    gostdlib.LanguageErrorInterfaceIdentity,
			BaseExport:            "CanonicalError",
			ProfileSourceIdentity: "errors|kind=4|receiver=|name=Is",
			ProfileKey:            key,
			TargetSourceIdentity:  targetIdentity,
			TargetExport:          "ProviderErrorUnwrapDirect",
			ViewExport:            "AsProviderErrorUnwrapDirect",
			ImplementationOwner:   "src/internal/facets/provider-error.ts",
			ViewFingerprint:       digest('6'),
		}},
	}}
	return document
}
