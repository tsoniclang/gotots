package function_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIntegerConstantsExecuteDifferentiallyThroughTsonic(t *testing.T) {
	tsonicRoot := selectedTsonicRoot(t)
	loaded := loadIntegerConstantsProject(t)
	workingDirectory := t.TempDir()
	targetFile := emitIntegerConstants(
		t,
		loaded,
		filepath.Join(workingDirectory, "src", "index.ts"),
	)
	printed := printTargetFile(t, targetFile, workingDirectory)

	sourceDirectory := filepath.Join(workingDirectory, "src")
	if err := os.MkdirAll(sourceDirectory, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(sourceDirectory, "index.ts"), printed)
	installTsonicTarget(t, workingDirectory, tsonicRoot)
	writeFile(t, filepath.Join(workingDirectory, "package.json"), `{
  "name": "gotots-integer-constants-proof",
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
  "entryPoint": "index.ts",
  "rootDir": "src",
  "outDir": "out",
  "targets": [
    {
      "id": "csharp",
      "options": {
        "namespace": "GoToTS.IntegerConstants",
        "assemblyName": "GoToTSIntegerConstants",
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
		"GoToTSIntegerConstants.csproj",
	)
	generatedSourcePath := filepath.Join(
		workingDirectory,
		"out",
		"csharp",
		"src",
		"Index.cs",
	)
	generatedSource, err := os.ReadFile(generatedSourcePath)
	if err != nil {
		t.Fatal(err)
	}
	generated := string(generatedSource)
	for _, exact := range []string{
		"return 42;",
		"return (2097152) * (4294967296) + (1);",
		"return (2147483647) * (4294967296) + (4294967295);",
		"return ((0) - (2147483647) - (1)) * (4294967296);",
	} {
		if !strings.Contains(generated, exact) {
			t.Fatalf("generated C# lacks %q:\n%s", exact, generated)
		}
	}
	if strings.Contains(generated, "9223372036854776000") {
		t.Fatalf("generated C# contains rounded maximum:\n%s", generated)
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
	writeFile(t, filepath.Join(runnerDirectory, "Program.cs"), `Console.WriteLine(GoToTS.IntegerConstants.Index.Small());
Console.WriteLine(GoToTS.IntegerConstants.Index.BeyondSafe());
Console.WriteLine(GoToTS.IntegerConstants.Index.Maximum());
Console.WriteLine(GoToTS.IntegerConstants.Index.Minimum());
`)
	runnerProject := filepath.Join(runnerDirectory, "Runner.csproj")
	run(t, runnerDirectory, "dotnet", "build", runnerProject, "--nologo", "--verbosity:quiet")
	targetOutput := run(
		t,
		runnerDirectory,
		"dotnet",
		filepath.Join(runnerDirectory, "bin", "Debug", "net10.0", "Runner.dll"),
	)
	goOutput := executeIntegerConstantsGo(t, workingDirectory)
	if targetOutput != goOutput {
		t.Fatalf("Tsonic/C# output = %q, Go output = %q", targetOutput, goOutput)
	}
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
	for name, target := range map[string]string{
		"csharp-js":      filepath.Join(parent, "csharp-js"),
		"csharp-runtime": filepath.Join(parent, "csharp-runtime"),
		"target-csharp":  filepath.Join(parent, "tsonic-csharp"),
	} {
		if _, err := os.Stat(filepath.Join(target, "package.json")); err != nil {
			t.Skipf("selected Tsonic dependency %s is unavailable: %v", name, err)
		}
		if err := os.Symlink(target, filepath.Join(scopeDirectory, name)); err != nil {
			t.Fatal(err)
		}
	}
}
