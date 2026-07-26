package function_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type tsonicProof struct {
	namespace       string
	assembly        string
	runnerSource    string
	requiredTarget  []string
	forbiddenTarget []string
}

func executeThroughTsonic(
	t *testing.T,
	printedSource string,
	proof tsonicProof,
) string {
	t.Helper()
	return executeFilesThroughTsonic(
		t,
		map[string]string{"index.ts": printedSource},
		"index.ts",
		proof,
	)
}

func executeFilesThroughTsonic(
	t *testing.T,
	printedSources map[string]string,
	entryPoint string,
	proof tsonicProof,
) string {
	t.Helper()
	tsonicRoot := selectedTsonicRoot(t)
	workingDirectory := t.TempDir()
	sourceDirectory := filepath.Join(workingDirectory, "src")
	if err := os.MkdirAll(sourceDirectory, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, source := range printedSources {
		writeFile(t, filepath.Join(sourceDirectory, name), source)
	}
	installTsonicTarget(t, workingDirectory, tsonicRoot)
	writeFile(t, filepath.Join(workingDirectory, "package.json"), `{
  "name": "gotots-consumer-proof",
  "private": true,
  "type": "module",
  "dependencies": {
    "@tsonic/csharp-js": "0.0.1",
    "@tsonic/csharp-runtime": "0.0.1",
    "@tsonic/target-csharp": "0.0.1"
  }
}
`)
	writeFile(t, filepath.Join(workingDirectory, "tsonic.json"), `{
  "entryPoint": "`+entryPoint+`",
  "rootDir": "src",
  "outDir": "out",
  "targets": [
    {
      "id": "csharp",
      "options": {
        "namespace": "`+proof.namespace+`",
        "assemblyName": "`+proof.assembly+`",
        "targetFramework": "net10.0",
        "outputType": "Exe"
      }
    }
  ]
}
`)

	cliPath := filepath.Join(tsonicRoot, "packages", "cli", "dist", "src", "index.js")
	run(t, workingDirectory, "node", cliPath, "build", "--project", "tsonic.json")
	generatedProject := filepath.Join(
		workingDirectory,
		"out",
		"csharp",
		proof.assembly+".csproj",
	)
	generatedSourceDirectory := filepath.Join(
		workingDirectory,
		"out",
		"csharp",
		"src",
	)
	generatedPaths, err := filepath.Glob(filepath.Join(generatedSourceDirectory, "*.cs"))
	if err != nil {
		t.Fatal(err)
	}
	if len(generatedPaths) == 0 {
		t.Fatal("Tsonic generated no C# source files")
	}
	var generated strings.Builder
	for _, generatedPath := range generatedPaths {
		generatedSource, err := os.ReadFile(generatedPath)
		if err != nil {
			t.Fatal(err)
		}
		generated.Write(generatedSource)
	}
	generatedText := generated.String()
	for _, required := range proof.requiredTarget {
		if !strings.Contains(generatedText, required) {
			t.Fatalf("generated C# lacks %q:\n%s", required, generatedText)
		}
	}
	for _, forbidden := range proof.forbiddenTarget {
		if strings.Contains(generatedText, forbidden) {
			t.Fatalf("generated C# contains forbidden %q:\n%s", forbidden, generatedText)
		}
	}
	run(t, workingDirectory, "dotnet", "build", generatedProject, "--nologo", "--verbosity:quiet")

	runnerDirectory := filepath.Join(workingDirectory, "runner")
	if err := os.MkdirAll(runnerDirectory, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(runnerDirectory, "Runner.csproj"), `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <OutputType>Exe</OutputType>
    <TargetFramework>net10.0</TargetFramework>
    <ImplicitUsings>enable</ImplicitUsings>
    <Nullable>enable</Nullable>
  </PropertyGroup>
  <ItemGroup>
    <ProjectReference Include="`+filepath.ToSlash(generatedProject)+`" />
  </ItemGroup>
</Project>
`)
	writeFile(t, filepath.Join(runnerDirectory, "Program.cs"), proof.runnerSource)
	runnerProject := filepath.Join(runnerDirectory, "Runner.csproj")
	run(t, runnerDirectory, "dotnet", "build", runnerProject, "--nologo", "--verbosity:quiet")
	return run(
		t,
		runnerDirectory,
		"dotnet",
		filepath.Join(runnerDirectory, "bin", "Debug", "net10.0", "Runner.dll"),
	)
}

func selectedTsonicRoot(t *testing.T) string {
	t.Helper()
	root := os.Getenv("GOTOTS_TSONIC_ROOT")
	if root == "" {
		repository, err := filepath.Abs(repositoryRoot())
		if err != nil {
			t.Fatal(err)
		}
		root = filepath.Join(filepath.Dir(repository), "tsonic")
	}
	cliPath := filepath.Join(root, "packages", "cli", "dist", "src", "index.js")
	if _, err := os.Stat(cliPath); err != nil {
		t.Skipf("selected Tsonic CLI is unavailable: %v", err)
	}
	revision := strings.TrimSpace(run(t, root, "git", "rev-parse", "HEAD"))
	t.Logf("selected Tsonic revision: %s", revision)
	return root
}

func installTsonicTarget(t *testing.T, workingDirectory, tsonicRoot string) {
	t.Helper()
	scopeDirectory := filepath.Join(workingDirectory, "node_modules", "@tsonic")
	if err := os.MkdirAll(scopeDirectory, 0o755); err != nil {
		t.Fatal(err)
	}
	parent := filepath.Dir(tsonicRoot)
	csharpTargetRoot := os.Getenv("GOTOTS_TSONIC_CSHARP_ROOT")
	if csharpTargetRoot == "" {
		csharpTargetRoot = filepath.Join(parent, "tsonic-csharp")
	}
	for name, target := range map[string]string{
		"csharp-js":      filepath.Join(parent, "csharp-js"),
		"csharp-runtime": filepath.Join(parent, "csharp-runtime"),
		"target-csharp":  csharpTargetRoot,
	} {
		if _, err := os.Stat(filepath.Join(target, "package.json")); err != nil {
			t.Skipf("selected Tsonic dependency %s is unavailable: %v", name, err)
		}
		revision := strings.TrimSpace(run(t, target, "git", "rev-parse", "HEAD"))
		t.Logf("selected %s revision: %s", name, revision)
		if err := os.Symlink(target, filepath.Join(scopeDirectory, name)); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("selected dotnet version: %s", strings.TrimSpace(run(t, workingDirectory, "dotnet", "--version")))
}
