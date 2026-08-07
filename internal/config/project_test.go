package config

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"testing"
)

func TestLoadResolvesStrictProjectAndCLIOverrides(t *testing.T) {
	root := t.TempDir()
	configDirectory := filepath.Join(root, "project")
	writeProjectConfig(t, filepath.Join(configDirectory, "gotots.json"), `{
  "schemaVersion": 1,
  "distribution": {"root": "../distribution"},
  "source": {"root": "../source", "package": "./cmd/app", "mode": "main"},
  "go": {"goos": "linux", "goarch": "amd64", "cgo": false, "tags": ["noasm"]},
  "semantics": {"integers": "number", "evaluationOrder": "direct", "concurrency": "disabled"},
  "providers": {"standardLibrary": true, "externals": false},
  "implementations": {"bundles": ["implementations/fast/contract.json"]},
  "output": {"directory": "generated"}
}
`)
	integer := "fixed64-bigint"
	concurrency := "cooperative"
	output := "alternate"
	project, err := Load(Request{
		ConfigPath: filepath.Join(configDirectory, "gotots.json"),
		Overrides: Overrides{
			IntegerRepresentation: &integer,
			ConcurrencySemantics:  &concurrency,
			OutputDirectory:       &output,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if project.SourceRoot() != filepath.Join(root, "source") ||
		project.DistributionRoot() != filepath.Join(root, "distribution") ||
		project.OutputDirectory() != filepath.Join(configDirectory, "alternate") {
		t.Fatalf("resolved paths = %q, %q, %q", project.SourceRoot(), project.DistributionRoot(), project.OutputDirectory())
	}
	if project.PackagePattern() != "./cmd/app" || project.RootMode() != RootModeMain ||
		project.IntegerRepresentation().String() != integer ||
		project.ConcurrencySemantics().String() != concurrency {
		t.Fatalf("resolved semantics are incorrect: %#v", project)
	}
	if got := project.BuildProfile().Tags(); !slices.Equal(got, []string{"noasm"}) {
		t.Fatalf("build tags = %v", got)
	}
	if got := project.ImplementationBundles(); !slices.Equal(got, []string{
		filepath.Join(configDirectory, "implementations", "fast", "contract.json"),
	}) {
		t.Fatalf("implementation bundles = %v", got)
	}
	if !project.StandardLibraryEnabled() || project.ExternalsEnabled() {
		t.Fatal("provider selection differs")
	}
}

func TestLoadSelectsExternalImplementationBundlesFromCLI(t *testing.T) {
	projectDirectory := t.TempDir()
	externalBundle := filepath.Join(t.TempDir(), "fast", "contract.json")
	path := filepath.Join(projectDirectory, "gotots.json")
	writeProjectConfig(t, path, `{
  "schemaVersion": 1,
  "distribution": {"root": "distribution"},
  "source": {"root": "source", "package": ".", "mode": "main"},
  "go": {"goos": "`+runtime.GOOS+`", "goarch": "`+runtime.GOARCH+`", "cgo": false, "tags": []},
  "semantics": {"integers": "number", "evaluationOrder": "direct", "concurrency": "disabled"},
  "providers": {"standardLibrary": false, "externals": false},
  "implementations": {"bundles": ["local/contract.json"]},
  "output": {"directory": "output"}
}
`)
	project, err := Load(Request{
		ConfigPath: path,
		Overrides: Overrides{
			ImplementationSet:     true,
			ImplementationBundles: []string{externalBundle},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := project.ImplementationBundles(); !slices.Equal(got, []string{externalBundle}) {
		t.Fatalf("implementation bundles = %v", got)
	}
}

func TestLoadRejectsUnknownFieldAndVersion(t *testing.T) {
	for name, source := range map[string]string{
		"unknown": `{"schemaVersion":1,"surprise":true}`,
		"version": `{"schemaVersion":2}`,
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "gotots.json")
			writeProjectConfig(t, path, source)
			if _, err := Load(Request{ConfigPath: path}); err == nil {
				t.Fatal("invalid project config was accepted")
			}
		})
	}
}

func TestSemanticDigestIgnoresOperationalRelocation(t *testing.T) {
	first := loadMinimalProject(t, filepath.Join(t.TempDir(), "one"), "out-a")
	second := loadMinimalProject(t, filepath.Join(t.TempDir(), "two"), "out-b")
	evidence := EvidenceDigests{
		Source:                "source-digest",
		SourceImplementations: "implementation-digest",
	}
	firstDigest, err := first.SemanticDigest(evidence)
	if err != nil {
		t.Fatal(err)
	}
	secondDigest, err := second.SemanticDigest(evidence)
	if err != nil {
		t.Fatal(err)
	}
	if firstDigest != secondDigest {
		t.Fatalf("relocation changed semantic digest: %s != %s", firstDigest, secondDigest)
	}
	changed, err := second.SemanticDigest(EvidenceDigests{
		Source:                "source-digest",
		SourceImplementations: "changed",
	})
	if err != nil {
		t.Fatal(err)
	}
	if changed == firstDigest {
		t.Fatal("implementation evidence did not change semantic digest")
	}
}

func TestCanonicalResolvedConfigRoundTrips(t *testing.T) {
	original := loadMinimalProject(t, filepath.Join(t.TempDir(), "original"), "output")
	payload, err := original.CanonicalJSON()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "resolved.json")
	writeProjectConfig(t, path, string(payload))
	roundTripped, err := Load(Request{ConfigPath: path})
	if err != nil {
		t.Fatal(err)
	}
	if roundTripped.SourceRoot() != original.SourceRoot() ||
		roundTripped.DistributionRoot() != original.DistributionRoot() ||
		roundTripped.OutputDirectory() != original.OutputDirectory() ||
		roundTripped.PackagePattern() != original.PackagePattern() ||
		roundTripped.RootMode() != original.RootMode() {
		t.Fatalf("round-tripped project differs: %#v != %#v", roundTripped, original)
	}
}

func TestOptionRegistryCoversEveryProjectField(t *testing.T) {
	want := documentPaths(reflect.TypeOf(document{}), "")
	descriptors := Descriptors()
	got := make([]string, len(descriptors))
	flags := make(map[string]struct{}, len(descriptors))
	for index, descriptor := range descriptors {
		got[index] = descriptor.JSONPath()
		if descriptor.Flag() == "" {
			t.Fatalf("%s has no CLI flag", descriptor.JSONPath())
		}
		if _, duplicate := flags[descriptor.Flag()]; duplicate {
			t.Fatalf("duplicate CLI flag %q", descriptor.Flag())
		}
		flags[descriptor.Flag()] = struct{}{}
	}
	if !slices.Equal(got, want) {
		t.Fatalf("descriptor JSON paths = %v, want %v", got, want)
	}
}

func documentPaths(selected reflect.Type, prefix string) []string {
	var result []string
	for index := range selected.NumField() {
		field := selected.Field(index)
		name := field.Tag.Get("json")
		if name == "schemaVersion" {
			continue
		}
		path := name
		if prefix != "" {
			path = prefix + "." + name
		}
		fieldType := field.Type
		if fieldType.Kind() == reflect.Struct {
			result = append(result, documentPaths(fieldType, path)...)
			continue
		}
		result = append(result, path)
	}
	slices.Sort(result)
	return result
}

func loadMinimalProject(t *testing.T, directory string, output string) Project {
	t.Helper()
	path := filepath.Join(directory, "gotots.json")
	writeProjectConfig(t, path, `{
  "schemaVersion": 1,
  "distribution": {"root": "distribution"},
  "source": {"root": "source", "package": ".", "mode": "main"},
  "go": {"goos": "`+runtime.GOOS+`", "goarch": "`+runtime.GOARCH+`", "cgo": false, "tags": []},
  "semantics": {"integers": "number", "evaluationOrder": "direct", "concurrency": "disabled"},
  "providers": {"standardLibrary": false, "externals": false},
  "implementations": {"bundles": ["implementation.json"]},
  "output": {"directory": "`+output+`"}
}
`)
	project, err := Load(Request{ConfigPath: path})
	if err != nil {
		t.Fatal(err)
	}
	return project
}

func writeProjectConfig(t *testing.T, path string, source string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
}
