package arrayvalue_test

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
)

const packedLiteralEntryCount = 4097

func TestLargeConstantArrayUsesOneBoundedPackedPayload(t *testing.T) {
	emission := compilePackedArrayFixture(t)
	directory := t.TempDir()
	target := materializeArrayProgram(t, directory, emission)
	var source strings.Builder
	for _, printed := range target.printed {
		source.WriteString(printed)
	}
	printed := source.String()
	if strings.Count(printed, "goArrayPacked<int32, 8193>") != 1 ||
		strings.Count(printed, "goArrayPacked<uint32, 4097>") != 1 {
		t.Fatalf(
			"large sparse constant array is not one packed call (packed=%d, literal=%d)",
			strings.Count(printed, "goArrayPacked<"),
			strings.Count(printed, ".literal<"),
		)
	}
	if strings.Count(printed, "function goArrayPacked<T") != 1 {
		t.Fatal("packed-array runtime operation was not demanded exactly once")
	}
	if strings.Count(printed, "GoArray.literal<int32, 3>") != 1 {
		t.Fatalf("small readable array did not retain the literal path")
	}
	if strings.Count(printed, "GoArray.literal<int32, 4097>") != 1 {
		t.Fatalf("nonconstant large array did not retain the literal path")
	}
	if strings.Contains(printed, "[0, 2, 4, 6, 8, 10") {
		t.Fatal("packed array retained its expanded index AST")
	}
}

func TestLargeConstantArrayExecutesDifferentiallyAndRejectsBadPayloads(
	t *testing.T,
) {
	emission := compilePackedArrayFixture(t)
	directory := t.TempDir()
	target := materializeArrayProgram(t, directory, emission)
	runner := filepath.Join(directory, "runner.ts")
	writeFile(t, runner, `import "`+target.programInit+`";
import { goArrayPacked } from "./runtime/array.js";
import { GoPanic } from "./runtime/panic.js";
import { DynamicAt, PackedAt, SmallAt, UnsignedAt } from "`+target.sourceModule+`";

console.log(PackedAt(0), PackedAt(1), PackedAt(4096), PackedAt(8192));
console.log(UnsignedAt(0), UnsignedAt(4096));
console.log(DynamicAt(0), DynamicAt(4096), SmallAt(2));
for (const action of [
    () => goArrayPacked<number, 4>(4, 0, 2, "0,1"),
    () => goArrayPacked<number, 4>(4, 0, 1, "0,1!"),
    () => goArrayPacked<number, 4>(4, 0, 1, "4,1"),
]) {
    try {
        action();
        console.log("malformed-accepted");
    } catch (error) {
        console.log(error instanceof GoPanic ? "malformed" : "wrong-error");
    }
}
`)
	writeFile(t, filepath.Join(directory, "package.json"), "{\"type\":\"module\"}\n")
	target.paths = append(target.paths, runner)
	if err := compileTypeScript(t, directory, target.paths); err != nil {
		t.Fatal(err)
	}
	typeScriptOutput := run(
		t,
		directory,
		"node",
		filepath.Join(directory, "out", "runner.js"),
	)
	goOutput := executePackedArrayGo(t, directory) + strings.Repeat("malformed\n", 3)
	if typeScriptOutput != goOutput {
		t.Fatalf(
			"packed array output differs\nTypeScript:\n%s\nGo:\n%s",
			typeScriptOutput,
			goOutput,
		)
	}
}

func TestLargeConstantArrayPackedPayloadStrictTypechecksWithBigIntProfile(
	t *testing.T,
) {
	options := emit.DefaultOptions()
	options.IntegerRepresentation = emit.IntegerRepresentationBigInt
	emission := compilePackedArrayFixtureWithOptions(t, options)
	directory := t.TempDir()
	target := materializeArrayProgram(t, directory, emission)
	writeFile(t, filepath.Join(directory, "package.json"), "{\"type\":\"module\"}\n")
	if err := compileTypeScript(t, directory, target.paths); err != nil {
		t.Fatal(err)
	}
}

func compilePackedArrayFixture(t *testing.T) emit.ProgramEmission {
	t.Helper()
	return compilePackedArrayFixtureWithOptions(t, arrayNumberOptions())
}

func compilePackedArrayFixtureWithOptions(
	t *testing.T,
	options emit.Options,
) emit.ProgramEmission {
	t.Helper()
	directory := t.TempDir()
	writeFile(
		t,
		filepath.Join(directory, "go.mod"),
		"module example.com/packedarray\n\ngo 1.26.4\n",
	)
	writeFile(t, filepath.Join(directory, "values.go"), packedArraySource())
	program, err := load.Load(context.Background(), load.Request{
		Directory: directory,
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	roots, err := emit.ExportedAPIRoots(program.Roots()[0])
	if err != nil {
		t.Fatal(err)
	}
	emission, err := emit.CompileWithOptions(program, roots, options)
	if err != nil {
		t.Fatal(err)
	}
	return emission
}

func packedArraySource() string {
	var source strings.Builder
	source.WriteString("package packedarray\n\n")
	source.WriteString("var Packed = [8193]int32{\n")
	for entry := range packedLiteralEntryCount {
		fmt.Fprintf(
			&source,
			"%d: %d,\n",
			entry*2,
			entry%257-128,
		)
	}
	source.WriteString("}\n\nvar Unsigned = [4097]uint32{\n")
	for entry := range packedLiteralEntryCount {
		fmt.Fprintf(&source, "%d,\n", uint64(1<<32-1)-uint64(entry))
	}
	source.WriteString("}\n\nvar Seed int32 = 7\n")
	source.WriteString("var Dynamic = [4097]int32{\n")
	for range packedLiteralEntryCount {
		source.WriteString("Seed,\n")
	}
	source.WriteString(`}

var Small = [3]int32{4, 5, 6}

func PackedAt(index int) int32 { return Packed[index] }
func UnsignedAt(index int) uint32 { return Unsigned[index] }
func DynamicAt(index int) int32 { return Dynamic[index] }
func SmallAt(index int) int32 { return Small[index] }
`)
	return source.String()
}

func executePackedArrayGo(t *testing.T, directory string) string {
	t.Helper()
	runnerDirectory := filepath.Join(directory, "go-runner-packed-array")
	fixtureDirectory := filepath.Join(directory, "go-fixture-packed-array")
	writeFile(
		t,
		filepath.Join(fixtureDirectory, "go.mod"),
		"module example.com/packedarray\n\ngo 1.26.4\n",
	)
	writeFile(
		t,
		filepath.Join(fixtureDirectory, "values.go"),
		packedArraySource(),
	)
	writeFile(t, filepath.Join(runnerDirectory, "go.mod"), `module example.com/runner

go 1.26.4

require example.com/packedarray v0.0.0

replace example.com/packedarray => `+filepath.ToSlash(fixtureDirectory)+"\n")
	writeFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"
	values "example.com/packedarray"
)

func main() {
	fmt.Println(values.PackedAt(0), values.PackedAt(1), values.PackedAt(4096), values.PackedAt(8192))
	fmt.Println(values.UnsignedAt(0), values.UnsignedAt(4096))
	fmt.Println(values.DynamicAt(0), values.DynamicAt(4096), values.SmallAt(2))
}
`)
	return run(t, runnerDirectory, "go", "run", ".")
}
