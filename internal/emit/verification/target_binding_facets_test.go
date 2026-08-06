package emit_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestGenericContainerStorageBindsExactTargetFacets(t *testing.T) {
	directory := filepath.Join(
		repositoryRoot(),
		"testdata",
		"constructs",
		"generic",
		"container-storage",
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
	for index, size := range artifacts.sizes {
		if index == 20 {
			break
		}
		t.Logf("artifact %s bytes=%d nodes=%d", size.path, size.bytes, size.nodes)
	}
	ordinaryBytes := artifacts.bytes
	concretizationBytes := 0
	concretizations := 0
	capabilityBytes := 0
	capabilities := 0
	for _, size := range artifacts.sizes {
		switch {
		case strings.HasPrefix(
			size.path,
			"support/generics/concretizations/",
		):
			ordinaryBytes -= size.bytes
			concretizationBytes += size.bytes
			concretizations++
		case strings.HasPrefix(
			size.path,
			"support/generics/capabilities/",
		):
			ordinaryBytes -= size.bytes
			capabilityBytes += size.bytes
			capabilities++
		}
	}
	for _, required := range []string{
		"class Bag<T>",
		"class Arena<T>",
		"RuntimeSlice<T>",
		"RuntimeSlice<GoContainerStorage<T>>",
		"GoPointerType<T>",
		"Bag.$zero<PlainItem>()",
		"Arena.$zero<Item>()",
		"class Item implements GoContainerStoredValue<Item$Storage>, GoPointerRepresentedValue<GoPointer<Item, Item$Storage>>",
		"GoPointer<Item, Item$Storage>",
		"GoPointer<int32, int32>",
		"function ArrayAddress$kernel<T>",
		"function ArrayAddress$concrete_",
		"RuntimeSlice.literal<Item$Storage>([Item.$storageOf(",
	} {
		if !strings.Contains(artifacts.printed, required) {
			t.Fatalf(
				"generic container-storage artifact lacks %q",
				required,
			)
		}
	}
	for _, forbidden := range []string{
		"T$Storage",
		"T$ContainerStorage",
		"T$Pointer",
		"RuntimeSlice<PlainItem$Storage>",
		"class PlainItem implements Go",
	} {
		if strings.Contains(artifacts.printed, forbidden) {
			t.Fatalf("generic container-storage artifact exposes %q", forbidden)
		}
	}
	runner := filepath.Join(workingDirectory, "runner.ts")
	writeProgramFile(t, runner, `import "./program.js";
import { Audit } from "`+artifacts.sourceModule+`";

const values = Audit();
console.log(Array.from({ length: values.length }, (_, index) =>
    String(values.get(index))).join(" "));
`)
	writeProgramFile(
		t,
		filepath.Join(workingDirectory, "package.json"),
		"{\"type\":\"module\"}\n",
	)
	waveThreeTypecheck(
		t,
		workingDirectory,
		append(artifacts.paths, runner),
	)
	assertPointerRepresentationMarkerIsRequired(
		t,
		workingDirectory,
		artifacts.paths,
	)
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

require example.com/genericcontainerstorage v0.0.0

replace example.com/genericcontainerstorage => %s
`,
		filepath.ToSlash(directory),
	))
	writeProgramFile(t, filepath.Join(goRunner, "main.go"), `package main

import (
	"fmt"

	values "example.com/genericcontainerstorage"
)

func main() {
	result := values.Audit()
	for index, value := range result {
		if index != 0 {
			fmt.Print(" ")
		}
		fmt.Print(value)
	}
	fmt.Println()
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
			"generic container-storage output differs\nTypeScript: %q\nGo: %q",
			targetOutput,
			goOutput,
		)
	}
	if ordinaryBytes > 34_000 ||
		concretizations != 11 || concretizationBytes > 15_000 ||
		capabilities != 22 || capabilityBytes > 10_000 ||
		artifacts.bytes > 58_000 ||
		artifacts.nodes > 9_000 ||
		artifacts.largest > 22_000 {
		t.Fatalf(
			"generic container-storage artifact bounds exceeded: ordinary=%d concretizations=%d/%d capabilities=%d/%d total=%d nodes=%d largest=%d",
			ordinaryBytes,
			concretizations,
			concretizationBytes,
			capabilities,
			capabilityBytes,
			artifacts.bytes,
			artifacts.nodes,
			artifacts.largest,
		)
	}
	t.Logf(
		"generic container-storage artifacts files=%d bytes=%d nodes=%d largest=%d",
		len(artifacts.paths),
		artifacts.bytes,
		artifacts.nodes,
		artifacts.largest,
	)
}

func assertPointerRepresentationMarkerIsRequired(
	t *testing.T,
	workingDirectory string,
	paths []string,
) {
	t.Helper()
	mutationDirectory := t.TempDir()
	mutated := false
	mutatedPaths := make([]string, 0, len(paths))
	for _, sourcePath := range paths {
		relative, err := filepath.Rel(workingDirectory, sourcePath)
		if err != nil || relative == "." || strings.HasPrefix(relative, "..") {
			t.Fatalf("generated artifact path %q is outside its root", sourcePath)
		}
		content, err := os.ReadFile(sourcePath)
		if err != nil {
			t.Fatal(err)
		}
		const pointerContract = ", GoPointerRepresentedValue<GoPointer<Item, Item$Storage>>"
		const pointerMember = "    declare readonly [$goPointerType]: GoPointer<Item, Item$Storage>;\n"
		if strings.Contains(string(content), pointerContract) {
			if mutated {
				t.Fatal("pointer-representation marker has multiple definition owners")
			}
			if !strings.Contains(string(content), pointerMember) {
				t.Fatal("pointer-representation contract has no marker member")
			}
			content = []byte(strings.Replace(string(content), pointerContract, "", 1))
			content = []byte(strings.Replace(string(content), pointerMember, "", 1))
			mutated = true
		}
		targetPath := filepath.Join(mutationDirectory, relative)
		writeProgramFile(t, targetPath, string(content))
		mutatedPaths = append(mutatedPaths, targetPath)
	}
	if !mutated {
		t.Fatal("pointer-representation mutation found no canonical marker")
	}
	writeProgramFile(
		t,
		filepath.Join(mutationDirectory, "package.json"),
		"{\"type\":\"module\"}\n",
	)
	arguments := []string{
		"--target", "es2022",
		"--module", "nodenext",
		"--moduleResolution", "nodenext",
		"--strict",
		"--noUncheckedIndexedAccess",
		"--outDir", filepath.Join(mutationDirectory, "out"),
	}
	arguments = append(arguments, mutatedPaths...)
	compileContext, cancel := context.WithTimeout(
		context.Background(),
		2*time.Minute,
	)
	defer cancel()
	err := tsgo.Compile(
		compileContext,
		repositoryRoot(),
		mutationDirectory,
		arguments,
	)
	if err == nil {
		t.Fatal("removing the concrete pointer marker preserved strict typing")
	}
	diagnostic := err.Error()
	if !strings.Contains(diagnostic, "Item$Storage") ||
		(!strings.Contains(diagnostic, "TS2345") &&
			!strings.Contains(diagnostic, "TS2322")) {
		t.Fatalf("pointer marker mutation failed at an unexpected boundary:\n%s", diagnostic)
	}
}
