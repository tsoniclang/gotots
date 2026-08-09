package slice_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
	runtimefixture "github.com/tsoniclang/gotots/internal/testfixture/gototsruntime"
	corefixture "github.com/tsoniclang/gotots/internal/testfixture/tsoniccore"
)

func TestScalarSlicesPrintTypecheckAndExecuteDifferentially(t *testing.T) {
	emission := compileFixture(t)
	workingDirectory := t.TempDir()
	targetPaths, module, printed := materialize(t, workingDirectory, emission)

	if strings.Contains(printed.source, ": int32[]") {
		t.Fatal("source module exposed a bare TypeScript array as a Go slice")
	}
	for _, fragment := range []string{
		"RuntimeSlice<int32>",
		".get(",
		".set(",
		".slice(",
		".append(",
		"RuntimeSlice.copy<int32>(",
		".length",
		".capacity",
		".isNil()",
	} {
		if !strings.Contains(printed.source, fragment) {
			t.Fatalf("source module lacks %q:\n%s", fragment, printed.source)
		}
	}
	if strings.Contains(printed.source, ".clone()") {
		t.Fatal("immutable slice descriptors emitted redundant physical clones")
	}
	for _, fragment := range []string{
		"export class RuntimeSlice<T>",
		"GoPanic.raise",
		"copyWithin",
	} {
		if !strings.Contains(printed.runtime, fragment) {
			t.Fatalf("runtime module lacks %q:\n%s", fragment, printed.runtime)
		}
	}

	runnerPath := filepath.Join(workingDirectory, "runner.ts")
	writeFile(t, runnerPath, `import "./program.js";
import {
    AppendDistinctNamedSlices,
    AppendDefinedStringBytes,
    AppendReallocates,
    AppendReusesBacking,
    AppendLargeSpread,
    AppendNoValues,
    AppendGrowthCapacity,
    AppendReallocationZeroTail,
	    AppendSpread,
	    AppendSpreadOverlap,
	    AppendStringBytes,
	    AppendUntypedStringConstantBytes,
	    BoolElements,
	    CopyCount,
	    CopyDefinedStringBytes,
	    CopyDistinct,
	    CopyOverlapping,
	    CopyStringBytes,
	    CopyUntypedStringConstantBytes,
    DescriptorAliasesBacking,
    EmptyIsNil,
    IndexBoundsPanic,
    IndexUpdates,
    KeyedLiteral,
    LiteralIndex,
    MakeShape,
    NamedSliceZeroIsNil,
    NegativeHighBoundsPanic,
    NilIsNil,
    NilSliceStaysNil,
    PackageSliceAliasesBacking,
    ParameterDescriptorIsIndependent,
    ParallelStoreOrder,
    ShadowedLenIsOrdinaryCall,
    SliceBoundsPanic,
    StoreBoundsPanic,
    StringIndexCompound,
    ThreeIndexSlice,
    TwoIndexSlice,
} from "`+module+`";

console.log(NilIsNil());
console.log(NamedSliceZeroIsNil());
console.log(PackageSliceAliasesBacking());
console.log(ParameterDescriptorIsIndependent());
console.log(EmptyIsNil());
console.log(MakeShape());
console.log(LiteralIndex());
console.log(KeyedLiteral());
console.log(DescriptorAliasesBacking());
console.log(TwoIndexSlice());
console.log(ThreeIndexSlice());
console.log(AppendReusesBacking());
console.log(AppendReallocates());
console.log(AppendNoValues());
console.log(AppendGrowthCapacity());
console.log(AppendReallocationZeroTail());
console.log(AppendSpread());
console.log(AppendSpreadOverlap());
console.log(AppendDistinctNamedSlices());
	console.log(AppendStringBytes());
	console.log(AppendDefinedStringBytes());
	console.log(AppendUntypedStringConstantBytes());
console.log(AppendLargeSpread());
console.log(IndexUpdates());
console.log(StringIndexCompound());
console.log(ParallelStoreOrder());
console.log(CopyOverlapping());
console.log(CopyDistinct());
console.log(CopyCount());
	console.log(CopyStringBytes());
	console.log(CopyDefinedStringBytes());
	console.log(CopyUntypedStringConstantBytes());
console.log(NilSliceStaysNil());
console.log(BoolElements());
console.log(ShadowedLenIsOrdinaryCall());
for (const operation of [
    IndexBoundsPanic,
    StoreBoundsPanic,
    SliceBoundsPanic,
    NegativeHighBoundsPanic,
]) {
    try {
        operation();
        console.log("missing-panic");
    } catch {
        console.log("panic");
    }
}
`)
	targetPaths = append(targetPaths, runnerPath)
	writeFile(t, filepath.Join(workingDirectory, "package.json"), "{\"type\":\"module\"}\n")
	typecheck(t, workingDirectory, targetPaths)
	targetOutput := run(
		t,
		workingDirectory,
		"node",
		filepath.Join(workingDirectory, "out", "runner.js"),
	)
	goOutput := executeGo(t, workingDirectory)
	const expected = "true\ntrue\n7\ntrue\nfalse\n25\n5\n57\n9\n15\n13\n934\n1\n24\n4\n0\n4\n123\n123\n29669\n29669\n29669\n7\n19\nab\n12345678\n1123\n78\n2\n2109669\n39669\n39669\ntrue\ntrue\n5\npanic\npanic\npanic\npanic\n"
	if goOutput != expected {
		t.Fatalf("Go output = %q, want exact slice mutation sentinel output", goOutput)
	}
	if targetOutput != goOutput {
		t.Fatalf("TypeScript output = %q, Go output = %q", targetOutput, goOutput)
	}
}

