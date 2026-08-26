package gostdlib_test

import (
	"testing"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
)

func TestGenericKernelFacetOwnsCallableContract(t *testing.T) {
	document := validDocument()
	document.FacetModules = []gostdlib.FacetModuleDocument{{
		Specifier:  "@gotots/gostdlib/internal/facets/generic.js",
		SourcePath: "src/internal/facets/generic.ts",
		Facets: []gostdlib.FacetDocument{{
			Kind:           gostdlib.FacetGenericCallableKernel,
			SourceIdentity: document.Modules[0].Bindings[0].Identity,
			Capabilities: []gostdlib.FacetCapability{
				gostdlib.FacetCapabilityKernel,
			},
			Export: "ContainsKernel",
			Effect: gostdlib.EffectSynchronous,
			CallableParameters: []gostdlib.ProviderCallableParameterDocument{{
				Parameter: 1,
				Effect:    gostdlib.EffectSynchronous,
			}},
			GenericTypeArguments: []gostdlib.GenericTypeArgumentDocument{{
				TypeParameter: 0,
				Facet:         gostdlib.GenericTypeArgumentLogical,
			}},
			ImplementationOwner: "src/internal/facets/generic.ts",
			TargetFingerprint:   digest('d'),
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
		document.Modules[0].Bindings[0].Identity,
		gostdlib.FacetGenericCallableKernel,
		gostdlib.FacetCapabilityKernel,
	)
	if !ok || facet.Effect() != gostdlib.EffectSynchronous {
		t.Fatalf("generic facet = %#v, %t", facet, ok)
	}
	parameters := facet.CallableParameters()
	parameters[0].Effect = gostdlib.EffectSynchronous
	if facet.CallableParameters()[0].Effect != gostdlib.EffectSynchronous {
		t.Fatal("generic facet exposed mutable callable-parameter storage")
	}

	document.FacetModules[0].Facets[0].Effect = gostdlib.EffectInvalid
	if _, err := gostdlib.Seal(document); err == nil {
		t.Fatal("generic facet without a callable effect passed")
	}
}
