package mixed_test

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
)

func TestMixedValueFamiliesDefaultNumberProfileTypechecksAndExecutes(
	t *testing.T,
) {
	program, err := load.Load(context.Background(), load.Request{
		Directory: fixtureDirectory(),
		Pattern:   "./api",
	})
	if err != nil {
		t.Fatal(err)
	}
	roots, err := emit.ExportedAPIRoots(program.Roots()[0])
	if err != nil {
		t.Fatal(err)
	}
	selected := roots[:0]
	for _, root := range roots {
		switch root.Object().Name() {
		case "NumberValue", "DividePanic":
			continue
		default:
			selected = append(selected, root)
		}
	}
	emission, err := emit.Compile(program, selected)
	if err != nil {
		t.Fatal(err)
	}
	workingDirectory := t.TempDir()
	artifacts := materialize(t, emission, workingDirectory)
	assertRuntimeGraph(
		t,
		artifacts,
		"runtime/string.ts",
		"runtime/array.ts",
		"runtime/slice.ts",
		"runtime/map.ts",
	)
	if _, ok := artifacts.printed["runtime/integer.ts"]; ok {
		t.Fatal("default number profile emitted unrequested BigInt integer runtime")
	}
	runnerPath := filepath.Join(workingDirectory, "runner.ts")
	writeFile(t, runnerPath, `import {
    ArrayPanic,
    ArrayValue,
    MapPanic,
    MapStoreOrder,
    MapValue,
    SlicePanic,
    SliceStoreOrder,
    SliceValue,
    StringByte,
    StringPanic,
    StringWindow,
} from "`+artifacts.apiModule+`";
import { GoPanic } from "./runtime/panic.js";

const panics = (operation: () => void): boolean => {
    try {
        operation();
        return false;
    } catch (error) {
        return error instanceof GoPanic;
    }
};

console.log(StringByte("abc"));
console.log(StringWindow("abcd"));
console.log(ArrayValue(3));
console.log(SliceValue(4));
console.log(MapValue(5));
console.log(SliceStoreOrder());
console.log(MapStoreOrder());
console.log(panics(() => { ArrayPanic(1); }));
console.log(panics(() => { SlicePanic(1); }));
console.log(panics(() => { StringPanic("a", 1); }));
console.log(panics(() => { MapPanic(); }));
`)
	writeFile(
		t,
		filepath.Join(workingDirectory, "package.json"),
		"{\"type\":\"module\"}\n",
	)
	typecheck(t, workingDirectory, artifacts.paths, runnerPath)
	targetOutput := run(
		t,
		workingDirectory,
		"node",
		filepath.Join(workingDirectory, "out", "runner.js"),
	)
	goOutput := executeDefaultGo(t, workingDirectory)
	if targetOutput != goOutput {
		t.Fatalf(
			"TypeScript output = %q, Go output = %q",
			targetOutput,
			goOutput,
		)
	}
}

func executeDefaultGo(t *testing.T, workingDirectory string) string {
	t.Helper()
	runnerDirectory := filepath.Join(workingDirectory, "go-default-runner")
	writeFile(t, filepath.Join(runnerDirectory, "go.mod"), fmt.Sprintf(`module example.com/runner

go 1.26.4

require example.com/mixedfamily v0.0.0

replace example.com/mixedfamily => %s
`, filepath.ToSlash(fixtureDirectory())))
	writeFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"
	values "example.com/mixedfamily/api"
)

func panics(operation func()) (result bool) {
	defer func() {
		result = recover() != nil
	}()
	operation()
	return false
}

func main() {
	fmt.Println(values.StringByte("abc"))
	fmt.Println(values.StringWindow("abcd"))
	fmt.Println(values.ArrayValue(3))
	fmt.Println(values.SliceValue(4))
	fmt.Println(values.MapValue(5))
	fmt.Println(values.SliceStoreOrder())
	fmt.Println(values.MapStoreOrder())
	fmt.Println(panics(func() { values.ArrayPanic(1) }))
	fmt.Println(panics(func() { values.SlicePanic(1) }))
	fmt.Println(panics(func() { values.StringPanic("a", 1) }))
	fmt.Println(panics(values.MapPanic))
}
`)
	return run(
		t,
		runnerDirectory,
		filepath.Join(runtime.GOROOT(), "bin", "go"),
		"run",
		".",
	)
}
