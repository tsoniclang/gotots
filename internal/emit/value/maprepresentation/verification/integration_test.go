package maprepresentation_test

import (
	"context"
	"fmt"
	"go/types"
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
)

func TestMapValuesCreateTypedTargetAST(t *testing.T) {
	loaded := loadMapValuesProject(t)
	emission := compileExported(t, loaded)
	sourceFile := targetFileBySuffix(
		t,
		emission.Files(),
		"source.ts",
	).SourceFile()

	nilLength := targetFunction(t, sourceFile, "NilLength")
	declarations := nilLength.Body().(tsgo.Block).Statements()
	if len(declarations) != 2 {
		t.Fatalf("NilLength statements = %d, want declaration and return", len(declarations))
	}
	local := declarations[0].(tsgo.VariableStatement).
		DeclarationList().
		Declarations()[0]
	if local.Initializer().Kind() != tsgo.SyntaxKindCallExpression {
		t.Fatalf("map zero = %d, want runtime nil constructor call", local.Initializer().Kind())
	}
	identity := targetFunction(t, sourceFile, "Identity")
	mapType, ok := identity.Parameters()[0].Type().(tsgo.TypeReferenceNode)
	if !ok {
		t.Fatalf("map parameter type = %T, want TypeReferenceNode", identity.Parameters()[0].Type())
	}
	if mapType.TypeName().(tsgo.Identifier).Text() != "GoMap" ||
		len(mapType.TypeArguments()) != 2 {
		t.Fatal("map type is not the typed two-argument runtime class")
	}
	missing := targetFunction(t, sourceFile, "Missing")
	missingStatements := missing.Body().(tsgo.Block).Statements()
	lookup := missingStatements[len(missingStatements)-1].(tsgo.ReturnStatement).
		Expression().(tsgo.CallExpression)
	if lookup.Expression().Kind() != tsgo.SyntaxKindPropertyAccessExpression {
		t.Fatalf("map lookup callee = %d", lookup.Expression().Kind())
	}

	runtimeFile := targetFileBySuffix(t, emission.Files(), "runtime/map.ts")
	var class tsgo.ClassDeclaration
	for _, statement := range runtimeFile.SourceFile().Statements() {
		candidate, ok := statement.(tsgo.ClassDeclaration)
		if ok && candidate.Name().Text() == "GoMap" {
			class = candidate
		}
	}
	if class == nil {
		t.Fatal("runtime map declaration is absent")
	}
	if class.Name().Text() != "GoMap" ||
		len(class.TypeParameters()) != 2 ||
		len(class.Members()) != 9 {
		t.Fatalf(
			"runtime class = %q with %d parameters and %d members",
			class.Name().Text(),
			len(class.TypeParameters()),
			len(class.Members()),
		)
	}
}

func TestMapValuesPrintTypecheckAndExecuteDifferentially(t *testing.T) {
	loaded := loadMapValuesProject(t)
	workingDirectory := t.TempDir()
	artifacts := materialize(t, compileExported(t, loaded), workingDirectory)
	runtimeSource := readFile(t, artifacts.file(t, "runtime/map.ts"))
	for _, forbidden := range []string{
		"any",
		"unknown",
		"Record<",
		"Object.create",
		".call(",
		".apply(",
		".bind(",
		"get(key)!",
		"clear(): void",
		"goMapClear",
	} {
		if strings.Contains(runtimeSource, forbidden) {
			t.Fatalf("runtime map artifact contains %q:\n%s", forbidden, runtimeSource)
		}
	}
	for _, required := range []string{
		"class GoMap<K extends boolean | number | bigint | string, V>",
		"Map<K, V>",
		"const storedValue = storage.get(key);",
		"storedValue === undefined",
		"return this.zeroValue;",
		"assignment to entry in nil map",
	} {
		if !strings.Contains(runtimeSource, required) {
			t.Fatalf("runtime map artifact lacks %q:\n%s", required, runtimeSource)
		}
	}

	goOutput := executeMapValuesGo(t, workingDirectory)
	typeScriptOutput := executeMapValuesTypeScript(
		t,
		artifacts,
		workingDirectory,
	)
	if typeScriptOutput != goOutput {
		t.Fatalf("TypeScript output = %q, Go output = %q", typeScriptOutput, goOutput)
	}
	lines := strings.Split(strings.TrimSpace(typeScriptOutput), "\n")
	if len(lines) != 19 {
		t.Fatalf("differential output lines = %d, want 19", len(lines))
	}
	if lines[0] != "0" {
		t.Fatalf("missing-value mutation guard = %q, want scalar zero", lines[0])
	}
	if lines[3] != "31" {
		t.Fatalf("copy-on-assignment mutation guard = %q, want aliased store", lines[3])
	}
	if lines[18] != "true" {
		t.Fatalf("nil-write mutation guard = %q, want failure", lines[18])
	}
}

