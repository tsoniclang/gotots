package certify

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestProviderProtocolConfigurationMutationsFailClosed(t *testing.T) {
	repository, err := filepath.Abs("../../../..")
	if err != nil {
		t.Fatal(err)
	}
	payload, err := os.ReadFile(
		filepath.Join(repository, "gostdlib", "contract", "facets.json"),
	)
	if err != nil {
		t.Fatal(err)
	}
	var base facetMapDocument
	if err := json.Unmarshal(payload, &base); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name   string
		mutate func(*facetMapDocument)
		want   string
	}{
		{
			name: "required-selection",
			mutate: func(document *facetMapDocument) {
				document.ProviderCallableProfiles[0].Required = false
			},
			want: "required selection and semantic protocols disagree",
		},
		{
			name: "value-owner-absent",
			mutate: func(document *facetMapDocument) {
				document.ProviderCallableProfiles[0].Protocols[0].ValueParameter = nil
			},
			want: "protocol identity or export is invalid",
		},
		{
			name: "value-owner-outside-canonical-roots",
			mutate: func(document *facetMapDocument) {
				selected := 9
				document.ProviderCallableProfiles[0].Protocols[0].ValueParameter = &selected
			},
			want: "protocol value parameter is not a canonical root",
		},
		{
			name: "duplicate-method-set",
			mutate: func(document *facetMapDocument) {
				duplicate := document.ProviderProtocols[0]
				duplicate.Name = "error-is-duplicate"
				document.ProviderProtocols = append(document.ProviderProtocols, duplicate)
			},
			want: "method set duplicates protocol error-is",
		},
		{
			name: "callable-reference-outside-canonical-roots",
			mutate: func(document *facetMapDocument) {
				selected := 9
				document.ProviderProtocols[0].Protocol.Methods[0].Parameters[0].CallableParameter = &selected
			},
			want: "protocol callable parameter is not a canonical root",
		},
		{
			name: "unregistered-protocol",
			mutate: func(document *facetMapDocument) {
				document.ProviderCallableProfiles[0].Protocols[0].Protocol = "absent"
			},
			want: "protocol reference or export is invalid",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mutatedPayload, err := json.Marshal(base)
			if err != nil {
				t.Fatal(err)
			}
			var document facetMapDocument
			if err := json.Unmarshal(mutatedPayload, &document); err != nil {
				t.Fatal(err)
			}
			test.mutate(&document)
			mutatedPayload, err = json.Marshal(document)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(t.TempDir(), "facets.json")
			if err := os.WriteFile(path, mutatedPayload, 0o644); err != nil {
				t.Fatal(err)
			}
			_, err = readFacetSeeds(path)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("mutation error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestReflectionFacetSeedRequiresOneCertifiedResultExport(t *testing.T) {
	path := filepath.Join(t.TempDir(), "facets.json")
	valid := `{
  "schemaVersion": 23,
  "facets": [{
    "kind": "reflection-type-operations",
    "sourceIdentity": "reflect|kind=2|receiver=|name=Type",
    "capabilities": ["metadata"],
    "specifier": "@gotots/gostdlib/internal/facets/named-reflect.js",
    "sourcePath": "src/internal/facets/named-reflect.ts",
    "export": "ReflectTypeMetadataOperations",
    "resultExport": "RuntimeType"
  }],
  "genericOperationSets": []
}`
	if err := os.WriteFile(path, []byte(valid), 0o644); err != nil {
		t.Fatal(err)
	}
	seeds, err := readFacetSeeds(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(seeds.facets) != 1 || seeds.facets[0].ResultExport != "RuntimeType" {
		t.Fatalf("reflection result seed = %#v", seeds.facets)
	}
	missing := strings.Replace(
		valid,
		`    "resultExport": "RuntimeType"`,
		`    "resultExport": ""`,
		1,
	)
	if err := os.WriteFile(path, []byte(missing), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := readFacetSeeds(path); err == nil {
		t.Fatal("reflection facet without a concrete result export passed")
	}
}

func TestReflectionFacetResultExactJoinsConstructorReturn(t *testing.T) {
	repository, err := filepath.Abs("../../../..")
	if err != nil {
		t.Fatal(err)
	}
	provider := filepath.Join(repository, "gostdlib")
	_, selectedTSGo := resolveTestTools(t, repository)
	client, err := tsgo.StartClientWithTool(selectedTSGo, provider)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("close TS-Go client: %v", err)
		}
	})
	project, err := client.OpenProject(filepath.Join(provider, "tsconfig.json"))
	if err != nil {
		t.Fatal(err)
	}
	exports, err := project.Exports(filepath.Join(
		provider,
		"src/internal/facets/named-reflect.ts",
	))
	if err != nil {
		t.Fatal(err)
	}
	byName := make(map[string]tsgo.ProjectExport, len(exports))
	for _, selected := range exports {
		byName[selected.Name()] = selected
	}
	seed := facetSeed{
		Kind:           gostdlib.FacetReflectionTypeOperations,
		SourceIdentity: "reflect|kind=2|receiver=|name=Type",
		Export:         "ReflectTypeMetadataOperations",
		ResultExport:   "RuntimeType",
	}
	if err := validateFacetResultTarget(
		project,
		seed,
		byName[seed.Export],
		byName[seed.ResultExport],
	); err != nil {
		t.Fatal(err)
	}
	seed.ResultExport = "ReflectKindValueOperations"
	err = validateFacetResultTarget(
		project,
		seed,
		byName[seed.Export],
		byName[seed.ResultExport],
	)
	if err == nil || !strings.Contains(err.Error(), "does not return") {
		t.Fatalf("wrong reflection result mutation error = %v", err)
	}
}
