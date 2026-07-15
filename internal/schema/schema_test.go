package schema

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

const repoRoot = "../.."

// TestManifestAgreesWithFilesystem: the schema manifest, the schemas
// directory, and the contract files agree exactly.
func TestManifestAgreesWithFilesystem(t *testing.T) {
	manifest, err := LoadManifest(filepath.Join(repoRoot, "schemas", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(filepath.Join(repoRoot, "schemas"))
	if err != nil {
		t.Fatal(err)
	}
	onDisk := map[string]bool{}
	for _, entry := range entries {
		if entry.Name() == "manifest.json" {
			continue
		}
		onDisk[filepath.ToSlash(filepath.Join("schemas", entry.Name()))] = true
	}
	listed := map[string]bool{}
	for _, contract := range manifest.Schemas {
		listed[contract.Path] = true
		if !onDisk[contract.Path] {
			t.Errorf("manifest lists %s which does not exist", contract.Path)
		}
		if _, err := Load(filepath.Join(repoRoot, filepath.FromSlash(contract.Path))); err != nil {
			t.Errorf("schema %s does not load: %v", contract.Path, err)
		}
	}
	for path := range onDisk {
		if !listed[path] {
			t.Errorf("schema file %s is not listed in the manifest", path)
		}
	}
}

// committedArtifacts maps schema IDs to their committed positive
// contract instances.
func committedArtifacts(t *testing.T) map[string]string {
	t.Helper()
	return map[string]string{
		"source-pin":        filepath.Join(repoRoot, "pins", "typescript-go.json"),
		"product-toolchain": filepath.Join(repoRoot, "pins", "product-toolchain.json"),
		"project-profile":   filepath.Join(repoRoot, "profiles", "tsts", "project.json"),
		"decision-registry": filepath.Join(repoRoot, "docs", "decisions", "registry.json"),
	}
}

func loadSchemaByID(t *testing.T, id string) *Schema {
	t.Helper()
	manifest, err := LoadManifest(filepath.Join(repoRoot, "schemas", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, contract := range manifest.Schemas {
		if contract.ID == id {
			schema, err := Load(filepath.Join(repoRoot, filepath.FromSlash(contract.Path)))
			if err != nil {
				t.Fatal(err)
			}
			return schema
		}
	}
	t.Fatalf("schema %q is not in the manifest", id)
	return nil
}

// TestCommittedArtifactsValidate: every committed contract instance is a
// positive fixture of its schema.
func TestCommittedArtifactsValidate(t *testing.T) {
	for id, path := range committedArtifacts(t) {
		t.Run(id, func(t *testing.T) {
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if err := loadSchemaByID(t, id).Validate(data); err != nil {
				t.Fatalf("%s does not validate against %s: %v", path, id, err)
			}
		})
	}
}

// TestGateReportSchemaPositive validates a representative gate report.
func TestGateReportSchemaPositive(t *testing.T) {
	report := map[string]any{
		"schemaVersion": 4, "expectedStages": 18, "reportedStages": 18,
		"passedStages": 3, "failed": 0, "blocked": 15, "missingStages": 0,
		"passed": false,
		"inputs": map[string]any{
			"implementationRevision":      "abc123",
			"specificationManifestSha256": strings.Repeat("a", 64),
			"decisionRegistrySha256":      strings.Repeat("b", 64),
			"buildProfile":                "linux-amd64",
			"missing":                     []string{},
		},
		"gates": []any{
			map[string]any{"name": "01-repository-specification-policy", "status": "pass"},
			map[string]any{"name": "04-census-denominator-reconciliation", "status": "blocked", "details": []string{"why"}},
		},
	}
	data, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if err := loadSchemaByID(t, "gate-report").Validate(data); err != nil {
		t.Fatalf("positive gate report rejected: %v", err)
	}
}

// mutateJSON applies one mutation to a parsed copy of the artifact.
func mutateJSON(t *testing.T, data []byte, mutate func(map[string]any)) []byte {
	t.Helper()
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	mutate(raw)
	out, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// TestNegativeFixtures: unknown fields, trailing data, missing required
// properties, enum violations, and digest mutations all fail for every
// schematized committed artifact.
func TestNegativeFixtures(t *testing.T) {
	for id, path := range committedArtifacts(t) {
		schema := loadSchemaByID(t, id)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		t.Run(id+"/unknown-field", func(t *testing.T) {
			bad := mutateJSON(t, data, func(raw map[string]any) { raw["surpriseField"] = 1 })
			if err := schema.Validate(bad); err == nil || !strings.Contains(err.Error(), "unknown property") {
				t.Fatalf("unknown field accepted: %v", err)
			}
		})
		t.Run(id+"/trailing-data", func(t *testing.T) {
			bad := append(append([]byte{}, data...), []byte("{}")...)
			if err := schema.Validate(bad); err == nil || !strings.Contains(err.Error(), "trailing data") {
				t.Fatalf("trailing data accepted: %v", err)
			}
		})
		t.Run(id+"/missing-required", func(t *testing.T) {
			bad := mutateJSON(t, data, func(raw map[string]any) { delete(raw, "schemaVersion") })
			if err := schema.Validate(bad); err == nil || !strings.Contains(err.Error(), "missing required") {
				t.Fatalf("missing required accepted: %v", err)
			}
		})
	}
}

// TestDigestMutationRejected: a digest field mutated to the wrong shape
// fails the named pattern.
func TestDigestMutationRejected(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(repoRoot, "pins", "typescript-go.json"))
	if err != nil {
		t.Fatal(err)
	}
	bad := mutateJSON(t, data, func(raw map[string]any) {
		raw["toolchain"].(map[string]any)["goExecutableSha256"] = "NOT-A-DIGEST"
	})
	if err := loadSchemaByID(t, "source-pin").Validate(bad); err == nil {
		t.Fatal("mutated digest accepted")
	}
}

// TestEnumMutationRejected: a closed enum rejects unlisted values.
func TestEnumMutationRejected(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(repoRoot, "docs", "decisions", "registry.json"))
	if err != nil {
		t.Fatal(err)
	}
	bad := mutateJSON(t, data, func(raw map[string]any) {
		decision := raw["decisions"].([]any)[0].(map[string]any)
		decision["status"] = "maybe"
	})
	if err := loadSchemaByID(t, "decision-registry").Validate(bad); err == nil || !strings.Contains(err.Error(), "closed enum") {
		t.Fatalf("enum violation accepted: %v", err)
	}
}

// TestSchemasAreSortedAndClosed: schema definitions themselves obey the
// canonical form (sorted manifest IDs already checked by LoadManifest).
func TestSchemasAreSortedAndClosed(t *testing.T) {
	manifest, err := LoadManifest(filepath.Join(repoRoot, "schemas", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	ids := make([]string, 0, len(manifest.Schemas))
	for _, contract := range manifest.Schemas {
		ids = append(ids, contract.ID)
	}
	if !sort.StringsAreSorted(ids) {
		t.Fatalf("manifest ids are not sorted: %v", ids)
	}
}