func TestMapValuesStrictTypecheckUnderBigIntProfile(t *testing.T) {
	loaded := loadMapValuesProject(t)
	roots, err := emit.ExportedAPIRoots(loaded)
	if err != nil {
		t.Fatal(err)
	}
	options := emit.DefaultOptions()
	options.IntegerRepresentation = emit.IntegerRepresentationBigInt
	emission, err := emit.CompileWithOptions(loaded.Program(), roots, options)
	if err != nil {
		t.Fatal(err)
	}
	workingDirectory := t.TempDir()
	artifacts := materialize(t, emission, workingDirectory)
	strictTypecheck(t, artifacts, workingDirectory)
	source := readFile(t, artifacts.file(t, "source.ts"))
	if !strings.Contains(source, "BigInt(values.length())") {
		t.Fatalf("bigint map length does not preserve the selected int representation:\n%s", source)
	}
}

func TestMapBuiltinSelectionUsesGoObjectIdentity(t *testing.T) {
	loaded := loadMapValuesProject(t)
	targets := map[types.Object]string{
		types.Universe.Lookup("make"):   "forgedMakeSpelling",
		types.Universe.Lookup("len"):    "forgedLenSpelling",
		types.Universe.Lookup("delete"): "forgedDeleteSpelling",
	}
	mutations := 0
	for identifier, object := range loaded.TypesInfo().Uses {
		if replacement := targets[object]; replacement != "" {
			identifier.Name = replacement
			mutations++
		}
	}
	if mutations < 3 {
		t.Fatalf("builtin identity mutations = %d, want make/len/delete", mutations)
	}
	if _, err := compileExportedResult(loaded); err != nil {
		t.Fatalf("identity-preserving spelling mutation failed: %v", err)
	}
}

type materialized struct {
	paths   []string
	modules map[string]string
}

func compileExported(t *testing.T, loaded *load.Package) emit.ProgramEmission {
	t.Helper()
	emission, err := compileExportedResult(loaded)
	if err != nil {
		t.Fatal(err)
	}
	return emission
}

func compileExportedResult(
	loaded *load.Package,
) (emit.ProgramEmission, error) {
	roots, err := emit.ExportedAPIRoots(loaded)
	if err != nil {
		return emit.ProgramEmission{}, err
	}
	emission, err := emit.Compile(loaded.Program(), roots)
	if err != nil {
		return emit.ProgramEmission{}, err
	}
	return emission, nil
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
	result := materialized{modules: make(map[string]string)}
	for _, file := range emission.Files() {
		printed, err := client.PrintNode(file.SourceFile(), tsgo.PrintOptions{})
		if err != nil {
			t.Fatal(err)
		}
		targetPath := filepath.Join(
			workingDirectory,
			filepath.FromSlash(file.OutputPath()),
		)
		writeFile(t, targetPath, printed)
		result.paths = append(result.paths, targetPath)
		if file.Kind() == emit.TargetFileSource {
			result.modules[filepath.Base(file.OutputPath())] = "./" +
				strings.TrimSuffix(file.OutputPath(), ".ts") + ".js"
		}
	}
	return result
}

func (m materialized) file(t *testing.T, suffix string) string {
	t.Helper()
	for _, path := range m.paths {
		if strings.HasSuffix(filepath.ToSlash(path), suffix) {
			return path
		}
	}
	t.Fatalf("materialized file with suffix %q is absent", suffix)
	return ""
}

func (m materialized) module(t *testing.T, base string) string {
	t.Helper()
	module := m.modules[base]
	if module == "" {
		t.Fatalf("source module %q is absent", base)
	}
	return module
}

func executeMapValuesGo(t *testing.T, workingDirectory string) string {
	t.Helper()
	modulePath, err := filepath.Abs(mapValuesProjectDirectory())
	if err != nil {
		t.Fatal(err)
	}
	runnerDirectory := filepath.Join(workingDirectory, "go-runner")
	writeFile(t, filepath.Join(runnerDirectory, "go.mod"), fmt.Sprintf(`module example.com/runner

go 1.26.4

require example.com/mapvalues v0.0.0

replace example.com/mapvalues => %s
`, filepath.ToSlash(modulePath)))
	writeFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"

	mapvalues "example.com/mapvalues"
)

func nilWriteFails() (failed bool) {
	defer func() { failed = recover() != nil }()
	mapvalues.NilWrite()
	return false
}

