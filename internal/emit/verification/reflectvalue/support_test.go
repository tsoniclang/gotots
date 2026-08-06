package reflectvalue_test

import (
	"context"
	"go/types"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
	"github.com/tsoniclang/gotots/internal/contracts/gostdlib/certify"
	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type renderedArtifacts struct {
	paths   []string
	printed string
}

func writeProgramFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func repositoryRoot() string {
	return filepath.Join("..", "..", "..", "..")
}

func mustRoot(t *testing.T, object types.Object) emit.Root {
	t.Helper()
	root, err := emit.NewRoot(object)
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func linkedProviderBuildProfile(t *testing.T) environmentcontract.BuildProfile {
	t.Helper()
	profile, err := environmentcontract.NewBuildProfile(
		"linux",
		"amd64",
		false,
		[]string{"noasm"},
	)
	if err != nil {
		t.Fatal(err)
	}
	return profile
}

func linkedProviderCertificate(t *testing.T) *certify.Certificate {
	t.Helper()
	repository := repositoryRoot()
	certificate, err := certify.Verify(certify.Config{
		RepositoryRoot: repository,
		ProviderRoot:   filepath.Join(repository, "gostdlib"),
		ManifestPath: filepath.Join(
			repository, "gostdlib", "contract", "manifest.json",
		),
		ModuleMapPath: filepath.Join(
			repository, "gostdlib", "contract", "modules.json",
		),
		FacetMapPath: filepath.Join(
			repository, "gostdlib", "contract", "facets.json",
		),
		RuntimeContractPath: filepath.Join(
			repository, "gostdlib", "contract", "runtime.json",
		),
		TSConfigPath:     filepath.Join(repository, "gostdlib", "tsconfig.json"),
		ScratchDirectory: t.TempDir(),
		GoBinary:         "go",
		BuildProfile:     linkedProviderBuildProfile(t),
		Backend:          "node",
		MinimumGoVersion: "go1.26.4",
		MaximumGoVersion: "go1.26.4",
	})
	if err != nil {
		t.Fatal(err)
	}
	return certificate
}

// compileReflectFixture loads one reflection fixture package, compiles it
// with the verified provider certificate under the cooperative profile, and
// returns the emission.
func compileReflectFixture(
	t *testing.T,
	project string,
	source string,
	roots []string,
) emit.ProgramEmission {
	t.Helper()
	writeProgramFile(
		t,
		filepath.Join(project, "go.mod"),
		"module example.com/reflectvalue\n\ngo 1.26.4\n",
	)
	writeProgramFile(t, filepath.Join(project, "source.go"), source)
	program, err := load.Load(context.Background(), load.Request{
		Directory:    project,
		Pattern:      ".",
		BuildProfile: linkedProviderBuildProfile(t),
	})
	if err != nil {
		t.Fatal(err)
	}
	scope := program.Roots()[0].Types().Scope()
	selected := make([]emit.Root, 0, len(roots))
	for _, name := range roots {
		selected = append(selected, mustRoot(t, scope.Lookup(name)))
	}
	options := emit.DefaultOptions()
	options.ConcurrencySemantics = emit.ConcurrencySemanticsCooperative
	options.StandardLibrary = linkedProviderCertificate(t)
	emission, err := emit.CompileWithOptions(program, selected, options)
	if err != nil {
		t.Fatal(err)
	}
	return emission
}

func materializeArtifacts(
	t *testing.T,
	emission emit.ProgramEmission,
	workingDirectory string,
) renderedArtifacts {
	t.Helper()
	client, err := tsgo.StartClient(repositoryRoot(), workingDirectory)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("close TS-Go client: %v", err)
		}
	})
	var result renderedArtifacts
	for _, file := range emission.Files() {
		printed, err := client.PrintNode(file.SourceFile(), tsgo.PrintOptions{})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := tsgo.EncodeSourceFile(file.SourceFile()); err != nil {
			t.Fatal(err)
		}
		for _, forbidden := range []string{
			": any",
			": unknown",
			" as any",
			" as unknown",
			".call(",
			".apply(",
			".bind(",
			"import(",
		} {
			if strings.Contains(printed, forbidden) {
				t.Fatalf(
					"%s contains forbidden %q:\n%s",
					file.OutputPath(),
					forbidden,
					printed,
				)
			}
		}
		targetPath := filepath.Join(
			workingDirectory,
			filepath.FromSlash(file.OutputPath()),
		)
		writeProgramFile(t, targetPath, printed)
		result.paths = append(result.paths, targetPath)
		result.printed += "\n// " + file.OutputPath() + "\n" + printed
	}
	if len(result.paths) == 0 {
		t.Fatal("reflection fixture emitted no target files")
	}
	return result
}

