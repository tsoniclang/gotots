package pointer_test

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

func TestAddressablePointersPrintTypecheckAndExecuteDifferentially(t *testing.T) {
	loaded := loadAddressablePointerProject(t)
	workingDirectory := t.TempDir()
	artifacts := materializeExportedProgram(t, loaded, workingDirectory)
	sourcePath := filepath.Join(
		workingDirectory,
		filepath.FromSlash(strings.TrimPrefix(artifacts.module(t, "source.ts"), "./")),
	)
	sourcePath = strings.TrimSuffix(sourcePath, ".js") + ".ts"
	printed, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	target := string(printed)
	for _, required := range []string{
		"GoPointer.cell",
		"GoPointer.field",
		"GoPointer.index",
		"goSliceAddress",
		"value$storage",
		"export class Box",
		"static Add(box:",
		"static Nil(box:",
		"selected$storage",
	} {
		if !strings.Contains(target, required) {
			t.Fatalf("addressable pointer artifact lacks %q:\n%s", required, target)
		}
	}
	for _, forbidden := range []string{
		"any",
		"unknown",
		".call(",
		".apply(",
		".bind(",
		"GoPointer.indexView",
		"goSliceAddressView",
		"export function Box_Add",
		"export function Box_Nil",
	} {
		if strings.Contains(target, forbidden) {
			t.Fatalf("addressable pointer artifact contains %q:\n%s", forbidden, target)
		}
	}

	runnerPath := filepath.Join(workingDirectory, "runner.ts")
	writeFile(t, runnerPath, `import "`+artifacts.programInitialization+`";
import {
    AddressOrder,
    AddressReceiverOrder,
    Array,
    ArrayAddress,
    ArrayThroughPointer,
    Box,
    Cancel,
    CancelIdentity,
    Closure,
    Composite,
    DefinedArrayAddress,
    DefinedSliceAddress,
    ElidedPointerArray,
    ElidedPointerCompositeArray,
    ElidedPointerMap,
    ElidedPointerSlice,
    EscapedValue,
    Field,
    FunctionVariable,
    Initialized,
    Local,
    MapVariable,
    NamedAggregate,
    NamedResult,
    NewArray,
    NewMap,
    NewPointer,
    NewSlice,
    NewStruct,
    NestedField,
    NilPointerReceiver,
    MultipleResult,
    Package,
    PackageValueAddress,
    Parallel,
    Parameter,
    PointerReceiverOnPointer,
    PointerReceiverOnValue,
	TypeSwitchPointerReceiverAudit,
    PointerField,
    PointerToPointer,
    ProjectionDoesNotRetarget,
    ReceiverStorage,
    ReceiverOrder,
    Shadowed,
    Slice,
    SliceAddress,
    SliceVariable,
    SliceReallocation,
    StructArrayAddress,
    StructSliceAddress,
    ValueReceiverThroughPointer,
} from "`+artifacts.module(t, "source.ts")+`";

console.log(...Local(10));
console.log(EscapedValue(11));
console.log(Parameter(10));
console.log(NamedResult(13));
console.log(...NamedAggregate(14));
console.log(...Parallel(15));
console.log(...MultipleResult(16));
const closure = Closure(20);
if (closure === undefined) {
    throw new Error("Closure returned nil");
}
console.log(closure(), closure());
console.log(...Field(30));
console.log(...NestedField(35));
console.log(...Array(40));
console.log(...ArrayThroughPointer(45));
try {
    ArrayAddress(1);
    console.log(false);
} catch {
    console.log(true);
}
console.log(...Slice(50));
try {
    SliceAddress(1);
    console.log(false);
} catch {
    console.log(true);
}
console.log(...DefinedArrayAddress(51));
console.log(...DefinedSliceAddress(52));
console.log(...StructArrayAddress(53));
console.log(...StructSliceAddress(54));
console.log(...Package(60));
console.log(...PackageValueAddress(61));
console.log(Composite(70));
console.log(...ElidedPointerSlice(71));
console.log(ElidedPointerArray(72));
console.log(ElidedPointerMap(73));
console.log(ElidedPointerCompositeArray(74));
console.log(PointerField(75));
console.log(PointerToPointer(76));
console.log(...SliceVariable(76));
console.log(...MapVariable(76));
console.log(...FunctionVariable(76));
console.log(NewMap(77));
console.log(NewSlice(78));
console.log(NewArray());
console.log(NewStruct());
console.log(NewPointer());
console.log(...ProjectionDoesNotRetarget(77));
console.log(...AddressOrder(78));
console.log(...AddressReceiverOrder(79));
console.log(...Shadowed(80));
console.log(...SliceReallocation(81));
console.log(CancelIdentity(82));
try {
    Cancel(undefined);
    console.log(false);
} catch {
    console.log(true);
}
console.log(PointerReceiverOnValue(80));
console.log(PointerReceiverOnPointer(90));
console.log(TypeSwitchPointerReceiverAudit(95));
try {
    Box.Add(undefined, 1);
    console.log(false);
} catch {
    console.log(true);
}
console.log(NilPointerReceiver());
console.log(...ValueReceiverThroughPointer(100));
console.log(ReceiverStorage(110));
console.log(...ReceiverOrder(120));
console.log(Initialized());
`)
	typeScriptOutput := executeMaterializedTypeScript(
		t,
		workingDirectory,
		artifacts,
		runnerPath,
	)
	goOutput := executeAddressablePointerGo(t, workingDirectory)
	if typeScriptOutput != goOutput {
		t.Fatalf("TypeScript output = %q, Go output = %q", typeScriptOutput, goOutput)
	}
}