func main() {
	fmt.Println(mapvalues.Missing())
	fmt.Println(mapvalues.Lookup(1))
	fmt.Println(mapvalues.Lookup(7))
	fmt.Println(mapvalues.Alias())
	fmt.Println(mapvalues.ThroughCall())
	fmt.Println(mapvalues.AliasMake())
	fmt.Println(mapvalues.DeleteAndLen())
	fmt.Println(mapvalues.BoolKey())
	fmt.Println(mapvalues.IndexedUpdates())
	fmt.Println(mapvalues.LiteralOrder())
	fmt.Println(mapvalues.NilLength())
	fmt.Println(mapvalues.ExplicitNil() == nil)
	fmt.Println(mapvalues.ResetToNil())
	fmt.Println(mapvalues.NilComparisons())
	fmt.Println(len(mapvalues.MakeSized(2)))
	fmt.Println(mapvalues.SizeEvaluated())
	fmt.Println(mapvalues.PackageState(8))
	fmt.Println(mapvalues.PackageState(9))
	fmt.Println(nilWriteFails())
}
`)
	return run(t, runnerDirectory, filepath.Join(runtime.GOROOT(), "bin", "go"), "run", ".")
}

func executeMapValuesTypeScript(
	t *testing.T,
	artifacts materialized,
	workingDirectory string,
) string {
	t.Helper()
	runnerPath := filepath.Join(workingDirectory, "runner.ts")
	writeFile(t, runnerPath, `import {
	    Alias,
	    AliasMake,
    BoolKey,
    DeleteAndLen,
    ExplicitNil,
    IndexedUpdates,
    LiteralOrder,
    Lookup,
    MakeSized,
    Missing,
    NilComparisons,
    NilLength,
    NilWrite,
    PackageState,
    ResetToNil,
    SizeEvaluated,
    ThroughCall,
} from "`+artifacts.module(t, "source.ts")+`";
import "./program.js";

console.log(Missing());
console.log(...Lookup(1));
console.log(...Lookup(7));
console.log(Alias());
console.log(ThroughCall());
console.log(AliasMake());
console.log(...DeleteAndLen());
console.log(BoolKey());
console.log(IndexedUpdates());
console.log(LiteralOrder());
console.log(NilLength());
console.log(ExplicitNil().isNil());
console.log(ResetToNil());
console.log(...NilComparisons());
console.log(MakeSized(2).length());
console.log(SizeEvaluated());
console.log(...PackageState(8));
console.log(...PackageState(9));
let nilWriteFailed = false;
try { NilWrite(); } catch { nilWriteFailed = true; }
console.log(nilWriteFailed);
`)
	writeFile(t, filepath.Join(workingDirectory, "package.json"), "{\"type\":\"module\"}\n")
	strictTypecheckWithRunner(t, artifacts, workingDirectory, runnerPath)
	return run(t, workingDirectory, "node", filepath.Join(workingDirectory, "out", "runner.js"))
}

func strictTypecheck(
	t *testing.T,
	artifacts materialized,
	workingDirectory string,
) {
	t.Helper()
	strictTypecheckWithRunner(t, artifacts, workingDirectory, "")
}

func strictTypecheckWithRunner(
	t *testing.T,
	artifacts materialized,
	workingDirectory string,
	runnerPath string,
) {
	t.Helper()
	outputDirectory := filepath.Join(workingDirectory, "out")
	arguments := []string{
		"--target", "es2022",
		"--module", "nodenext",
		"--moduleResolution", "nodenext",
		"--strict",
		"--outDir", outputDirectory,
	}
	arguments = append(arguments, artifacts.paths...)
	if runnerPath != "" {
		arguments = append(arguments, runnerPath)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
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

func loadMapValuesProject(t *testing.T) *load.Package {
	t.Helper()
	loaded, err := load.One(context.Background(), load.Request{
		Directory: mapValuesProjectDirectory(),
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	return loaded
}

func targetFileBySuffix(
	t *testing.T,
	files []emit.TargetFile,
	suffix string,
) emit.TargetFile {
	t.Helper()
	for _, file := range files {
		if strings.HasSuffix(file.OutputPath(), suffix) {
			return file
		}
	}
	t.Fatalf("target file with suffix %q is absent", suffix)
	return emit.TargetFile{}
}

func targetFunction(
	t *testing.T,
	file tsgo.SourceFile,
	name string,
) tsgo.FunctionDeclaration {
	t.Helper()
	for _, statement := range file.Statements() {
		function, ok := statement.(tsgo.FunctionDeclaration)
		if ok && function.Name().Text() == name {
			return function
		}
	}
	t.Fatalf("target function %q is absent", name)
	return nil
}

func run(
	t *testing.T,
	directory string,
	name string,
	arguments ...string,
) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, name, arguments...)
	command.Dir = directory
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v: %v\n%s", name, arguments, err, output)
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

func readFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(content)
}

func repositoryRoot() string {
	_, current, _, ok := runtime.Caller(0)
	if !ok {
		panic("resolve repository root")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(current), "../../../../.."))
}

func mapValuesProjectDirectory() string {
	return filepath.Join(
		repositoryRoot(),
		"testdata",
		"constructs",
		"value",
		"map",
		"basic",
	)
}
