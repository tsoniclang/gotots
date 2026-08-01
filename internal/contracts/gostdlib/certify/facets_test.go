package certify

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
)

func TestFacetMapOwnsClosedGenericOperationSets(t *testing.T) {
	path := filepath.Join(t.TempDir(), "facets.json")
	payload := `{
  "schemaVersion": 6,
  "facets": [],
  "genericOperationSets": [
    {
      "sourceIdentity": "slices|kind=4|receiver=|name=Grow",
      "operations": [
        {"kind":"zero","parameters":[],"results":[{"typeParameter":1}]},
        {"kind":"copy","parameters":[{"typeParameter":1}],"results":[{"typeParameter":1}]}
      ]
    }
  ]
}`
	if err := os.WriteFile(path, []byte(payload), 0o644); err != nil {
		t.Fatal(err)
	}
	_, _, _, _, operations, err := readFacetSeeds(path)
	if err != nil {
		t.Fatal(err)
	}
	selected := operations["slices|kind=4|receiver=|name=Grow"]
	if len(selected) != 2 ||
		selected[0].Kind != gostdlib.GenericOperationCopy ||
		selected[1].Kind != gostdlib.GenericOperationZero {
		t.Fatalf("generic operations = %#v", selected)
	}

	payload = `{"schemaVersion":6,"facets":[],"genericCallableProjections":[],"genericOperationSets":[
  {"sourceIdentity":"x","operations":[
    {"kind":"invented","parameters":[],"results":[{"typeParameter":0}]}
  ]}
]}`
	if err := os.WriteFile(path, []byte(payload), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, _, _, _, err := readFacetSeeds(path); err == nil {
		t.Fatal("open provider generic operation passed")
	}
}

func TestFacetMapOwnsClosedGenericCallableProjections(t *testing.T) {
	path := filepath.Join(t.TempDir(), "facets.json")
	payload := `{
  "schemaVersion": 6,
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
	_, _, _, projections, _, err := readFacetSeeds(path)
	if err != nil {
		t.Fatal(err)
	}
	projections["slices|kind=4|receiver=|name=Grow"][0].TypeParameter = 9
	_, _, _, next, _, err := readFacetSeeds(path)
	if err != nil ||
		next["slices|kind=4|receiver=|name=Grow"][0].TypeParameter != 0 {
		t.Fatalf("projection source mutated: %#v, %v", next, err)
	}
	invalid := strings.Replace(payload, "container-storage", "invented", 1)
	if err := os.WriteFile(path, []byte(invalid), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, _, _, _, err := readFacetSeeds(path); err == nil {
		t.Fatal("open generic callable projection facet passed")
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
		Backend:             "node",
		MinimumGoVersion:    "go1.26.4",
		MaximumGoVersion:    "go1.26.4",
	})
	if err == nil || !strings.Contains(err.Error(), "provider callable has 1") {
		t.Fatalf("wrong generic callable projection error = %v", err)
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