func TestAddressabilityDoesNotWrapUnrelatedLocals(t *testing.T) {
	loaded := loadAddressablePointerProject(t)
	workingDirectory := t.TempDir()
	artifacts := materializeExportedProgram(t, loaded, workingDirectory)
	var source string
	for _, path := range artifacts.targetPaths {
		if strings.HasSuffix(path, "/source.ts") {
			printed, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			source = string(printed)
			break
		}
	}
	if source == "" {
		t.Fatal("source.ts was not materialized")
	}
	if strings.Contains(source, "delta$storage") {
		t.Fatal("unaddressed pointer-receiver argument became a storage cell")
	}
	if strings.Contains(source, "result$storage") &&
		!strings.Contains(source, "function NamedResult") {
		t.Fatal("storage evidence was not scoped to the selected declaration")
	}
}

func TestImportedPackageStoragePrintsTypechecksAndExecutesDifferentially(
	t *testing.T,
) {
	projectDirectory := filepath.Join(
		repositoryRoot(),
		"testdata",
		"projects",
		"pointer-addressable",
	)
	projectDirectory, err := filepath.Abs(projectDirectory)
	if err != nil {
		t.Fatal(err)
	}
	program, err := load.Load(context.Background(), load.Request{
		Directory: projectDirectory,
		Pattern:   "./app",
	})
	if err != nil {
		t.Fatal(err)
	}
	workingDirectory := t.TempDir()
	artifacts := materializeExportedProgram(
		t,
		program.Roots()[0],
		workingDirectory,
	)
	runnerPath := filepath.Join(workingDirectory, "runner.ts")
	writeFile(t, runnerPath, `import "`+artifacts.programInitialization+`";
import { Mutate } from "`+artifacts.module(t, "app.ts")+`";
console.log(...Mutate(20));
`)
	typeScriptOutput := executeMaterializedTypeScript(
		t,
		workingDirectory,
		artifacts,
		runnerPath,
	)
	goRunner := filepath.Join(workingDirectory, "go-runner")
	writeFile(t, filepath.Join(goRunner, "go.mod"), fmt.Sprintf(`module example.com/runner

go 1.26.4

require example.com/pointer-addressable v0.0.0

replace example.com/pointer-addressable => %s
`, filepath.ToSlash(projectDirectory)))
	writeFile(t, filepath.Join(goRunner, "main.go"), `package main

import (
    "fmt"
    "example.com/pointer-addressable/app"
)

func main() {
    fmt.Println(app.Mutate(20))
}
`)
	goOutput := run(
		t,
		goRunner,
		filepath.Join(runtime.GOROOT(), "bin", "go"),
		"run",
		".",
	)
	if typeScriptOutput != goOutput {
		t.Fatalf("TypeScript output = %q, Go output = %q", typeScriptOutput, goOutput)
	}
}

func TestPointerIndexIdentityNormalizesNumberAndBigInt(t *testing.T) {
	loaded := loadAddressablePointerProject(t)
	workingDirectory := t.TempDir()
	artifacts := materializeExportedProgram(t, loaded, workingDirectory)
	runnerPath := filepath.Join(workingDirectory, "runner.ts")
	writeFile(t, runnerPath, `import { GoPointer } from "./runtime/pointer.js";

class Indexed {
    readonly values = [10, 20];

    get(index: number | bigint): number {
        return this.values[Number(index)];
    }

    set(index: number | bigint, value: number): void {
        this.values[Number(index)] = value;
    }
}

const parent = GoPointer.cell<Indexed, Indexed>(new Indexed());
const numberIndex = GoPointer.index<number, number, Indexed, Indexed>(parent, 1);
const bigintIndex = GoPointer.index<number, number, Indexed, Indexed>(parent, 1n);
console.log(GoPointer.equal(numberIndex, bigintIndex));
`)
	if output := executeMaterializedTypeScript(
		t,
		workingDirectory,
		artifacts,
		runnerPath,
	); output != "true\n" {
		t.Fatalf("mixed-index pointer identity = %q, want true", output)
	}
}