func TestScalarSlicesBigIntProfileRemainsStrictAndTyped(t *testing.T) {
	emission := compileFixtureWithOptions(t, emit.Options{
		IntegerRepresentation: emit.IntegerRepresentationBigInt,
		EvaluationOrder:       emit.EvaluationOrderDirect,
	})
	workingDirectory := t.TempDir()
	targetPaths, _, printed := materialize(t, workingDirectory, emission)
	writeFile(t, filepath.Join(workingDirectory, "package.json"), "{\"type\":\"module\"}\n")
	typecheck(t, workingDirectory, targetPaths)
	for _, fragment := range []string{
		"RuntimeSlice.make<int32>(0n, null, 0)",
		".get(1n)",
		".set(1n, 9)",
		"BigInt(values.length)",
		"BigInt(values.capacity)",
		"BigInt(RuntimeSlice.copy<int32>(",
	} {
		if !strings.Contains(printed.source, fragment) {
			t.Fatalf("bigint source module lacks %q:\n%s", fragment, printed.source)
		}
	}
}

func TestScalarSlicesPreserveGoOrderProfileRemainsStrict(t *testing.T) {
	emission := compileFixtureWithOptions(t, emit.Options{
		IntegerRepresentation: emit.IntegerRepresentationNumber,
		EvaluationOrder:       emit.EvaluationOrderPreserveGo,
	})
	workingDirectory := t.TempDir()
	targetPaths, _, printed := materialize(t, workingDirectory, emission)
	writeFile(t, filepath.Join(workingDirectory, "package.json"), "{\"type\":\"module\"}\n")
	typecheck(t, workingDirectory, targetPaths)
	if strings.Contains(printed.source, ".clone()") {
		t.Fatal("preserve-go profile emitted redundant immutable descriptor clones")
	}
}

func compileFixture(t *testing.T) emit.ProgramEmission {
	t.Helper()
	return compileFixtureWithOptions(t, emit.Options{
		IntegerRepresentation: emit.IntegerRepresentationNumber,
		EvaluationOrder:       emit.EvaluationOrderDirect,
	})
}

