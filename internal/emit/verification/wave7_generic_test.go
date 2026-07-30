package emit_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
)

func TestWaveSevenGenericFoundationCompilesThroughPublicPipeline(t *testing.T) {
	for _, testCase := range []struct {
		name    string
		options emit.Options
	}{
		{name: "number", options: emit.DefaultOptions()},
		{
			name: "bigint",
			options: emit.Options{
				IntegerRepresentation: emit.IntegerRepresentationBigInt,
				EvaluationOrder:       emit.EvaluationOrderPreserveGo,
			},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			program, err := load.Load(context.Background(), load.Request{
				Directory: waveSevenGenericDirectory(),
				Pattern:   ".",
			})
			if err != nil {
				t.Fatal(err)
			}
			root, err := emit.NewRoot(
				program.Roots()[0].Types().Scope().Lookup("AuditFunctions"),
			)
			if err != nil {
				t.Fatal(err)
			}
			emission, err := emit.CompileWithOptions(
				program,
				[]emit.Root{root},
				testCase.options,
			)
			if err != nil {
				t.Fatal(err)
			}
			workingDirectory := t.TempDir()
			artifacts := materializeArtifacts(
				t,
				emission,
				workingDirectory,
			)
			assertWaveSevenGenericFoundationShape(t, artifacts.printed)
			runner := filepath.Join(workingDirectory, "runner.ts")
			writeProgramFile(t, runner, `import "./program.js";
import { AuditFunctions } from "`+artifacts.sourceModule+`";

const values = AuditFunctions();
const output: string[] = [];
for (let index = 0; index < values.length; index++) {
    output.push(String(values.get(index)));
}
console.log(output.join(" "));
`)
			writeProgramFile(
				t,
				filepath.Join(workingDirectory, "package.json"),
				"{\"type\":\"module\"}\n",
			)
			paths := append(artifacts.paths, runner)
			waveThreeTypecheck(t, workingDirectory, paths)
			targetOutput := runProgram(
				t,
				workingDirectory,
				"node",
				filepath.Join(workingDirectory, "out", "runner.js"),
			)
			goOutput := executeWaveSevenGenericGo(
				t,
				workingDirectory,
				"AuditFunctions",
			)
			if targetOutput != goOutput {
				t.Fatalf(
					"Wave 7 generic output differs\nTypeScript:\n%s\nGo:\n%s",
					targetOutput,
					goOutput,
				)
			}
		})
	}
}