func loadAddressablePointerProject(t *testing.T) *load.Package {
	t.Helper()
	loaded, err := load.One(context.Background(), load.Request{
		Directory: addressablePointerProjectDirectory(),
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	return loaded
}

func executeAddressablePointerGo(t *testing.T, workingDirectory string) string {
	t.Helper()
	modulePath, err := filepath.Abs(addressablePointerProjectDirectory())
	if err != nil {
		t.Fatal(err)
	}
	runnerDirectory := filepath.Join(workingDirectory, "go-runner")
	writeFile(t, filepath.Join(runnerDirectory, "go.mod"), fmt.Sprintf(`module example.com/runner

go 1.26.4

require example.com/addressablepointer v0.0.0

replace example.com/addressablepointer => %s
`, filepath.ToSlash(modulePath)))
	writeFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
    "fmt"

    pointer "example.com/addressablepointer"
)

func main() {
    fmt.Println(pointer.Local(10))
    fmt.Println(pointer.EscapedValue(11))
    fmt.Println(pointer.Parameter(10))
    fmt.Println(pointer.NamedResult(13))
    fmt.Println(pointer.NamedAggregate(14))
    fmt.Println(pointer.Parallel(15))
    fmt.Println(pointer.MultipleResult(16))
    closure := pointer.Closure(20)
    fmt.Println(closure(), closure())
    fmt.Println(pointer.Field(30))
    fmt.Println(pointer.NestedField(35))
    fmt.Println(pointer.Array(40))
    fmt.Println(pointer.ArrayThroughPointer(45))
    arrayAddressPanicked := false
    func() {
        defer func() {
            arrayAddressPanicked = recover() != nil
        }()
        pointer.ArrayAddress(1)
    }()
    fmt.Println(arrayAddressPanicked)
    fmt.Println(pointer.Slice(50))
    sliceAddressPanicked := false
    func() {
        defer func() {
            sliceAddressPanicked = recover() != nil
        }()
        pointer.SliceAddress(1)
    }()
    fmt.Println(sliceAddressPanicked)
    fmt.Println(pointer.DefinedArrayAddress(51))
    fmt.Println(pointer.DefinedSliceAddress(52))
    fmt.Println(pointer.StructArrayAddress(53))
    fmt.Println(pointer.StructSliceAddress(54))
    fmt.Println(pointer.Package(60))
    fmt.Println(pointer.PackageValueAddress(61))
    fmt.Println(pointer.Composite(70))
    fmt.Println(pointer.ElidedPointerSlice(71))
    fmt.Println(pointer.ElidedPointerArray(72))
    fmt.Println(pointer.ElidedPointerMap(73))
    fmt.Println(pointer.ElidedPointerCompositeArray(74))
    fmt.Println(pointer.PointerField(75))
    fmt.Println(pointer.PointerToPointer(76))
    fmt.Println(pointer.SliceVariable(76))
    fmt.Println(pointer.MapVariable(76))
    fmt.Println(pointer.FunctionVariable(76))
    fmt.Println(pointer.NewMap(77))
    fmt.Println(pointer.NewSlice(78))
    fmt.Println(pointer.NewArray())
    fmt.Println(pointer.NewStruct())
    fmt.Println(pointer.NewPointer())
    fmt.Println(pointer.ProjectionDoesNotRetarget(77))
    fmt.Println(pointer.AddressOrder(78))
    fmt.Println(pointer.AddressReceiverOrder(79))
    fmt.Println(pointer.Shadowed(80))
    fmt.Println(pointer.SliceReallocation(81))
    fmt.Println(pointer.CancelIdentity(82))
    panicked := false
    func() {
        defer func() {
            panicked = recover() != nil
        }()
        pointer.Cancel(nil)
    }()
    fmt.Println(panicked)
    fmt.Println(pointer.PointerReceiverOnValue(80))
    fmt.Println(pointer.PointerReceiverOnPointer(90))
	fmt.Println(pointer.TypeSwitchPointerReceiverAudit(95))
    pointerReceiverPanicked := false
    func() {
        defer func() {
            pointerReceiverPanicked = recover() != nil
        }()
        var box *pointer.Box
        box.Add(1)
    }()
    fmt.Println(pointerReceiverPanicked)
    fmt.Println(pointer.NilPointerReceiver())
    fmt.Println(pointer.ValueReceiverThroughPointer(100))
    fmt.Println(pointer.ReceiverStorage(110))
    fmt.Println(pointer.ReceiverOrder(120))
    fmt.Println(pointer.Initialized())
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

func addressablePointerProjectDirectory() string {
	return filepath.Join(
		repositoryRoot(),
		"testdata",
		"constructs",
		"value",
		"pointer",
		"addressable",
	)
}

func compileAddressablePointerProject(
	t *testing.T,
	loaded *load.Package,
) emit.ProgramEmission {
	t.Helper()
	roots, err := emit.ExportedAPIRoots(loaded)
	if err != nil {
		t.Fatal(err)
	}
	emission, err := emit.Compile(loaded.Program(), roots)
	if err != nil {
		t.Fatal(err)
	}
	return emission
}