func compileFixtureWithOptions(
	t *testing.T,
	options emit.Options,
) emit.ProgramEmission {
	t.Helper()
	program, err := load.Load(context.Background(), load.Request{
		Directory: fixtureDirectory(),
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

func compileSliceSource(
	t *testing.T,
	source string,
) emit.ProgramEmission {
	t.Helper()
	directory := t.TempDir()
	writeFile(
		t,
		filepath.Join(directory, "go.mod"),
		"module example.com/demand\n\ngo 1.26.4\n",
	)
	writeFile(t, filepath.Join(directory, "source.go"), source)
	loaded, err := load.One(context.Background(), load.Request{
		Directory: directory,
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
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

type printedFiles struct {
	source  string
	runtime string
}

func materialize(
	t *testing.T,
	workingDirectory string,
	emission emit.ProgramEmission,
) ([]string, string, printedFiles) {
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
	var targetPaths []string
	var module string
	var printed printedFiles
	for _, file := range emission.Files() {
		source, err := client.PrintNode(file.SourceFile(), tsgo.PrintOptions{})
		if err != nil {
			t.Fatal(err)
		}
		targetPath := filepath.Join(
			workingDirectory,
			filepath.FromSlash(file.OutputPath()),
		)
		writeFile(t, targetPath, source)
		targetPaths = append(targetPaths, targetPath)
		switch {
		case file.Kind() == emit.TargetFileSource:
			printed.source += source
		case file.Kind() == emit.TargetFilePackageAssembly:
			module = "./" + strings.TrimSuffix(file.OutputPath(), ".ts") + ".js"
		case file.OutputPath() == "runtime/slice.ts":
			printed.runtime = source
		}
	}
	if module == "" || printed.source == "" || printed.runtime == "" {
		t.Fatalf(
			"materialized module=%q source=%d runtime=%d",
			module,
			len(printed.source),
			len(printed.runtime),
		)
	}
	return targetPaths, module, printed
}

func typecheck(t *testing.T, workingDirectory string, paths []string) {
	t.Helper()
	if err := corefixture.InstallResolutionOnly(workingDirectory); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	arguments := []string{
		"--target", "es2022",
		"--module", "nodenext",
		"--moduleResolution", "nodenext",
		"--strict",
		"--outDir", filepath.Join(workingDirectory, "out"),
	}
	arguments = append(arguments, paths...)
	if err := runtimefixture.InstallResolution(workingDirectory, filepath.Join(workingDirectory, "out")); err != nil {
		t.Fatal(err)
	}
	if err := tsgo.Compile(
		ctx,
		repositoryRoot(),
		workingDirectory,
		arguments,
	); err != nil {
		t.Fatal(err)
	}
}

func executeGo(t *testing.T, workingDirectory string) string {
	t.Helper()
	modulePath, err := filepath.Abs(fixtureDirectory())
	if err != nil {
		t.Fatal(err)
	}
	runnerDirectory := filepath.Join(workingDirectory, "go-runner")
	writeFile(t, filepath.Join(runnerDirectory, "go.mod"), fmt.Sprintf(`module example.com/runner

go 1.26.4

require example.com/slicevalues v0.0.0

replace example.com/slicevalues => %s
`, filepath.ToSlash(modulePath)))
	writeFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"

	values "example.com/slicevalues"
)

func didPanic(operation func()) (result string) {
	result = "missing-panic"
	defer func() {
		if recover() != nil {
			result = "panic"
		}
	}()
	operation()
	return result
}

func main() {
	fmt.Println(values.NilIsNil())
	fmt.Println(values.NamedSliceZeroIsNil())
	fmt.Println(values.PackageSliceAliasesBacking())
	fmt.Println(values.ParameterDescriptorIsIndependent())
	fmt.Println(values.EmptyIsNil())
	fmt.Println(values.MakeShape())
	fmt.Println(values.LiteralIndex())
	fmt.Println(values.KeyedLiteral())
	fmt.Println(values.DescriptorAliasesBacking())
	fmt.Println(values.TwoIndexSlice())
	fmt.Println(values.ThreeIndexSlice())
	fmt.Println(values.AppendReusesBacking())
	fmt.Println(values.AppendReallocates())
	fmt.Println(values.AppendNoValues())
	fmt.Println(values.AppendGrowthCapacity())
	fmt.Println(values.AppendReallocationZeroTail())
	fmt.Println(values.AppendSpread())
	fmt.Println(values.AppendSpreadOverlap())
	fmt.Println(values.AppendDistinctNamedSlices())
	fmt.Println(values.AppendStringBytes())
	fmt.Println(values.AppendDefinedStringBytes())
	fmt.Println(values.AppendUntypedStringConstantBytes())
	fmt.Println(values.AppendLargeSpread())
	fmt.Println(values.IndexUpdates())
	fmt.Println(values.StringIndexCompound())
	fmt.Println(values.ParallelStoreOrder())
	fmt.Println(values.CopyOverlapping())
	fmt.Println(values.CopyDistinct())
	fmt.Println(values.CopyCount())
	fmt.Println(values.CopyStringBytes())
	fmt.Println(values.CopyDefinedStringBytes())
	fmt.Println(values.CopyUntypedStringConstantBytes())
	fmt.Println(values.NilSliceStaysNil())
	fmt.Println(values.BoolElements())
	fmt.Println(values.ShadowedLenIsOrdinaryCall())
	fmt.Println(didPanic(func() { values.IndexBoundsPanic() }))
	fmt.Println(didPanic(func() { values.StoreBoundsPanic() }))
	fmt.Println(didPanic(func() { values.SliceBoundsPanic() }))
	fmt.Println(didPanic(func() { values.NegativeHighBoundsPanic() }))
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

func run(
	t *testing.T,
	directory string,
	name string,
	arguments ...string,
) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	command := exec.CommandContext(ctx, name, arguments...)
	command.Dir = directory
	command.Env = append(os.Environ(), "GOMEMLIMIT=1GiB")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %s: %v\n%s", name, strings.Join(arguments, " "), err, output)
	}
	return string(output)
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func fixtureDirectory() string {
	return filepath.Join(
		repositoryRoot(),
		"testdata",
		"constructs",
		"value",
		"slice",
		"scalars",
	)
}

func repositoryRoot() string {
	return filepath.Join("..", "..", "..", "..")
}
