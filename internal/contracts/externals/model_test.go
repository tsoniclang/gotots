package externals

import (
	"encoding/json"
	"testing"
)

func TestManifestSealsCanonicalExternalBindings(t *testing.T) {
	document := testDocument()
	document.Bindings[0].TargetSignature =
		"func(element int32) int32|params=element|results="
	payload, err := Seal(document)
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := Parse(payload)
	if err != nil {
		t.Fatal(err)
	}
	if manifest.Digest() == "" || manifest.ProviderDigest() != digest('b') ||
		manifest.StandardLibraryDigest() != digest('a') ||
		manifest.ProviderIntegerRepresentation() != "bigint" {
		t.Fatalf("manifest metadata is incomplete: %#v", manifest)
	}
	bindings := manifest.Bindings()
	if len(bindings) != 2 ||
		bindings[0].SourceIdentity() != "example.com/native|kind=4|receiver=|name=Accelerated" ||
		bindings[0].TargetKind() != TargetSource ||
		bindings[1].SourceIdentity() != "example.com/native|kind=4|receiver=|name=Call" ||
		bindings[1].TargetKind() != TargetModule {
		t.Fatalf("bindings = %#v", bindings)
	}
	encoded, err := Encode(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if string(encoded) != string(payload) {
		t.Fatal("manifest encoding is not canonical")
	}
}

func TestManifestRejectsStaleOrAmbiguousEvidence(t *testing.T) {
	tests := map[string]func(*Document){
		"schema": func(document *Document) {
			document.SchemaVersion++
		},
		"standard library": func(document *Document) {
			document.StandardLibraryDigest = "stale"
		},
		"target profile": func(document *Document) {
			document.ProviderIntegerRepresentation = "mixed"
		},
		"binding order": func(document *Document) {
			document.Bindings[0], document.Bindings[1] =
				document.Bindings[1], document.Bindings[0]
		},
		"module source target": func(document *Document) {
			document.Bindings[1].TargetIdentity =
				"example.com/native|kind=4|receiver=|name=Portable"
		},
		"source module target": func(document *Document) {
			document.Bindings[0].ModuleSpecifier =
				"@gotots/externals/example.com/native.js"
		},
		"source target evidence": func(document *Document) {
			document.Bindings[0].TargetSignature = ""
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			document := testDocument()
			mutate(&document)
			if _, err := Seal(document); err == nil {
				t.Fatal("invalid external manifest was sealed")
			}
		})
	}
}

func TestManifestDigestRejectsPayloadMutation(t *testing.T) {
	payload, err := Seal(testDocument())
	if err != nil {
		t.Fatal(err)
	}
	var document Document
	if err := json.Unmarshal(payload, &document); err != nil {
		t.Fatal(err)
	}
	document.Bindings[1].TargetFingerprint = digest('c')
	mutated, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Parse(mutated); err == nil {
		t.Fatal("payload mutation preserved the manifest digest")
	}
}

func testDocument() Document {
	return Document{
		SchemaVersion:                 SchemaVersion,
		PackageName:                   PackageName,
		PackageVersion:                "0.0.0",
		Backend:                       "node",
		GoVersion:                     "go1.26.4",
		GOOS:                          "linux",
		GOARCH:                        "amd64",
		BuildTags:                     []string{"noasm"},
		ProviderIntegerRepresentation: "bigint",
		StandardLibraryDigest:         digest('a'),
		ProviderDigest:                digest('b'),
		Bindings: []BindingDocument{
			{
				SourceIdentity:      "example.com/native|kind=4|receiver=|name=Accelerated",
				SourceSignature:     "func(value int32) int32|params=value|results=",
				SourceModulePath:    "example.com/native",
				SourceModuleVersion: "v1.2.3",
				SourceLocation:      "native.go:8:1",
				TargetKind:          TargetSource,
				TargetIdentity:      "example.com/native|kind=4|receiver=|name=Portable",
				TargetSignature:     "func(value int32) int32|params=value|results=",
				TargetLocation:      "portable.go:3:1",
			},
			{
				SourceIdentity:      "example.com/native|kind=4|receiver=|name=Call",
				SourceSignature:     "func(value int32) int32|params=value|results=",
				SourceModulePath:    "example.com/native",
				SourceModuleVersion: "v1.2.3",
				SourceLocation:      "native.go:5:1",
				TargetKind:          TargetModule,
				ModuleSpecifier:     "@gotots/externals/example.com/native.js",
				Export:              "Call",
				ImplementationOwner: "src/example.com/native.ts",
				TargetFingerprint:   digest('d'),
			},
		},
	}
}

func digest(character byte) string {
	value := make([]byte, 64)
	for index := range value {
		value[index] = character
	}
	return string(value)
}
