package emit_test

import (
	"bytes"
	"context"
	"fmt"
	"go/ast"
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

func TestWaveThreeNamedMapFamilyIgnoresCompositeTypeSpelling(t *testing.T) {
	program, err := load.Load(context.Background(), load.Request{
		Directory: waveThreeExpressionDirectory(),
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	roots, err := emit.ExportedAPIRoots(program.Roots()[0])
	if err != nil {
		t.Fatal(err)
	}
	baseline, err := emit.Compile(program, roots)
	if err != nil {
		t.Fatal(err)
	}
	var selected *ast.Ident
	ast.Inspect(program.Roots()[0].Files()[0].Syntax(), func(node ast.Node) bool {
		literal, ok := node.(*ast.CompositeLit)
		if !ok {
			return true
		}
		identifier, ok := literal.Type.(*ast.Ident)
		if ok && identifier.Name == "Table" && len(literal.Elts) != 0 {
			selected = identifier
			return false
		}
		return true
	})
	if selected == nil {
		t.Fatal("named map literal mutation target is absent")
	}
	selected.Name = "SpellingMustNotSelectTheMapOwner"
	mutated, err := emit.Compile(program, roots)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(
		encodeWaveThreeProgram(t, baseline),
		encodeWaveThreeProgram(t, mutated),
	) {
		t.Fatal("named map family changed after a source-spelling-only mutation")
	}
}

func TestWaveThreeExpressionMatrixPrintsTypechecksAndMatchesGo(
	t *testing.T,
) {
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
				Directory: waveThreeExpressionDirectory(),
				Pattern:   ".",
			})
			if err != nil {
				t.Fatal(err)
			}
			roots, err := emit.ExportedAPIRoots(program.Roots()[0])
			if err != nil {
				t.Fatal(err)
			}
			emission, err := emit.CompileWithOptions(
				program,
				roots,
				testCase.options,
			)
			if err != nil {
				t.Fatal(err)
			}
			workingDirectory := t.TempDir()
			artifacts := materializeWaveThreeExpressions(
				t,
				emission,
				workingDirectory,
			)
			assertWaveThreeOwnerShapes(t, artifacts.source)
			runner := filepath.Join(workingDirectory, "runner.ts")
			writeProgramFile(t, runner, `import "./program.js";
import { Audit } from "`+artifacts.sourceModule+`";

console.log(Audit().map(String).join(" "));
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
			goOutput := executeWaveThreeGo(t, workingDirectory)
			if targetOutput != goOutput {
				t.Fatalf(
					"Wave 3 output differs\nTypeScript:\n%s\nGo:\n%s",
					targetOutput,
					goOutput,
				)
			}
			t.Logf(
				"Wave 3 matrix: files=%d bytes=%d largest=%d",
				len(artifacts.paths),
				artifacts.bytes,
				artifacts.largest,
			)
		})
	}
}

func encodeWaveThreeProgram(
	t *testing.T,
	emission emit.ProgramEmission,
) []byte {
	t.Helper()
	var result []byte
	for _, file := range emission.Files() {
		result = append(result, file.OutputPath()...)
		result = append(result, 0)
		encoded, err := tsgo.EncodeSourceFile(file.SourceFile())
		if err != nil {
			t.Fatal(err)
		}
		result = append(result, encoded...)
	}
	return result
}

type waveThreeArtifacts struct {
	paths        []string
	sourceModule string
	source       string
	bytes        int
	largest      int
}

func materializeWaveThreeExpressions(
	t *testing.T,
	emission emit.ProgramEmission,
	workingDirectory string,
) waveThreeArtifacts {
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
	result := waveThreeArtifacts{}
	var largest []int
	for _, file := range emission.Files() {
		printed, err := client.PrintNode(
			file.SourceFile(),
			tsgo.PrintOptions{},
		)
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
		result.bytes += len(printed)
		largest = append(largest, len(printed))
		if file.Kind() == emit.TargetFileSource {
			if result.sourceModule != "" {
				t.Fatal("Wave 3 fixture emitted multiple source modules")
			}
			result.source = printed
			result.sourceModule = "./" +
				strings.TrimSuffix(file.OutputPath(), ".ts") +
				".js"
		}
	}
	if result.sourceModule == "" {
		t.Fatal("Wave 3 fixture emitted no source module")
	}
	sort.Sort(sort.Reverse(sort.IntSlice(largest)))
	result.largest = largest[0]
	if result.bytes > 55_000 || result.largest > 22_000 {
		t.Fatalf(
			"Wave 3 artifact bounds exceeded: total=%d largest=%d",
			result.bytes,
			result.largest,
		)
	}
	return result
}

func assertWaveThreeOwnerShapes(t *testing.T, source string) {
	t.Helper()
	for _, required := range []string{
		"stores(Box.$make(",
		"Table.$wrapMap(GoMap.make<int32, int32>",
		"definedBasicOperators(new Flag(false)",
		"complexValue.$value.real",
		"complexValue.$value.imag",
	} {
		if !strings.Contains(source, required) {
			t.Fatalf("Wave 3 source lacks owner shape %q:\n%s", required, source)
		}
	}
}

func waveThreeTypecheck(
	t *testing.T,
	workingDirectory string,
	paths []string,
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

func executeWaveThreeGo(t *testing.T, workingDirectory string) string {
	t.Helper()
	modulePath, err := filepath.Abs(waveThreeExpressionDirectory())
	if err != nil {
		t.Fatal(err)
	}
	runnerDirectory := filepath.Join(workingDirectory, "go-runner-wave3")
	writeProgramFile(t, filepath.Join(runnerDirectory, "go.mod"), fmt.Sprintf(
		`module example.com/runner

go 1.26.4

require example.com/wave3expressions v0.0.0

replace example.com/wave3expressions => %s
`,
		filepath.ToSlash(modulePath),
	))
	writeProgramFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"

	values "example.com/wave3expressions"
)

func main() {
	fmt.Println(values.Audit())
}
`)
	return runProgram(
		t,
		runnerDirectory,
		filepath.Join(runtime.GOROOT(), "bin", "go"),
		"run",
		".",
	)
}

func waveThreeExpressionDirectory() string {
	return filepath.Join(
		repositoryRoot(),
		"testdata",
		"constructs",
		"expression",
		"wave3",
	)
}
