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
	runtimefixture "github.com/tsoniclang/gotots/internal/testfixture/gototsruntime"
	corefixture "github.com/tsoniclang/gotots/internal/testfixture/tsoniccore"
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

func TestWaveThreeExpressionMatrixPrintsAndTypechecks(
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
			waveThreeTypecheck(t, workingDirectory, artifacts.paths)
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
	paths                      []string
	sourceModule               string
	source                     string
	bytes                      int
	nodes                      int
	largest                    int
	genericConcretizations     int
	genericConcretizationBytes int
	genericCapabilities        int
	genericCapabilityBytes     int
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
		encoded, err := tsgo.EncodeSourceFile(file.SourceFile())
		if err != nil {
			t.Fatal(err)
		}
		result.nodes += waveFourEncodedNodes(t, encoded)
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
		switch {
		case strings.HasPrefix(
			file.OutputPath(),
			"support/generics/concretizations/",
		):
			result.genericConcretizations++
			result.genericConcretizationBytes += len(printed)
		case strings.HasPrefix(
			file.OutputPath(),
			"support/generics/capabilities/",
		):
			result.genericCapabilities++
			result.genericCapabilityBytes += len(printed)
		}
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
	t.Logf(
		"Wave 3 artifacts: files=%d bytes=%d nodes=%d largest=%d concretizations=%d/%d capabilities=%d/%d",
		len(result.paths),
		result.bytes,
		result.nodes,
		result.largest,
		result.genericConcretizations,
		result.genericConcretizationBytes,
		result.genericCapabilities,
		result.genericCapabilityBytes,
	)
	if result.genericConcretizations != 1 ||
		result.genericConcretizationBytes > 850 ||
		result.genericCapabilities != 0 ||
		result.genericCapabilityBytes != 0 {
		t.Fatalf(
			"Wave 3 generic artifact bounds exceeded: concretizations=%d/%d capabilities=%d/%d",
			result.genericConcretizations,
			result.genericConcretizationBytes,
			result.genericCapabilities,
			result.genericCapabilityBytes,
		)
	}
	if result.bytes > 55_000 || result.nodes > 11_250 ||
		result.largest > 25_000 {
		t.Fatalf(
			"Wave 3 artifact bounds exceeded: total=%d nodes=%d largest=%d",
			result.bytes,
			result.nodes,
			result.largest,
		)
	}
	return result
}

func assertWaveThreeOwnerShapes(t *testing.T, source string) {
	t.Helper()
	for _, required := range []string{
		"stores(new Box(1, new Pair(",
		"new Table(GoMap.make<int32, int32>",
		"definedBasicOperators(new Flag(false)",
		"complexValue.$value.real",
		"complexValue.$value.imag",
	} {
		if !strings.Contains(source, required) {
			t.Fatalf("Wave 3 source lacks owner shape %q:\n%s", required, source)
		}
	}
	const tupleFunction = "export function variadicInterfaceTuple()"
	tupleStart := strings.Index(source, tupleFunction)
	if tupleStart < 0 {
		t.Fatalf("Wave 3 source lacks %q:\n%s", tupleFunction, source)
	}
	tupleEnd := strings.Index(source[tupleStart:], "\nexport function builtins(")
	if tupleEnd < 0 {
		t.Fatalf("Wave 3 source lacks variadic tuple function boundary:\n%s", source)
	}
	tupleSource := source[tupleStart : tupleStart+tupleEnd]
	for _, required := range []string{
		"RuntimeSlice.literal<GoInterface",
		"[new GoInterfaceAdapter",
		"(results",
	} {
		if !strings.Contains(tupleSource, required) {
			t.Fatalf(
				"variadic tuple interface transfer lacks %q:\n%s",
				required,
				tupleSource,
			)
		}
	}
}

func waveThreeTypecheck(
	t *testing.T,
	workingDirectory string,
	paths []string,
) {
	t.Helper()
	if err := corefixture.InstallResolutionOnly(workingDirectory); err != nil {
		t.Fatal(err)
	}
	arguments := []string{
		"--target", "es2022",
		"--module", "nodenext",
		"--moduleResolution", "nodenext",
		"--strict",
		"--noFallthroughCasesInSwitch",
		"--noUncheckedIndexedAccess",
		"--outDir", filepath.Join(workingDirectory, "out"),
	}
	arguments = append(arguments, paths...)
	if err := runtimefixture.InstallResolution(workingDirectory, filepath.Join(workingDirectory, "out")); err != nil {
		t.Fatal(err)
	}
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

func TestGenericInterfaceValueKeepsTypeArgumentsAndNil(t *testing.T) {
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
				Directory: genericInterfaceValueDirectory(),
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
			for _, required := range []string{
				"Value<int32> | undefined",
				"extends GoInterfaceValue implements GoInterface, Value__from_genericinterface<int32>",
				"goInterfaceNonNil<Value<int32>>",
				".Get()",
				"($argument0:",
				").Clone()",
			} {
				if !strings.Contains(artifacts.printed, required) {
					t.Fatalf(
						"generic-interface artifact lacks %q:\n%s",
						required,
						artifacts.printed,
					)
				}
			}
			for _, forbidden := range []string{".bind(", ".call(", ".apply("} {
				if strings.Contains(artifacts.printed, forbidden) {
					t.Fatalf(
						"generic-interface artifact contains %q:\n%s",
						forbidden,
						artifacts.printed,
					)
				}
			}
			waveThreeTypecheck(t, workingDirectory, artifacts.paths)
		})
	}
}

func executeGenericInterfaceValueGo(
	t *testing.T,
	workingDirectory string,
) string {
	t.Helper()
	modulePath, err := filepath.Abs(genericInterfaceValueDirectory())
	if err != nil {
		t.Fatal(err)
	}
	runnerDirectory := filepath.Join(
		workingDirectory,
		"go-runner-generic-interface",
	)
	writeProgramFile(
		t,
		filepath.Join(runnerDirectory, "go.mod"),
		fmt.Sprintf(
			`module example.com/runner

go 1.26.4

require example.com/genericinterface v0.0.0

replace example.com/genericinterface => %s
`,
			filepath.ToSlash(modulePath),
		),
	)
	writeProgramFile(
		t,
		filepath.Join(runnerDirectory, "main.go"),
		`package main

import (
	"fmt"

	values "example.com/genericinterface"
)

func main() {
	fmt.Println(values.Audit())
}
`,
	)
	return runProgram(
		t,
		runnerDirectory,
		filepath.Join(runtime.GOROOT(), "bin", "go"),
		"run",
		".",
	)
}

func genericInterfaceValueDirectory() string {
	return filepath.Join(
		repositoryRoot(),
		"testdata",
		"constructs",
		"interface",
		"generic-value",
	)
}
