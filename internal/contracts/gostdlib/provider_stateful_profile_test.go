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
			Fields: []gostdlib.ProviderStatefulProfileFieldDocument{{
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
	fields[0] = gostdlib.ProviderStatefulProfileField{}
	if _, ok := profile.Field("Value"); !ok {
		t.Fatal("stateful profile exposed mutable field storage")
	}
}
