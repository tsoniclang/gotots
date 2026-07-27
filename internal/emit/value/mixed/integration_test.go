package mixed_test

import (
	"context"
	"encoding/binary"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestMixedValueFamiliesTypecheckAndExecuteThroughOneRuntimeGraph(
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
	options := emit.DefaultOptions()
	options.IntegerRepresentation = emit.IntegerRepresentationBigInt
	emission, err := emit.CompileWithOptions(program, roots, options)
	if err != nil {
		t.Fatal(err)
	}
	workingDirectory := t.TempDir()
	artifacts := materialize(t, emission, workingDirectory)
	assertRuntimeGraph(
		t,
		artifacts,
		"runtime/integer.ts",
		"runtime/string.ts",
		"runtime/pointer.ts",
		"runtime/array.ts",
		"runtime/slice.ts",
		"runtime/map.ts",
	)
	runnerPath := filepath.Join(workingDirectory, "runner.ts")
	writeFile(t, runnerPath, `import {
    ArrayPanic,
    ArrayValue,
    DividePanic,
    MapPanic,
    MapStoreOrder,
    MapValue,
    NumberValue,
    PointerPanic,
    PointerValue,
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

console.log(NumberValue(17n, 5n).toString());
console.log(StringByte("abc").toString());
console.log(StringWindow("abcd"));
console.log(PointerValue(7n).toString());
console.log(ArrayValue(3n).toString());
console.log(SliceValue(4n).toString());
console.log(MapValue(5n).toString());
console.log(SliceStoreOrder().toString());
console.log(MapStoreOrder().toString());
console.log(panics(() => { PointerPanic(); }));
console.log(panics(() => { ArrayPanic(1n); }));
console.log(panics(() => { SlicePanic(1n); }));
console.log(panics(() => { StringPanic("a", 1n); }));
console.log(panics(() => { MapPanic(); }));
console.log(panics(() => { DividePanic(0n); }));
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
	goOutput := executeGo(t, workingDirectory)
	if targetOutput != goOutput {
		t.Fatalf(
			"TypeScript output = %q, Go output = %q",
			targetOutput,
			goOutput,
		)
	}
	t.Logf(
		"mixed families: Go=%d source=%d runtime=%d scalar=%d assembly=%d total=%d nodes=%d files=%d",
		sourceBytes(t),
		artifacts.sourceBytes,
		artifacts.runtimeBytes,
		artifacts.scalarBytes,
		artifacts.assemblyBytes,
		artifacts.bytes,
		artifacts.nodes,
		len(artifacts.paths),
	)
	for index, form := range artifacts.largestForms(20) {
		t.Logf(
			"mixed artifact tail %02d: %s bytes=%d nodes=%d",
			index+1,
			form.identity,
			form.bytes,
			form.nodes,
		)
	}
}

type materialized struct {
	paths         []string
	printed       map[string]string
	forms         []artifactForm
	apiModule     string
	bytes         int
	nodes         int
	sourceBytes   int
	runtimeBytes  int
	scalarBytes   int
	assemblyBytes int
}

type artifactForm struct {
	identity string
	bytes    int
	nodes    int
}

func (m materialized) largestForms(limit int) []artifactForm {
	forms := append([]artifactForm(nil), m.forms...)
	sort.Slice(forms, func(left, right int) bool {
		if forms[left].bytes != forms[right].bytes {
			return forms[left].bytes > forms[right].bytes
		}
		return forms[left].identity < forms[right].identity
	})
	if len(forms) > limit {
		forms = forms[:limit]
	}
	return forms
}

func materialize(
	t *testing.T,
	emission emit.ProgramEmission,
	workingDirectory string,
) materialized {
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
	result := materialized{printed: make(map[string]string)}
	for _, file := range emission.Files() {
		printed, err := client.PrintNode(file.SourceFile(), tsgo.PrintOptions{})
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(
			workingDirectory,
			filepath.FromSlash(file.OutputPath()),
		)
		writeFile(t, path, printed)
		result.paths = append(result.paths, path)
		result.printed[file.OutputPath()] = printed
		result.bytes += len(printed)
		encoded, err := tsgo.EncodeSourceFile(file.SourceFile())
		if err != nil {
			t.Fatal(err)
		}
		result.nodes += encodedNodeCount(t, encoded)
		switch {
		case file.Kind() == emit.TargetFileSource:
			result.sourceBytes += len(printed)
		case strings.HasPrefix(file.OutputPath(), "runtime/"):
			result.runtimeBytes += len(printed)
		case file.OutputPath() == "support/scalars.ts":
			result.scalarBytes += len(printed)
		default:
			result.assemblyBytes += len(printed)
		}
		for index, statement := range file.SourceFile().Statements() {
			result.forms = append(
				result.forms,
				measureForm(
					t,
					client,
					fmt.Sprintf("%s#statement[%d]", file.OutputPath(), index),
					statement,
				),
			)
			class, ok := statement.(tsgo.ClassDeclaration)
			if !ok {
				continue
			}
			for memberIndex, member := range class.Members() {
				result.forms = append(
					result.forms,
					measureForm(
						t,
						client,
						fmt.Sprintf(
							"%s#statement[%d].member[%d]",
							file.OutputPath(),
							index,
							memberIndex,
						),
						member,
					),
				)
			}
		}
		if file.Kind() == emit.TargetFileSource &&
			file.PackageName() == "api" {
			result.apiModule = "./" +
				strings.TrimSuffix(file.OutputPath(), ".ts") + ".js"
		}
	}
	if result.apiModule == "" {
		t.Fatal("mixed fixture emitted no API source module")
	}
	return result
}

func measureForm(
	t *testing.T,
	client *tsgo.Client,
	identity string,
	node tsgo.Node,
) artifactForm {
	t.Helper()
	printed, err := client.PrintNode(node, tsgo.PrintOptions{})
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := tsgo.EncodeNode(node)
	if err != nil {
		t.Fatal(err)
	}
	return artifactForm{
		identity: identity,
		bytes:    len(printed),
		nodes:    encodedNodeCount(t, encoded),
	}
}

func encodedNodeCount(t *testing.T, encoded []byte) int {
	t.Helper()
	const (
		headerSize       = 44
		nodesOffsetField = 40
		nodeWidth        = 28
	)
	if len(encoded) < headerSize {
		t.Fatalf(
			"encoded TS-Go AST = %d bytes, shorter than protocol header",
			len(encoded),
		)
	}
	nodesOffset := int(binary.LittleEndian.Uint32(
		encoded[nodesOffsetField:headerSize],
	))
	if nodesOffset < headerSize ||
		nodesOffset > len(encoded) ||
		(len(encoded)-nodesOffset)%nodeWidth != 0 {
		t.Fatalf(
			"encoded TS-Go AST has invalid node section offset %d",
			nodesOffset,
		)
	}
	return (len(encoded) - nodesOffset) / nodeWidth
}

func assertRuntimeGraph(
	t *testing.T,
	artifacts materialized,
	familyPaths ...string,
) {
	t.Helper()
	panicSource := artifacts.printed["runtime/panic.ts"]
	if strings.Count(panicSource, "export class GoPanic<T>") != 1 {
		t.Fatalf("panic runtime definitions are not exact:\n%s", panicSource)
	}
	for _, path := range familyPaths {
		source := artifacts.printed[path]
		if strings.Count(
			source,
			`import { GoPanic } from "./panic.js";`,
		) != 1 {
			t.Fatalf("%s does not exact-join the panic dependency:\n%s", path, source)
		}
	}
	for path, source := range artifacts.printed {
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
			if strings.Contains(source, forbidden) {
				t.Fatalf("%s contains forbidden %q:\n%s", path, forbidden, source)
			}
		}
	}
}

func typecheck(
	t *testing.T,
	workingDirectory string,
	paths []string,
	runner string,
) {
	t.Helper()
	arguments := []string{
		"--target", "es2022",
		"--module", "nodenext",
		"--moduleResolution", "nodenext",
		"--strict",
		"--outDir", filepath.Join(workingDirectory, "out"),
	}
	arguments = append(arguments, paths...)
	arguments = append(arguments, runner)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
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

func executeGo(t *testing.T, workingDirectory string) string {
	t.Helper()
	runnerDirectory := filepath.Join(workingDirectory, "go-runner")
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
	fmt.Println(values.NumberValue(17, 5))
	fmt.Println(values.StringByte("abc"))
	fmt.Println(values.StringWindow("abcd"))
	fmt.Println(values.PointerValue(7))
	fmt.Println(values.ArrayValue(3))
	fmt.Println(values.SliceValue(4))
	fmt.Println(values.MapValue(5))
	fmt.Println(values.SliceStoreOrder())
	fmt.Println(values.MapStoreOrder())
	fmt.Println(panics(func() { values.PointerPanic() }))
	fmt.Println(panics(func() { values.ArrayPanic(1) }))
	fmt.Println(panics(func() { values.SlicePanic(1) }))
	fmt.Println(panics(func() { values.StringPanic("a", 1) }))
	fmt.Println(panics(values.MapPanic))
	fmt.Println(panics(func() { values.DividePanic(0) }))
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

func sourceBytes(t *testing.T) int {
	t.Helper()
	total := 0
	for _, path := range []string{
		filepath.Join(fixtureDirectory(), "data", "values.go"),
		filepath.Join(fixtureDirectory(), "api", "api.go"),
	} {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		total += len(content)
	}
	return total
}

func run(t *testing.T, directory, name string, arguments ...string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
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

func writeFile(t *testing.T, path, content string) {
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
		"mixed-family",
	)
}

func repositoryRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		panic("resolve mixed-family repository root")
	}
	return filepath.Clean(
		filepath.Join(filepath.Dir(file), "..", "..", "..", ".."),
	)
}
