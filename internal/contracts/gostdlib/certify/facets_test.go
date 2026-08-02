package certify

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
)

func TestFacetMapOwnsClosedGenericOperationSets(t *testing.T) {
	path := filepath.Join(t.TempDir(), "facets.json")
	payload := `{
	  "schemaVersion": 13,
  "facets": [],
  "genericOperationSets": [
    {
      "sourceIdentity": "slices|kind=4|receiver=|name=Grow",
      "operations": [
        {"kind":"zero","parameters":[],"results":[{"kind":"type-parameter","typeParameter":1}]},
        {"kind":"copy","parameters":[{"kind":"type-parameter","typeParameter":1}],"results":[{"kind":"type-parameter","typeParameter":1}]}
      ]
    }
  ]
}`
	if err := os.WriteFile(path, []byte(payload), 0o644); err != nil {
		t.Fatal(err)
	}
	seeds, err := readFacetSeeds(path)
	if err != nil {
		t.Fatal(err)
	}
	selected := seeds.genericOperations["slices|kind=4|receiver=|name=Grow"]
	if len(selected) != 2 ||
		selected[0].Kind != gostdlib.GenericOperationCopy ||
		selected[1].Kind != gostdlib.GenericOperationZero {
		t.Fatalf("generic operations = %#v", selected)
	}

	payload = `{"schemaVersion":13,"facets":[],"genericCallableProjections":[],"genericOperationSets":[
  {"sourceIdentity":"x","operations":[
    {"kind":"invented","parameters":[],"results":[{"kind":"type-parameter","typeParameter":0}]}
  ]}
]}`
	if err := os.WriteFile(path, []byte(payload), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := readFacetSeeds(path); err == nil {
		t.Fatal("open provider generic operation passed")
	}
}

func TestFacetMapOwnsClosedGenericCallableProjections(t *testing.T) {
	path := filepath.Join(t.TempDir(), "facets.json")
	payload := `{
	  "schemaVersion": 13,
  "facets": [],
  "genericCallableProjections": [
    {"sourceIdentity":"slices|kind=4|receiver=|name=Grow","typeArguments":[
      {"typeParameter":0,"facet":"logical"},
      {"typeParameter":1,"facet":"container-storage"}
    ]}
  ],
  "genericOperationSets": []
}`
	if err := os.WriteFile(path, []byte(payload), 0o644); err != nil {
		t.Fatal(err)
	}
	seeds, err := readFacetSeeds(path)
	if err != nil {
		t.Fatal(err)
	}
	seeds.genericProjections["slices|kind=4|receiver=|name=Grow"][0].TypeParameter = 9
	next, err := readFacetSeeds(path)
	if err != nil ||
		next.genericProjections["slices|kind=4|receiver=|name=Grow"][0].TypeParameter != 0 {
		t.Fatalf("projection source mutated: %#v, %v", next.genericProjections, err)
	}
	invalid := strings.Replace(payload, "container-storage", "invented", 1)
	if err := os.WriteFile(path, []byte(invalid), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := readFacetSeeds(path); err == nil {
		t.Fatal("open generic callable projection facet passed")
	}
}

func TestStatefulProfileSeparatesInterfaceSetFromTypeArgumentOrder(
	t *testing.T,
) {
	path := filepath.Join(t.TempDir(), "facets.json")
	payload := `{
	  "schemaVersion": 13,
  "facets": [],
  "providerStatefulProfiles": [{
    "sourceIdentity": "example.com/source|kind=2|receiver=|name=State",
    "specifier": "@gotots/gostdlib/internal/facets/provider-state.js",
    "sourcePath": "src/internal/facets/provider-state.ts",
    "export": "CanonicalState",
    "interfaces": [
      {"sourceIdentity":"example.com/a|kind=2|receiver=|name=A","export":"CanonicalA"},
      {"sourceIdentity":"example.com/b|kind=2|receiver=|name=B","export":"CanonicalB"}
    ],
    "typeArguments": [
      "example.com/b|kind=2|receiver=|name=B",
      "example.com/a|kind=2|receiver=|name=A"
    ]
  }]
}`
	if err := os.WriteFile(path, []byte(payload), 0o644); err != nil {
		t.Fatal(err)
	}
	seeds, err := readFacetSeeds(path)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"example.com/b|kind=2|receiver=|name=B",
		"example.com/a|kind=2|receiver=|name=A",
	}
	if !slices.Equal(seeds.statefulProfiles[0].TypeArguments, want) {
		t.Fatalf(
			"stateful type arguments = %v, want %v",
			seeds.statefulProfiles[0].TypeArguments,
			want,
		)
	}
	invalid := strings.Replace(
		payload,
		`,
      "example.com/a|kind=2|receiver=|name=A"`,
		"",
		1,
	)
	if err := os.WriteFile(path, []byte(invalid), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := readFacetSeeds(path); err == nil {
		t.Fatal("incomplete stateful type-argument mapping passed")
	}
}

func TestImplementedResultRequiresContractOwner(t *testing.T) {
	path := filepath.Join(t.TempDir(), "facets.json")
	payload := `{
  "schemaVersion": 13,
  "facets": [],
  "providerCallableProfiles": [{
    "sourceIdentity": "example.com/source|kind=4|receiver=|name=Build",
    "specifier": "@gotots/gostdlib/internal/facets/provider-build.js",
    "sourcePath": "src/internal/facets/provider-build.ts",
    "export": "BuildCanonical",
    "canonicalParameters": [0],
    "canonicalResults": [0],
    "implementedResultInterfaces": ["example.com/source|kind=2|receiver=|name=Result"],
    "interfaces": [{
      "sourceIdentity": "example.com/source|kind=2|receiver=|name=Result",
      "export": "CanonicalResult"
    }]
  }]
}`
	if err := os.WriteFile(path, []byte(payload), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := readFacetSeeds(path)
	if err == nil || !strings.Contains(
		err.Error(),
		"implemented result interface is not a contract interface",
	) {
		t.Fatalf("wrong implemented-result ownership error = %v", err)
	}
}

func TestGenericCallableProjectionRejectsProviderArityDrift(t *testing.T) {
	repository, err := filepath.Abs("../../../..")
	if err != nil {
		t.Fatal(err)
	}
	provider := filepath.Join(repository, "gostdlib")
	source, err := os.ReadFile(filepath.Join(provider, "contract", "facets.json"))
	if err != nil {
		t.Fatal(err)
	}
	mutated := bytes.Replace(
		source,
		[]byte(`{ "sourceIdentity": "slices|kind=4|receiver=|name=Concat", "typeArguments": [{"typeParameter":1,"facet":"container-storage"}] }`),
		[]byte(`{ "sourceIdentity": "slices|kind=4|receiver=|name=Concat", "typeArguments": [{"typeParameter":0,"facet":"logical"},{"typeParameter":1,"facet":"container-storage"}] }`),
		1,
	)
	if bytes.Equal(mutated, source) {
		t.Fatal("generic callable projection mutation was not applied")
	}
	facetMap := filepath.Join(t.TempDir(), "facets.json")
	if err := os.WriteFile(facetMap, mutated, 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = Generate(Config{
		RepositoryRoot:      repository,
		ProviderRoot:        provider,
		ManifestPath:        filepath.Join(provider, "contract", "manifest.json"),
		ModuleMapPath:       filepath.Join(provider, "contract", "modules.json"),
		FacetMapPath:        facetMap,
		RuntimeContractPath: filepath.Join(provider, "contract", "runtime.json"),
		TSConfigPath:        filepath.Join(provider, "tsconfig.json"),
		ScratchDirectory:    t.TempDir(),
		GoBinary:            "go",
		BuildProfile:        environmentcontract.DefaultBuildProfile(),
		Backend:             "node",
		MinimumGoVersion:    "go1.26.4",
		MaximumGoVersion:    "go1.26.4",
	})
	if err == nil || !strings.Contains(err.Error(), "provider callable has 1") {
		t.Fatalf("wrong generic callable projection error = %v", err)
	}
}

func TestGenericOperationRejectsCallableParameterArityDrift(t *testing.T) {
	repository, err := filepath.Abs("../../../..")
	if err != nil {
		t.Fatal(err)
	}
	provider := filepath.Join(repository, "gostdlib")
	source, err := os.ReadFile(filepath.Join(provider, "contract", "facets.json"))
	if err != nil {
		t.Fatal(err)
	}
	mutated := bytes.Replace(
		source,
		[]byte(`"kind": "interface-assert-ok",
          "parameters": [
            { "kind": "callable-parameter", "callableParameter": 0 }`),
		[]byte(`"kind": "interface-assert-ok",
          "parameters": [
            { "kind": "callable-parameter", "callableParameter": 9 }`),
		1,
	)
	if bytes.Equal(mutated, source) {
		t.Fatal("callable-parameter mutation was not applied")
	}
	facetMap := filepath.Join(t.TempDir(), "facets.json")
	if err := os.WriteFile(facetMap, mutated, 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = Generate(Config{
		RepositoryRoot:      repository,
		ProviderRoot:        provider,
		ManifestPath:        filepath.Join(provider, "contract", "manifest.json"),
		ModuleMapPath:       filepath.Join(provider, "contract", "modules.json"),
		FacetMapPath:        facetMap,
		RuntimeContractPath: filepath.Join(provider, "contract", "runtime.json"),
		TSConfigPath:        filepath.Join(provider, "tsconfig.json"),
		ScratchDirectory:    t.TempDir(),
		GoBinary:            "go",
		BuildProfile:        environmentcontract.DefaultBuildProfile(),
		Backend:             "node",
		MinimumGoVersion:    "go1.26.4",
		MaximumGoVersion:    "go1.26.4",
	})
	if err == nil || !strings.Contains(
		err.Error(),
		"callable-parameter index is outside its Go declaration",
	) {
		t.Fatalf("wrong callable-parameter error = %v", err)
	}
}

func TestRepresentationCertificationRejectsNonInterfaceSource(t *testing.T) {
	repository, err := filepath.Abs("../../../..")
	if err != nil {
		t.Fatal(err)
	}
	provider := filepath.Join(repository, "gostdlib")
	source, err := os.ReadFile(filepath.Join(provider, "contract", "facets.json"))
	if err != nil {
		t.Fatal(err)
	}
	mutated := bytes.Replace(
		source,
		[]byte(`"encoding/binary|kind=2|receiver=|name=ByteOrder"`),
		[]byte(`"runtime|kind=2|receiver=|name=MemStats"`),
		1,
	)
	if bytes.Equal(mutated, source) {
		t.Fatal("representation interface mutation was not applied")
	}
	facetMap := filepath.Join(t.TempDir(), "facets.json")
	if err := os.WriteFile(facetMap, mutated, 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = Generate(Config{
		RepositoryRoot:      repository,
		ProviderRoot:        provider,
		ManifestPath:        filepath.Join(provider, "contract", "manifest.json"),
		ModuleMapPath:       filepath.Join(provider, "contract", "modules.json"),
		FacetMapPath:        facetMap,
		RuntimeContractPath: filepath.Join(provider, "contract", "runtime.json"),
		TSConfigPath:        filepath.Join(provider, "tsconfig.json"),
		ScratchDirectory:    filepath.Join(t.TempDir(), "certify"),
		GoBinary:            "go",
		BuildProfile:        environmentcontract.DefaultBuildProfile(),
		Backend:             "node",
		MinimumGoVersion:    "go1.26.4",
		MaximumGoVersion:    "go1.26.4",
	})
	if err == nil || !strings.Contains(err.Error(), "not an interface") {
		t.Fatalf("wrong representation interface error = %v", err)
	}
}

func TestNamedStructFacetRejectsAbsentCapabilityMember(t *testing.T) {
	repository, err := filepath.Abs("../../../..")
	if err != nil {
		t.Fatal(err)
	}
	provider := filepath.Join(repository, "gostdlib")
	source, err := os.ReadFile(filepath.Join(provider, "contract", "facets.json"))
	if err != nil {
		t.Fatal(err)
	}
	mutated := bytes.Replace(
		source,
		[]byte(`"export": "IoFsPathErrorOperations",
      "capabilities": [
        "make",
        "storage"
      ]`),
		[]byte(`"export": "IoFsPathErrorOperations",
      "capabilities": [
        "hash",
        "storage"
      ]`),
		1,
	)
	if bytes.Equal(mutated, source) {
		t.Fatal("named-struct capability mutation was not applied")
	}
	facetMap := filepath.Join(t.TempDir(), "facets.json")
	if err := os.WriteFile(facetMap, mutated, 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = Generate(Config{
		RepositoryRoot:      repository,
		ProviderRoot:        provider,
		ManifestPath:        filepath.Join(provider, "contract", "manifest.json"),
		ModuleMapPath:       filepath.Join(provider, "contract", "modules.json"),
		FacetMapPath:        facetMap,
		RuntimeContractPath: filepath.Join(provider, "contract", "runtime.json"),
		TSConfigPath:        filepath.Join(provider, "tsconfig.json"),
		ScratchDirectory:    filepath.Join(t.TempDir(), "certify"),
		GoBinary:            "go",
		BuildProfile:        environmentcontract.DefaultBuildProfile(),
		Backend:             "node",
		MinimumGoVersion:    "go1.26.4",
		MaximumGoVersion:    "go1.26.4",
	})
	if err == nil ||
		!strings.Contains(err.Error(), "IoFsPathErrorOperations.$hash") ||
		!strings.Contains(err.Error(), "absent") {
		t.Fatalf("wrong absent capability error = %v", err)
	}
}
