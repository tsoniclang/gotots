package gostdlib_test

import (
	"testing"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
)

func TestManifestOwnsReflectionFacetConcreteResultEvidence(t *testing.T) {
	document := validDocument()
	document.FacetModules = []gostdlib.FacetModuleDocument{{
		Specifier:  "@gotots/gostdlib/internal/facets/named-reflect.js",
		SourcePath: "src/internal/facets/named-reflect.ts",
		Facets: []gostdlib.FacetDocument{{
			Kind:                      gostdlib.FacetReflectionTypeOperations,
			SourceIdentity:            "reflect|kind=2|receiver=|name=Type",
			Capabilities:              []gostdlib.FacetCapability{gostdlib.FacetCapabilityMetadata},
			Export:                    "ReflectTypeMetadataOperations",
			ResultExport:              "RuntimeType",
			ImplementationOwner:       "src/internal/facets/named-reflect.ts",
			ResultImplementationOwner: "src/internal/portable/reflect/runtime-type.ts",
			TargetFingerprint:         digest('d'),
			ResultTargetFingerprint:   digest('e'),
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
		"reflect|kind=2|receiver=|name=Type",
		gostdlib.FacetReflectionTypeOperations,
		gostdlib.FacetCapabilityMetadata,
	)
	if !ok || facet.ResultExport() != "RuntimeType" ||
		facet.ResultImplementationOwner() !=
			"src/internal/portable/reflect/runtime-type.ts" ||
		facet.ResultTargetFingerprint() != digest('e') {
		t.Fatalf("reflection result evidence = %#v, %t", facet, ok)
	}

	mutations := []func(*gostdlib.FacetDocument){
		func(selected *gostdlib.FacetDocument) { selected.ResultExport = "" },
		func(selected *gostdlib.FacetDocument) {
			selected.ResultImplementationOwner = ""
		},
		func(selected *gostdlib.FacetDocument) {
			selected.ResultTargetFingerprint = ""
		},
	}
	for index, mutate := range mutations {
		changed := document
		changed.FacetModules = []gostdlib.FacetModuleDocument{{
			Specifier:  document.FacetModules[0].Specifier,
			SourcePath: document.FacetModules[0].SourcePath,
			Facets: append(
				[]gostdlib.FacetDocument(nil),
				document.FacetModules[0].Facets...,
			),
		}}
		mutate(&changed.FacetModules[0].Facets[0])
		if _, err := gostdlib.Seal(changed); err == nil {
			t.Fatalf("reflection result evidence mutation %d passed", index)
		}
	}
}
