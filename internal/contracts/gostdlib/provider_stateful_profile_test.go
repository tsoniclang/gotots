package gostdlib_test

import (
	"slices"
	"testing"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
)

func TestStatefulProfilePreservesCertifiedTypeArgumentOrder(t *testing.T) {
	const (
		firstIdentity  = "example.com/a|kind=2|receiver=|name=A"
		secondIdentity = "example.com/b|kind=2|receiver=|name=B"
		stateIdentity  = "example.com/state|kind=2|receiver=|name=State"
	)
	method := func(
		identity string,
		member string,
		fingerprint byte,
	) gostdlib.ProviderInterfaceMethodDocument {
		return gostdlib.ProviderInterfaceMethodDocument{
			SourceIdentity:      identity,
			Kind:                gostdlib.ProviderInterfaceMethodCallable,
			Member:              member,
			Effect:              gostdlib.EffectSynchronous,
			SourceSignature:     "func()",
			ContractSignature:   "func()",
			SourceLocation:      "source.go:1:1",
			ImplementationOwner: "src/internal/facets/provider-state.ts",
			TargetFingerprint:   digest(fingerprint),
		}
	}
	firstMethod := method(firstIdentity+".First", "First", 'c')
	secondMethod := method(secondIdentity+".Second", "Second", 'd')
	key, err := gostdlib.BuildProviderCallableProfileKey(
		[]gostdlib.ProviderCallableProfileKeyInterface{
			{
				SourceIdentity: firstIdentity,
				Methods: []gostdlib.ProviderCallableProfileKeyMethod{{
					SourceIdentity: firstMethod.SourceIdentity,
					Effect:         firstMethod.Effect,
				}},
			},
			{
				SourceIdentity: secondIdentity,
				Methods: []gostdlib.ProviderCallableProfileKeyMethod{{
					SourceIdentity: secondMethod.SourceIdentity,
					Effect:         secondMethod.Effect,
				}},
			},
		},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	interfaces := []gostdlib.ProviderCallableProfileInterfaceDocument{
		{
			SourceIdentity: firstIdentity,
			Export:         "CanonicalA",
			ProviderInterface: gostdlib.ProviderInterfaceDocument{
				Mode:    gostdlib.ProviderInterfaceModeBridge,
				Methods: []gostdlib.ProviderInterfaceMethodDocument{firstMethod},
			},
			TargetFingerprint: digest('a'),
		},
		{
			SourceIdentity: secondIdentity,
			Export:         "CanonicalB",
			ProviderInterface: gostdlib.ProviderInterfaceDocument{
				Mode:    gostdlib.ProviderInterfaceModeBridge,
				Methods: []gostdlib.ProviderInterfaceMethodDocument{secondMethod},
			},
			TargetFingerprint: digest('b'),
		},
	}
	document := validDocument()
	document.FacetModules = []gostdlib.FacetModuleDocument{{
		Specifier:  "@gotots/gostdlib/internal/facets/provider-state.js",
		SourcePath: "src/internal/facets/provider-state.ts",
		StatefulProfiles: []gostdlib.ProviderStatefulProfileDocument{{
			SourceIdentity: stateIdentity,
			ProfileKey:     key,
			Export:         "CanonicalState",
			Interfaces:     interfaces,
			TypeArguments:  []string{secondIdentity, firstIdentity},
			Fields: []gostdlib.ProviderStructFieldDocument{{
				Member:              "Value",
				Ordinal:             1,
				SourceSignature:     "int",
				SourceLocation:      "source.go:2:1",
				ImplementationOwner: "src/internal/facets/provider-state.ts",
				TargetFingerprint:   digest('0'),
			}},
			Methods: []gostdlib.ProviderStatefulProfileMethodDocument{{
				SourceIdentity:            stateIdentity + ".Read",
				Member:                    "Read",
				Effect:                    gostdlib.EffectSynchronous,
				SourceSignature:           "func()",
				SourceLocation:            "source.go:2:1",
				ImplementationOwner:       "src/internal/facets/provider-state.ts",
				InstanceTargetFingerprint: digest('1'),
				StaticTargetFingerprint:   digest('2'),
			}},
			ImplementationOwner: "src/internal/facets/provider-state.ts",
			TargetFingerprint:   digest('3'),
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
	profile, ok := manifest.ProviderStatefulProfile(stateIdentity, key)
	if !ok {
		t.Fatal("stateful profile is absent")
	}
	want := []string{secondIdentity, firstIdentity}
	if !slices.Equal(profile.TypeArguments(), want) {
		t.Fatalf("type arguments = %v, want %v", profile.TypeArguments(), want)
	}
	mutated := profile.TypeArguments()
	mutated[0] = "mutated"
	if !slices.Equal(profile.TypeArguments(), want) {
		t.Fatal("stateful profile exposed mutable type-argument storage")
	}
	field, ok := profile.Field("Value")
	if !ok || field.Member() != "Value" || field.Ordinal() != 1 ||
		field.Embedded() || field.SourceSignature() != "int" ||
		field.SourceLocation() != "source.go:2:1" ||
		field.ImplementationOwner() != "src/internal/facets/provider-state.ts" ||
		field.TargetFingerprint() != digest('0') {
		t.Fatalf("stateful field = %#v", field)
	}
	fields := profile.Fields()
	fields[0] = gostdlib.ProviderStructField{}
	if _, ok := profile.Field("Value"); !ok {
		t.Fatal("stateful profile exposed mutable field storage")
	}
}

func TestDirectStructBindingOwnsImmutableFieldEvidence(t *testing.T) {
	document := validDocument()
	document.Modules[0].Bindings = []gostdlib.BindingDocument{{
		Identity:       "runtime|kind=2|receiver=|name=MemStats",
		Kind:           gostdlib.BindingType,
		Access:         gostdlib.AccessExport,
		Representation: gostdlib.RepresentationDirect,
		Export:         "MemStats",
		StructFields: []gostdlib.ProviderStructFieldDocument{
			{
				Member:              "Alloc",
				Ordinal:             0,
				SourceSignature:     "uint64",
				SourceLocation:      "runtime/mstats.go:52:2",
				ImplementationOwner: "src/internal/portable/runtime/mem-stats.ts",
				TargetFingerprint:   digest('d'),
			},
			{
				Member:              "Sys",
				Ordinal:             2,
				SourceSignature:     "uint64",
				SourceLocation:      "runtime/mstats.go:64:2",
				ImplementationOwner: "src/internal/portable/runtime/mem-stats.ts",
				TargetFingerprint:   digest('e'),
			},
		},
		SourceSignature:     "defined=runtime.MemStats|underlying=struct{...}",
		SourceLocation:      "runtime/mstats.go:48:1",
		ImplementationOwner: "src/internal/portable/runtime/mem-stats.ts",
		TargetFingerprint:   digest('f'),
	}}
	payload, err := gostdlib.Seal(document)
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := gostdlib.Parse(payload)
	if err != nil {
		t.Fatal(err)
	}
	binding, ok := manifest.Binding("runtime|kind=2|receiver=|name=MemStats")
	if !ok {
		t.Fatal("direct struct binding is absent")
	}
	field, ok := binding.StructField("Alloc")
	if !ok || field.Member() != "Alloc" || field.Ordinal() != 0 ||
		field.SourceSignature() != "uint64" ||
		field.SourceLocation() != "runtime/mstats.go:52:2" ||
		field.ImplementationOwner() !=
			"src/internal/portable/runtime/mem-stats.ts" ||
		field.TargetFingerprint() != digest('d') {
		t.Fatalf("direct struct field = %#v", field)
	}
	fields := binding.StructFields()
	fields[0] = gostdlib.ProviderStructField{}
	if _, ok := binding.StructField("Alloc"); !ok {
		t.Fatal("direct struct binding exposed mutable field storage")
	}

	unsorted := document
	unsorted.Modules = append([]gostdlib.ModuleDocument(nil), document.Modules...)
	unsorted.Modules[0].Bindings = append(
		[]gostdlib.BindingDocument(nil),
		document.Modules[0].Bindings...,
	)
	unsorted.Modules[0].Bindings[0].StructFields = []gostdlib.ProviderStructFieldDocument{
		document.Modules[0].Bindings[0].StructFields[1],
		document.Modules[0].Bindings[0].StructFields[0],
	}
	if _, err := gostdlib.Seal(unsorted); err == nil {
		t.Fatal("manifest accepted reordered direct struct-field evidence")
	}

	wrongOwner := document
	wrongOwner.Modules = append([]gostdlib.ModuleDocument(nil), document.Modules...)
	wrongOwner.Modules[0].Bindings = append(
		[]gostdlib.BindingDocument(nil),
		document.Modules[0].Bindings...,
	)
	wrongOwner.Modules[0].Bindings[0].Kind = gostdlib.BindingFunction
	wrongOwner.Modules[0].Bindings[0].Effect = gostdlib.EffectSynchronous
	wrongOwner.Modules[0].Bindings[0].Representation =
		gostdlib.RepresentationInvalid
	if _, err := gostdlib.Seal(wrongOwner); err == nil {
		t.Fatal("manifest attached struct-field evidence to a function")
	}
}
