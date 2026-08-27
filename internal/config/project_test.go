package config

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/target/tsgo"
	"github.com/tsoniclang/gotots/internal/toolchain"
)

func TestLoadResolvesStrictProjectAndCLIOverrides(t *testing.T) {
	root := t.TempDir()
	configDirectory := filepath.Join(root, "project")
	writeProjectConfig(t, filepath.Join(configDirectory, "gotots.json"), `{
  "schemaVersion": 3,
  "distribution": {"root": "../distribution"},
  "source": {"root": "../source", "package": "./cmd/app", "mode": "main"},
  "go": {"goos": "linux", "goarch": "amd64", "cgo": false, "tags": ["noasm"]},
  "tools": `+testToolsDocument(t)+`,
  "semantics": {"integers": "number", "evaluationOrder": "direct"},
  "providers": {"standardLibrary": true, "externals": false},
  "implementations": {
    "packages": ["implementations/fast/contract.json"],
    "callables": ["implementations/hot/contract.json"]
  },
  "output": {"directory": "generated"}
}
`)
	integer := "fixed64-bigint"
	output := "alternate"
	toolCache := filepath.Join(".temp", "cache", "selected-tools")
	project, err := Load(Request{
		ConfigPath: filepath.Join(configDirectory, "gotots.json"),
		Overrides: Overrides{
			IntegerRepresentation: &integer,
			OutputDirectory:       &output,
			ToolCacheRoot:         &toolCache,
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
		project.IntegerRepresentation().String() != integer {
		t.Fatalf("resolved semantics are incorrect: %#v", project)
	}
	if got := project.BuildProfile().Tags(); !slices.Equal(got, []string{"noasm"}) {
		t.Fatalf("build tags = %v", got)
	}
	if !project.GoTool().Valid() || !project.TSGoTool().Valid() ||
		project.BuildProfile().ToolchainVersion() != project.GoTool().Version() {
		t.Fatal("resolved project did not retain one exact Go and TS-Go selection")
	}
	if project.ToolCacheRoot() != filepath.Join(configDirectory, toolCache) {
		t.Fatalf("tool cache = %q", project.ToolCacheRoot())
	}
	if got := project.PackageImplementations(); !slices.Equal(got, []string{
		filepath.Join(configDirectory, "implementations", "fast", "contract.json"),
	}) {
		t.Fatalf("package implementations = %v", got)
	}
	if got := project.CallableImplementations(); !slices.Equal(got, []string{
		filepath.Join(configDirectory, "implementations", "hot", "contract.json"),
	}) {
		t.Fatalf("callable implementations = %v", got)
	}
	if !project.StandardLibraryEnabled() || project.ExternalsEnabled() {
		t.Fatal("provider selection differs")
	}
}

func TestLoadDefaultsToolsAndBuildProfileFromSelectedGo(t *testing.T) {
	repository, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "gotots.json")
	writeProjectConfig(t, path, `{
  "schemaVersion": 3,
  "distribution": {"root": `+quoteJSON(t, repository)+`},
  "source": {"root": ".", "package": ".", "mode": "main"},
  "go": {"cgo": false, "tags": []},
  "semantics": {"integers": "number", "evaluationOrder": "direct"},
  "providers": {"standardLibrary": false, "externals": false},
  "implementations": {"packages": [], "callables": []},
  "output": {"directory": "generated"}
}
`)
	project, err := Load(Request{ConfigPath: path})
	if err != nil {
		t.Fatal(err)
	}
	if !project.GoTool().Valid() || !project.TSGoTool().Valid() {
		t.Fatal("default tool selection is invalid")
	}
	if project.BuildProfile().ToolchainVersion() != project.GoTool().Version() ||
		project.BuildProfile().GOOS() != project.GoTool().DefaultGOOS() ||
		project.BuildProfile().GOARCH() != project.GoTool().DefaultGOARCH() {
		t.Fatalf("default build profile did not come from selected Go: %#v", project.BuildProfile())
	}
}

func TestLoadRejectsCgoWithoutSelectedExternalToolContract(t *testing.T) {
	repository, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "gotots.json")
	writeProjectConfig(t, path, `{
  "schemaVersion": 3,
  "distribution": {"root": `+quoteJSON(t, repository)+`},
  "source": {"root": ".", "package": ".", "mode": "main"},
  "go": {"cgo": true, "tags": []},
  "semantics": {"integers": "number", "evaluationOrder": "direct"},
  "providers": {"standardLibrary": false, "externals": false},
  "implementations": {"packages": [], "callables": []},
  "output": {"directory": "generated"}
}
`)
	if _, err := Load(Request{ConfigPath: path}); err == nil ||
		!strings.Contains(err.Error(), "external-tool contract") {
		t.Fatalf("cgo selection error = %v", err)
	}
}

func TestLoadSelectsExternalImplementationContractsFromCLI(t *testing.T) {
	projectDirectory := t.TempDir()
	externalBundle := filepath.Join(t.TempDir(), "fast", "contract.json")
	path := filepath.Join(projectDirectory, "gotots.json")
	writeProjectConfig(t, path, `{
  "schemaVersion": 3,
  "distribution": {"root": "distribution"},
  "source": {"root": "source", "package": ".", "mode": "main"},
  "go": {"goos": "`+runtime.GOOS+`", "goarch": "`+runtime.GOARCH+`", "cgo": false, "tags": []},
  "tools": `+testToolsDocument(t)+`,
  "semantics": {"integers": "number", "evaluationOrder": "direct"},
  "providers": {"standardLibrary": false, "externals": false},
  "implementations": {
    "packages": ["local/package.json"],
    "callables": ["local/callable.json"]
  },
  "output": {"directory": "output"}
}
`)
	project, err := Load(Request{
		ConfigPath: path,
		Overrides: Overrides{
			PackageImplementationsSet:  true,
			PackageImplementations:     []string{externalBundle},
			CallableImplementationsSet: true,
			CallableImplementations:    []string{externalBundle + ".callable"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := project.PackageImplementations(); !slices.Equal(got, []string{externalBundle}) {
		t.Fatalf("package implementations = %v", got)
	}
	if got := project.CallableImplementations(); !slices.Equal(got, []string{externalBundle + ".callable"}) {
		t.Fatalf("callable implementations = %v", got)
	}
}

func TestLoadRejectsUnknownFieldAndVersion(t *testing.T) {
	for name, source := range map[string]string{
		"unknown": `{"schemaVersion":3,"surprise":true}`,
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
		Source:                  "source-digest",
		PackageImplementations:  "package-implementation-digest",
		CallableImplementations: "callable-implementation-digest",
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
		Source:                  "source-digest",
		PackageImplementations:  "changed",
		CallableImplementations: "callable-implementation-digest",
	})
	if err != nil {
		t.Fatal(err)
	}
	if changed == firstDigest {
		t.Fatal("implementation evidence did not change semantic digest")
	}
}

func TestSemanticDigestIncludesSelectedToolBuildIdentity(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture is Unix-only")
	}
	realGo, err := exec.LookPath("go")
	if err != nil {
		t.Fatal(err)
	}
	selectedGo, err := toolchain.ResolveGo(
		realGo,
		projectTestToolCache(t),
	)
	if err != nil {
		t.Fatal(err)
	}
	repository, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	selectedTSGo, err := tsgo.ResolveTool(selectedGo, repository, "")
	if err != nil {
		t.Fatal(err)
	}
	projects := make([]Project, 2)
	for index := range projects {
		directory := t.TempDir()
		wrapper := filepath.Join(directory, "selected-go")
		source := "#!/bin/sh\n# identity " + string(rune('a'+index)) + "\nexec " +
			configShellQuote(realGo) + " \"$@\"\n"
		if err := os.WriteFile(wrapper, []byte(source), 0o755); err != nil {
			t.Fatal(err)
		}
		projects[index] = loadProjectWithTools(
			t,
			directory,
			wrapper,
			selectedTSGo.Path(),
		)
	}
	evidence := EvidenceDigests{Source: "source-digest"}
	first, err := projects[0].SemanticDigest(evidence)
	if err != nil {
		t.Fatal(err)
	}
	second, err := projects[1].SemanticDigest(evidence)
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("different selected Go build identities produced one semantic digest")
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
  "schemaVersion": 3,
  "distribution": {"root": "distribution"},
  "source": {"root": "source", "package": ".", "mode": "main"},
  "go": {"goos": "`+runtime.GOOS+`", "goarch": "`+runtime.GOARCH+`", "cgo": false, "tags": []},
  "tools": `+testToolsDocument(t)+`,
  "semantics": {"integers": "number", "evaluationOrder": "direct"},
  "providers": {"standardLibrary": false, "externals": false},
  "implementations": {
    "packages": ["package-implementation.json"],
    "callables": ["callable-implementation.json"]
  },
  "output": {"directory": "`+output+`"}
}
`)
	project, err := Load(Request{ConfigPath: path})
	if err != nil {
		t.Fatal(err)
	}
	return project
}

func loadProjectWithTools(t *testing.T, directory, goPath, tsgoPath string) Project {
	t.Helper()
	path := filepath.Join(directory, "gotots.json")
	writeProjectConfig(t, path, `{
  "schemaVersion": 3,
  "distribution": {"root": "distribution"},
  "source": {"root": "source", "package": ".", "mode": "main"},
  "go": {"goos": "`+runtime.GOOS+`", "goarch": "`+runtime.GOARCH+`", "cgo": false, "tags": []},
  "tools": {"go": `+quoteJSON(t, goPath)+`, "tsgo": `+quoteJSON(t, tsgoPath)+`},
  "semantics": {"integers": "number", "evaluationOrder": "direct"},
  "providers": {"standardLibrary": false, "externals": false},
  "implementations": {"packages": [], "callables": []},
  "output": {"directory": "generated"}
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

func testToolsDocument(t *testing.T) string {
	t.Helper()
	goPath, err := exec.LookPath("go")
	if err != nil {
		t.Fatal(err)
	}
	selectedGo, err := toolchain.ResolveGo(
		goPath,
		projectTestToolCache(t),
	)
	if err != nil {
		t.Fatal(err)
	}
	repository, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	selectedTSGo, err := tsgo.ResolveTool(selectedGo, repository, "")
	if err != nil {
		t.Fatal(err)
	}
	return `{"go":` + quoteJSON(t, goPath) + `,"tsgo":` + quoteJSON(t, selectedTSGo.Path()) + `}`
}

func projectTestToolCache(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Join(root, ".temp", "cache", "toolchain-tests")
}

func quoteJSON(t *testing.T, value string) string {
	t.Helper()
	payload, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(payload)
}

func configShellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}