func waveThreeTypecheck(
	t *testing.T,
	workingDirectory string,
	paths []string,
) {
	t.Helper()
	writeProgramFile(
		t,
		filepath.Join(workingDirectory, "package.json"),
		"{\"type\":\"module\"}\n",
	)
	providerRoot, err := filepath.Abs(
		filepath.Join(repositoryRoot(), "gostdlib"),
	)
	if err != nil {
		t.Fatal(err)
	}
	packageRoot := filepath.Join(
		workingDirectory,
		"node_modules",
		"@gotots",
	)
	if err := os.MkdirAll(packageRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	providerPackage := filepath.Join(packageRoot, "gostdlib")
	if err := os.MkdirAll(providerPackage, 0o755); err != nil {
		t.Fatal(err)
	}
	packageDocument, err := os.ReadFile(
		filepath.Join(providerRoot, "package.json"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(providerPackage, "package.json"),
		packageDocument,
		0o644,
	); err != nil {
		t.Fatal(err)
	}
	if err := os.CopyFS(
		filepath.Join(providerPackage, "dist"),
		os.DirFS(filepath.Join(providerRoot, "dist")),
	); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(
		"../../runtime",
		filepath.Join(packageRoot, "runtime"),
	); err != nil {
		t.Fatal(err)
	}
	arguments := []string{
		"--target", "es2022",
		"--module", "nodenext",
		"--moduleResolution", "nodenext",
		"--strict",
		"--noUncheckedIndexedAccess",
		"--outDir", filepath.Join(workingDirectory, "out"),
	}
	arguments = append(arguments, paths...)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if err := tsgo.Compile(
		ctx,
		repositoryRoot(),
		workingDirectory,
		arguments,
	); err != nil {
		t.Fatal(err)
	}
}

func executeReflectTypeScript(
	t *testing.T,
	workingDirectory string,
	paths []string,
	assemblyPath string,
	exports []string,
	runnerBody string,
) string {
	t.Helper()
	waveThreeTypecheck(t, workingDirectory, paths)
	runnerPath := filepath.Join(workingDirectory, "runner.ts")
	modulePath := "./" + strings.TrimSuffix(assemblyPath, ".ts") + ".js"
	writeProgramFile(t, runnerPath, `import "./program.js";
import { `+strings.Join(exports, ", ")+` } from "`+modulePath+`";

`+runnerBody)
	outputDirectory := filepath.Join(workingDirectory, "out")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if err := tsgo.Compile(
		ctx,
		repositoryRoot(),
		workingDirectory,
		[]string{
			"--target", "es2022",
			"--module", "nodenext",
			"--moduleResolution", "nodenext",
			"--strict",
			"--noUncheckedIndexedAccess",
			"--outDir", outputDirectory,
			runnerPath,
		},
	); err != nil {
		t.Fatal(err)
	}
	runtimeLink := filepath.Join(
		workingDirectory,
		"node_modules",
		"@gotots",
		"runtime",
	)
	if err := os.Remove(runtimeLink); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(
		filepath.Join(outputDirectory, "runtime"),
		runtimeLink,
	); err != nil {
		t.Fatal(err)
	}
	runContext, runCancel := context.WithTimeout(
		context.Background(),
		2*time.Minute,
	)
	defer runCancel()
	command := exec.CommandContext(
		runContext,
		"node",
		filepath.Join(outputDirectory, "runner.js"),
	)
	command.Dir = workingDirectory
	command.Env = append(os.Environ(), "NODE_OPTIONS=--max-old-space-size=1024")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("execute generated reflection program: %v\n%s", err, output)
	}
	return string(output)
}

// runReflectDifferential compiles the fixture, executes the generated
// TypeScript and the native Go program, and exact-compares their output.
func runReflectDifferential(
	t *testing.T,
	source string,
	rootName string,
	packageName string,
	typescriptRunner string,
	goRunner string,
) {
	t.Helper()
	runReflectDifferentialInspect(
		t,
		source,
		rootName,
		packageName,
		typescriptRunner,
		goRunner,
		nil,
	)
}

func runReflectDifferentialInspect(
	t *testing.T,
	source string,
	rootName string,
	packageName string,
	typescriptRunner string,
	goRunner string,
	inspect func(renderedArtifacts),
) {
	t.Helper()
	runReflectDifferentialProjectInspect(
		t,
		source,
		rootName,
		packageName,
		typescriptRunner,
		goRunner,
		nil,
		inspect,
	)
}

func runReflectDifferentialProjectInspect(
	t *testing.T,
	source string,
	rootName string,
	packageName string,
	typescriptRunner string,
	goRunner string,
	prepare func(string),
	inspect func(renderedArtifacts),
) {
	t.Helper()
	project := t.TempDir()
	if prepare != nil {
		prepare(project)
	}
	emission := compileReflectFixture(t, project, source, []string{rootName})
	workingDirectory := t.TempDir()
	artifacts := materializeArtifacts(t, emission, workingDirectory)
	if inspect != nil {
		inspect(artifacts)
	}
	assemblyPath := ""
	for _, file := range emission.Files() {
		if file.Kind() == emit.TargetFilePackageAssembly &&
			file.PackageName() == packageName {
			assemblyPath = file.OutputPath()
			break
		}
	}
	if assemblyPath == "" {
		t.Fatal("reflection fixture package assembly is absent")
	}
	targetOutput := executeReflectTypeScript(
		t,
		workingDirectory,
		artifacts.paths,
		assemblyPath,
		[]string{rootName},
		typescriptRunner,
	)
	runnerDirectory := filepath.Join(project, "cmd", "compare")
	writeProgramFile(t, filepath.Join(runnerDirectory, "main.go"), goRunner)
	sourceContext, sourceCancel := context.WithTimeout(
		context.Background(),
		2*time.Minute,
	)
	defer sourceCancel()
	command := exec.CommandContext(sourceContext, "go", "run", ".")
	command.Dir = runnerDirectory
	sourceOutput, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("execute Go reflection comparison: %v\n%s", err, sourceOutput)
	}
	if targetOutput != string(sourceOutput) {
		t.Fatalf(
			"reflection differential:\nGo:\n%s\nTypeScript:\n%s",
			sourceOutput,
			targetOutput,
		)
	}
}