func TestWaveSevenGenericNamedTypesCompileThroughPublicPipeline(t *testing.T) {
	for _, testCase := range []struct {
		name    string
		options emit.Options
	}{
		{name: "number", options: emit.DefaultOptions()},
		{
			name: "bigint",
			options: emit.Options{
				IntegerRepresentation: emit.IntegerRepresentationBigInt,
				EvaluationOrder:       emit.EvaluationOrderPreserveGo,
			},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			program, err := load.Load(context.Background(), load.Request{
				Directory: waveSevenGenericDirectory(),
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
			emission, err := emit.CompileWithOptions(
				program,
				[]emit.Root{root},
				testCase.options,
			)
			if err != nil {
				t.Fatal(err)
			}
			workingDirectory := t.TempDir()
			artifacts := materializeArtifacts(t, emission, workingDirectory)
			local := targetFunctionText(
				t,
				artifacts.printed,
				"LocalTypeCapability",
			)
			if !strings.Contains(local, "function $goCapability_") ||
				strings.Contains(local, "export function $goCapability_") {
				t.Fatalf(
					"local-type capability is not lexical and unexported:\n%s",
					local,
				)
			}
			sourceModule := sourceModuleForExport(
				t,
				artifacts,
				workingDirectory,
				"Audit",
			)
			runner := filepath.Join(workingDirectory, "runner.ts")
			writeProgramFile(t, runner, `import "./program.js";
import { Audit } from "`+sourceModule+`";

const values = Audit();
const output: string[] = [];
for (let index = 0; index < values.length; index++) {
    output.push(String(values.get(index)));
}
console.log(output.join(" "));
`)
			writeProgramFile(
				t,
				filepath.Join(workingDirectory, "package.json"),
				"{\"type\":\"module\"}\n",
			)
			paths := append(artifacts.paths, runner)
			waveThreeTypecheck(t, workingDirectory, paths)
			targetOutput := runProgram(
				t,
				workingDirectory,
				"node",
				filepath.Join(workingDirectory, "out", "runner.js"),
			)
			goOutput := executeWaveSevenGenericGo(
				t,
				workingDirectory,
				"Audit",
			)
			if targetOutput != goOutput {
				t.Fatalf(
					"Wave 7 named-generic output differs\nTypeScript:\n%s\nGo:\n%s",
					targetOutput,
					goOutput,
				)
			}
		})
	}
}

func TestWaveSevenGenericAssertionsCompileThroughPublicPipeline(t *testing.T) {
	for _, testCase := range []struct {
		name    string
		options emit.Options
	}{
		{name: "number", options: emit.DefaultOptions()},
		{
			name: "bigint",
			options: emit.Options{
				IntegerRepresentation: emit.IntegerRepresentationBigInt,
				EvaluationOrder:       emit.EvaluationOrderPreserveGo,
			},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			program, err := load.Load(context.Background(), load.Request{
				Directory: waveSevenGenericDirectory(),
				Pattern:   ".",
			})
			if err != nil {
				t.Fatal(err)
			}
			root, err := emit.NewRoot(
				program.Roots()[0].Types().Scope().Lookup(
					"AuditGenericAssertions",
				),
			)
			if err != nil {
				t.Fatal(err)
			}
			emission, err := emit.CompileWithOptions(
				program,
				[]emit.Root{root},
				testCase.options,
			)
			if err != nil {
				t.Fatal(err)
			}
			workingDirectory := t.TempDir()
			artifacts := materializeArtifacts(t, emission, workingDirectory)
			for _, required := range []string{
				"export function AssertValue<T>",
				"export function MustAssertValue<T>",
				"export function TypeSwitchValue<T>",
				"$go$interface_assert_",
				"$go$interface_assert_ok_",
			} {
				if !strings.Contains(artifacts.printed, required) {
					t.Fatalf(
						"generic assertion artifacts lack %q:\n%s",
						required,
						artifacts.printed,
					)
				}
			}
			runner := filepath.Join(workingDirectory, "runner.ts")
			writeProgramFile(t, runner, `import "./program.js";
import { AuditGenericAssertions } from "`+artifacts.sourceModule+`";

const values = AuditGenericAssertions();
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
			targetOutput := runProgram(
				t,
				workingDirectory,
				"node",
				filepath.Join(workingDirectory, "out", "runner.js"),
			)
			goOutput := executeWaveSevenGenericGo(
				t,
				workingDirectory,
				"AuditGenericAssertions",
			)
			if targetOutput != goOutput {
				t.Fatalf(
					"generic assertion output differs\nTypeScript:\n%s\nGo:\n%s",
					targetOutput,
					goOutput,
				)
			}
		})
	}
}

func TestNonGenericAliasToGenericInstantiationCompilesInPointerField(
	t *testing.T,
) {
	project := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(project, "go.mod"),
		"module example.com/genericalias\n\ngo 1.26.4\n",
	)
	writeProgramFile(
		t,
		filepath.Join(project, "source.go"),
		`package genericalias

type Box[T any] struct {
	Value T
}

type IntBox = Box[int]

type Holder struct {
	Box *IntBox
}

func EmptyHolder() Holder {
	return Holder{}
}
`,
	)
	program, err := load.Load(context.Background(), load.Request{
		Directory: project,
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	root, err := emit.NewRoot(
		program.Roots()[0].Types().Scope().Lookup("EmptyHolder"),
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
	if !strings.Contains(
		artifacts.printed,
		"GoPointer<Box<int64, int64>, Box$Storage<int64, int64>>",
	) {
		t.Fatalf(
			"alias to generic instantiation was not canonicalized:\n%s",
			artifacts.printed,
		)
	}
	waveThreeTypecheck(t, workingDirectory, artifacts.paths)
}

func sourceModuleForExport(
	t *testing.T,
	artifacts waveFourArtifacts,
	workingDirectory string,
	name string,
) string {
	t.Helper()
	var selected string
	for _, path := range artifacts.paths {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		printed := string(content)
		if !strings.Contains(
			printed,
			"export function "+name+"(",
		) && !strings.Contains(
			printed,
			"export async function "+name+"(",
		) {
			continue
		}
		if selected != "" {
			t.Fatalf("multiple source modules export %s", name)
		}
		selected = path
	}
	if selected == "" {
		t.Fatalf("no source module exports %s", name)
	}
	relative, err := filepath.Rel(workingDirectory, selected)
	if err != nil {
		t.Fatal(err)
	}
	return "./" + strings.TrimSuffix(filepath.ToSlash(relative), ".ts") + ".js"
}

func assertWaveSevenGenericFoundationShape(t *testing.T, printed string) {
	t.Helper()
	for _, required := range []string{
		"export function Identity<T>",
		"export function Add<T>",
		"export function Zero<T>",
		"export function ZeroFromNew<T>",
		"export function Equal<T>",
		"export function Twice<T>",
		"$goCapability_",
	} {
		if !strings.Contains(printed, required) {
			t.Fatalf("Wave 7 generic artifacts lack %q:\n%s", required, printed)
		}
	}
	for _, forbidden := range []string{
		"class GoValueOps",
		"interface GoValueOps",
		"Record<string",
		"switch (typeof",
	} {
		if strings.Contains(printed, forbidden) {
			t.Fatalf(
				"Wave 7 generic artifacts contain %q:\n%s",
				forbidden,
				printed,
			)
		}
	}
	if strings.Count(printed, "export function Add<T>") != 1 {
		t.Fatalf("generic Add body was duplicated:\n%s", printed)
	}
	zeroFromNew := targetGenericFunctionText(t, printed, "ZeroFromNew")
	if !strings.Contains(zeroFromNew, "$go$zero_") ||
		strings.Contains(zeroFromNew, "GoPointer") ||
		strings.Contains(zeroFromNew, "$go$pointer_") {
		t.Fatalf(
			"generic *new(T) did not lower directly to its zero owner:\n%s",
			zeroFromNew,
		)
	}
	if strings.Contains(printed, "GoPointer.field<T, Box") {
		t.Fatalf(
			"direct generic pointer-receiver field assignment formed an interior pointer:\n%s",
			printed,
		)
	}
	twice := targetGenericFunctionText(t, printed, "Twice")
	copyNames := regexp.MustCompile(`\$go\$copy_[0-9a-f]+`).
		FindAllString(twice, -1)
	if len(copyNames) != 3 ||
		copyNames[0] != copyNames[1] ||
		copyNames[1] != copyNames[2] {
		t.Fatalf(
			"generic forwarding did not exact-join repeated copy capability: %v\n%s",
			copyNames,
			twice,
		)
	}
}

func targetGenericFunctionText(
	t *testing.T,
	printed string,
	name string,
) string {
	t.Helper()
	startMarker := "export function " + name + "<"
	start := strings.Index(printed, startMarker)
	if start < 0 {
		t.Fatalf("Wave 7 artifacts lack generic function %s", name)
	}
	remaining := printed[start+len(startMarker):]
	end := strings.Index(remaining, "\nexport function ")
	if end < 0 {
		return printed[start:]
	}
	return printed[start : start+len(startMarker)+end]
}

func executeWaveSevenGenericGo(
	t *testing.T,
	workingDirectory string,
	function string,
) string {
	t.Helper()
	modulePath, err := filepath.Abs(waveSevenGenericDirectory())
	if err != nil {
		t.Fatal(err)
	}
	runnerDirectory := filepath.Join(workingDirectory, "go-runner-wave7")
	writeProgramFile(t, filepath.Join(runnerDirectory, "go.mod"), fmt.Sprintf(
		`module example.com/runner

go 1.26.4

require example.com/wave7generics v0.0.0

replace example.com/wave7generics => %s
`,
		filepath.ToSlash(modulePath),
	))
	writeProgramFile(t, filepath.Join(runnerDirectory, "main.go"), fmt.Sprintf(`package main

import (
	"fmt"

	values "example.com/wave7generics"
)

func main() {
	for index, value := range values.%s() {
		if index != 0 {
			fmt.Print(" ")
		}
		fmt.Print(value)
	}
	fmt.Println()
}
`, function))
	return runProgram(
		t,
		runnerDirectory,
		filepath.Join(runtime.GOROOT(), "bin", "go"),
		"run",
		".",
	)
}

func waveSevenGenericDirectory() string {
	return filepath.Join(
		repositoryRoot(),
		"testdata",
		"constructs",
		"generic",
		"wave7",
	)
}
