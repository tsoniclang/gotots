package gostdlib_test

import (
	"bytes"
	"slices"
	"testing"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
)

const genericKernelIdentity = "slices|kind=4|receiver=|name=Equal"

func TestGenericCallableKernelRoundTripIsCanonicalAndImmutable(t *testing.T) {
	document := genericKernelDocument()
	payload, err := gostdlib.Seal(document)
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := gostdlib.Parse(payload)
	if err != nil {
		t.Fatal(err)
	}
	kernel, ok := manifest.GenericCallableKernel(genericKernelIdentity)
	if !ok || kernel.Kind() != gostdlib.FacetGenericCallableKernel ||
		kernel.ModuleSpecifier() !=
			"@gotots/gostdlib/internal/facets/generic-slices.js" ||
		kernel.Export() != "SlicesEqualKernel" ||
		kernel.Effect() != gostdlib.EffectSynchronous ||
		!slices.Equal(
			kernel.Capabilities(),
			[]gostdlib.FacetCapability{gostdlib.FacetCapabilityKernel},
		) {
		t.Fatalf("generic callable kernel = %#v, %t", kernel, ok)
	}
	projection := kernel.GenericTypeArguments()
	if !slices.Equal(projection, []gostdlib.GenericTypeArgumentDocument{{
		TypeParameter: 1,
		Facet:         gostdlib.GenericTypeArgumentLogical,
	}}) {
		t.Fatalf("generic callable kernel projection = %#v", projection)
	}
	projection[0].TypeParameter = 9
	if kernel.GenericTypeArguments()[0].TypeParameter != 1 {
		t.Fatal("generic callable kernel exposed mutable projection storage")
	}
	modules := manifest.FacetModules()
	moduleProjection := modules[0].Facets()[0].GenericTypeArguments()
	moduleProjection[0].TypeParameter = 9
	if manifest.FacetModules()[0].Facets()[0].
		GenericTypeArguments()[0].TypeParameter != 1 {
		t.Fatal("facet module exposed mutable kernel projection storage")
	}
	reencoded, err := gostdlib.Encode(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(reencoded, payload) {
		t.Fatal("generic callable kernel changed canonical round-trip bytes")
	}
}

func TestGenericCallableKernelRejectsMalformedProjection(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*gostdlib.FacetDocument)
	}{
		{
			name: "missing projection",
			mutate: func(facet *gostdlib.FacetDocument) {
				facet.GenericTypeArguments = nil
			},
		},
		{
			name: "duplicate projection",
			mutate: func(facet *gostdlib.FacetDocument) {
				facet.GenericTypeArguments = append(
					facet.GenericTypeArguments,
					facet.GenericTypeArguments[0],
				)
			},
		},
		{
			name: "wrong capability",
			mutate: func(facet *gostdlib.FacetDocument) {
				facet.Capabilities = []gostdlib.FacetCapability{
					gostdlib.FacetCapabilityRecovery,
				}
			},
		},
		{
			name: "projection on recovery facet",
			mutate: func(facet *gostdlib.FacetDocument) {
				facet.Kind = gostdlib.FacetRecoveryCallable
				facet.Capabilities = []gostdlib.FacetCapability{
					gostdlib.FacetCapabilityRecovery,
				}
				facet.Effect = gostdlib.EffectSynchronous
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			document := genericKernelDocument()
			test.mutate(&document.FacetModules[0].Facets[0])
			if _, err := gostdlib.Seal(document); err == nil {
				t.Fatal("malformed generic callable kernel passed")
			}
		})
	}
}

func genericKernelDocument() gostdlib.Document {
	document := validDocument()
	document.FacetModules = []gostdlib.FacetModuleDocument{{
		Specifier:  "@gotots/gostdlib/internal/facets/generic-slices.js",
		SourcePath: "src/internal/facets/generic-slices.ts",
		Facets: []gostdlib.FacetDocument{{
			Kind:           gostdlib.FacetGenericCallableKernel,
			SourceIdentity: genericKernelIdentity,
			Capabilities: []gostdlib.FacetCapability{
				gostdlib.FacetCapabilityKernel,
			},
			Export: "SlicesEqualKernel",
			Effect: gostdlib.EffectSynchronous,
			GenericTypeArguments: []gostdlib.GenericTypeArgumentDocument{{
				TypeParameter: 1,
				Facet:         gostdlib.GenericTypeArgumentLogical,
			}},
			ImplementationOwner: "src/internal/facets/generic-slices.ts",
			TargetFingerprint:   digest('d'),
		}},
	}}
	return document
}
