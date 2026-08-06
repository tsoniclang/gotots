package emit_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
)

func TestCrossPackageGenericLocalTypeConcretizesAtCaller(t *testing.T) {
	directory := filepath.Join(
		repositoryRoot(),
		"testdata",
		"constructs",
		"generic",
		"cross-package-local",
	)
	directory, err := filepath.Abs(directory)
	if err != nil {
		t.Fatal(err)
	}
	program, err := load.Load(context.Background(), load.Request{
		Directory: directory,
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	root, err := emit.NewRoot(
		program.Roots()[0].Types().Scope().Lookup("Audit"),
	)
	if err != nil {
		t.Fatal(err)
	}
	emission, err := emit.Compile(program, []emit.Root{root})
	if err != nil {
		t.Fatal(err)
	}
	workingDirectory := t.TempDir()
	artifacts := materializeArtifacts(t, emission, workingDirectory)

	var callerSource string
	var providerSource string
	for _, file := range emission.Files() {
		if strings.Contains(
			file.OutputPath(),
			"support/generics/concretizations/",
		) {
			t.Fatalf(
				"caller-local concretization escaped to compilation scope: %s",
				file.OutputPath(),
			)
		}
		if file.Kind() != emit.TargetFileSource {
			continue
		}
		content, readErr := os.ReadFile(filepath.Join(
			workingDirectory,
			filepath.FromSlash(file.OutputPath()),
		))
		if readErr != nil {
			t.Fatal(readErr)
		}
		switch file.PackageName() {
		case "crosslocal":
			callerSource = string(content)
		case "api":
			providerSource = string(content)
		}
	}
	for fragment, source := range map[string]string{
		"function Twice$concrete_":        callerSource,
		"function BothEqual$concrete_":    callerSource,
		"enum Local":                      callerSource,
		"enum Outer":                      callerSource,
		"enum Inner":                      callerSource,
		"function $goCapability_":         callerSource,
		"function Twice$kernel<T>":        providerSource,
		"function BothEqual$kernel<A, B>": providerSource,
	} {
		if !strings.Contains(source, fragment) {
			t.Fatalf("source lacks %q:\n%s", fragment, source)
		}
	}
	if count := strings.Count(
		callerSource,
		"function BothEqual$concrete_",
	); count != 1 {
		t.Fatalf("same lexical generic instance emitted %d wrappers, want 1", count)
	}
	inner := strings.Index(callerSource, "enum Inner")
	wrapper := strings.Index(callerSource, "function BothEqual$concrete_")
	if inner < 0 || wrapper < inner {
		t.Fatalf(
			"multi-component wrapper was not placed after its deepest type:\n%s",
			callerSource,
		)
	}
	if strings.Contains(providerSource, "crosslocal") ||
		strings.Contains(providerSource, "Twice$concrete_") {
		t.Fatalf(
			"generic provider acquired a caller-local reverse dependency:\n%s",
			providerSource,
		)
	}

	runner := filepath.Join(workingDirectory, "runner.ts")
	writeProgramFile(t, runner, `import "./program.js";
import { Audit } from "`+sourceModuleForExport(
		t,
		artifacts,
		workingDirectory,
		"Audit",
	)+`";

console.log(String(Audit()));
`)
	writeProgramFile(
		t,
		filepath.Join(workingDirectory, "package.json"),
		"{\"type\":\"module\"}\n",
	)
	waveThreeTypecheck(t, workingDirectory, append(artifacts.paths, runner))
	targetOutput := runProgram(
		t,
		workingDirectory,
		"node",
		filepath.Join(workingDirectory, "out", "runner.js"),
	)
	goRunner := filepath.Join(workingDirectory, "go-runner")
	writeProgramFile(t, filepath.Join(goRunner, "go.mod"), fmt.Sprintf(
		`module example.com/runner

go 1.26.4

require example.com/crosslocal v0.0.0

replace example.com/crosslocal => %s
`,
		filepath.ToSlash(directory),
	))
	writeProgramFile(t, filepath.Join(goRunner, "main.go"), `package main

import (
	"fmt"
	"example.com/crosslocal"
)

func main() {
	fmt.Println(crosslocal.Audit())
}
`)
	goOutput := runProgram(
		t,
		goRunner,
		filepath.Join(runtime.GOROOT(), "bin", "go"),
		"run",
		".",
	)
	if targetOutput != goOutput {
		t.Fatalf(
			"cross-package local generic differs\nTypeScript: %q\nGo: %q",
			targetOutput,
			goOutput,
		)
	}
}

func TestDeferredGenericCallablesWithoutRecoverUseOrdinaryEntries(t *testing.T) {
	directory, err := filepath.Abs(filepath.Join(
		repositoryRoot(),
		"testdata",
		"constructs",
		"generic",
		"deferred-free",
	))
	if err != nil {
		t.Fatal(err)
	}
	program, err := load.Load(context.Background(), load.Request{
		Directory: directory,
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	root, err := emit.NewRoot(
		program.Roots()[0].Types().Scope().Lookup("Audit"),
	)
	if err != nil {
		t.Fatal(err)
	}
	emission, err := emit.Compile(program, []emit.Root{root})
	if err != nil {
		t.Fatal(err)
	}
	workingDirectory := t.TempDir()
	artifacts := materializeArtifacts(t, emission, workingDirectory)
	for _, required := range []string{
		"export function store$kernel<T>",
		"export function store$concrete_",
		"$goDeferred_",
		".resolve(",
		"deferred_callable_registry",
	} {
		if !strings.Contains(artifacts.printed, required) {
			t.Fatalf("deferred generic artifact lacks %q:\n%s", required, artifacts.printed)
		}
	}
	for _, forbidden := range []string{
		"$kernel$deferred",
		"$deferred($go$recovery",
		".register(",
	} {
		if strings.Contains(artifacts.printed, forbidden) {
			t.Fatalf(
				"non-recovering generic defer emitted %q:\n%s",
				forbidden,
				artifacts.printed,
			)
		}
	}
	if resolutions := strings.Count(artifacts.printed, ".resolve("); resolutions < 5 {
		t.Fatalf(
			"deferred callable forms resolved %d values, want at least 5:\n%s",
			resolutions,
			artifacts.printed,
		)
	}
	audit := targetFunctionText(t, artifacts.printed, "Audit")
	selected := targetFunctionText(t, artifacts.printed, "selected")
	for name, source := range map[string]string{
		"Audit":    audit,
		"selected": selected,
	} {
		if strings.Contains(source, "$goCapability_") ||
			strings.Contains(source, "$go$copy_") ||
			strings.Contains(source, "$go$pointer_") {
			t.Fatalf("source function %s exposes private mechanics:\n%s", name, source)
		}
	}

	runner := filepath.Join(workingDirectory, "runner.ts")
	writeProgramFile(t, runner, `import "./program.js";
import { Audit } from "`+artifacts.sourceModule+`";

console.log(String(Audit()));
`)
	writeProgramFile(
		t,
		filepath.Join(workingDirectory, "package.json"),
		"{\"type\":\"module\"}\n",
	)
	waveThreeTypecheck(t, workingDirectory, append(artifacts.paths, runner))
	targetOutput := runProgram(
		t,
		workingDirectory,
		"node",
		filepath.Join(workingDirectory, "out", "runner.js"),
	)
	goRunner := filepath.Join(workingDirectory, "go-runner")
	writeProgramFile(t, filepath.Join(goRunner, "go.mod"), fmt.Sprintf(
		`module example.com/runner

go 1.26.4

require example.com/deferredfree v0.0.0

replace example.com/deferredfree => %s
`,
		filepath.ToSlash(directory),
	))
	writeProgramFile(t, filepath.Join(goRunner, "main.go"), `package main

import (
	"fmt"
	"example.com/deferredfree"
)

func main() {
	fmt.Println(deferredfree.Audit())
}
`)
	goOutput := runProgram(
		t,
		goRunner,
		filepath.Join(runtime.GOROOT(), "bin", "go"),
		"run",
		".",
	)
	if targetOutput != goOutput {
		t.Fatalf(
			"deferred generic differs\nTypeScript: %q\nGo: %q",
			targetOutput,
			goOutput,
		)
	}
}
