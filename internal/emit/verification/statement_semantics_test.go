package emit_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestWaveFourStatementsPrintTypecheckAndMatchGo(t *testing.T) {
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
				Directory: waveFourStatementDirectory(),
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
			artifacts := materializeArtifacts(
				t,
				emission,
				workingDirectory,
			)
			if artifacts.bytes > 65_000 || artifacts.largest > 28_000 {
				t.Fatalf(
					"Wave 4 artifact bounds exceeded: total=%d largest=%d",
					artifacts.bytes,
					artifacts.largest,
				)
			}
			assertWaveFourArtifactShape(t, artifacts.printed)
			runner := filepath.Join(workingDirectory, "runner.ts")
			writeProgramFile(t, runner, `import "./program.js";
import { Audit } from "`+artifacts.sourceModule+`";

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
			goOutput := executeWaveFourGo(t, workingDirectory)
			if targetOutput != goOutput {
				t.Fatalf(
					"Wave 4 output differs\nTypeScript:\n%s\nGo:\n%s",
					targetOutput,
					goOutput,
				)
			}
			t.Logf(
				"Wave 4 matrix: files=%d bytes=%d largest=%d",
				len(artifacts.paths),
				artifacts.bytes,
				artifacts.largest,
			)
		})
	}
}

type waveFourArtifacts struct {
	paths         []string
	sourceModule  string
	bytes         int
	nodes         int
	largest       int
	sizes         []artifactSize
	printed       string
	printedByKind map[emit.TargetFileKind][]string
}

type artifactSize struct {
	path  string
	bytes int
	nodes int
}

func materializeArtifacts(
	t *testing.T,
	emission emit.ProgramEmission,
	workingDirectory string,
) waveFourArtifacts {
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
	result := waveFourArtifacts{
		printedByKind: make(map[emit.TargetFileKind][]string),
	}
	for _, file := range emission.Files() {
		printed, err := client.PrintNode(file.SourceFile(), tsgo.PrintOptions{})
		if err != nil {
			t.Fatal(err)
		}
		encoded, err := tsgo.EncodeSourceFile(file.SourceFile())
		if err != nil {
			t.Fatal(err)
		}
		nodes := waveFourEncodedNodes(t, encoded)
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
		result.nodes += nodes
		result.printed += "\n// " + file.OutputPath() + "\n" + printed
		result.printedByKind[file.Kind()] = append(
			result.printedByKind[file.Kind()],
			printed,
		)
		result.sizes = append(result.sizes, artifactSize{
			path:  file.OutputPath(),
			bytes: len(printed),
			nodes: nodes,
		})
		if file.Kind() == emit.TargetFileSource {
			result.sourceModule = "./" +
				strings.TrimSuffix(file.OutputPath(), ".ts") +
				".js"
		}
	}
	if result.sourceModule == "" || len(result.sizes) == 0 {
		t.Fatal("Wave 4 fixture emitted no source module")
	}
	sort.Slice(result.sizes, func(left, right int) bool {
		if result.sizes[left].bytes != result.sizes[right].bytes {
			return result.sizes[left].bytes > result.sizes[right].bytes
		}
		return result.sizes[left].path < result.sizes[right].path
	})
	result.largest = result.sizes[0].bytes
	return result
}

func assertWaveFourArtifactShape(t *testing.T, printed string) {
	t.Helper()
	for _, required := range []string{
		"__gotots_switch_selection_",
		"__gotots_range_keys_",
		".keys()",
		"GoDenseIndex.get(__gotots_range_keys_",
		"outer:",
		"continue outer;",
	} {
		if !strings.Contains(printed, required) {
			t.Fatalf("Wave 4 artifacts lack %q:\n%s", required, printed)
		}
	}
	if direct := regexp.MustCompile(
		`__gotots_range_keys_[0-9]+\[__gotots_range_index_[0-9]+\]`,
	); direct.MatchString(printed) {
		t.Fatalf("map range retained a direct dense-key read:\n%s", printed)
	}
	rangeFunction := targetFunctionText(
		t,
		printed,
		"nonconstantArraySourceEvaluates",
	)
	constantRangeFunction := targetFunctionText(
		t,
		printed,
		"constantLengthDoesNotEvaluate",
	)
	rangeCapture := regexp.MustCompile(`const __gotots_range_[0-9]+ =`)
	if rangeCapture.MatchString(constantRangeFunction) {
		t.Fatalf(
			"constant-length key-only range evaluated its operand:\n%s",
			constantRangeFunction,
		)
	}
	if !rangeCapture.MatchString(rangeFunction) {
		t.Fatalf(
			"non-constant array range did not capture its operand:\n%s",
			rangeFunction,
		)
	}
	callee := regexp.MustCompile(
		`const (__gotots_callee_[0-9]+) = makeArray;`,
	).FindStringSubmatch(rangeFunction)
	if len(callee) != 2 {
		t.Fatalf(
			"non-constant array range did not capture its callable once:\n%s",
			rangeFunction,
		)
	}
	guardedCall := regexp.MustCompile(
		`\(` + regexp.QuoteMeta(callee[1]) +
			` \?\? GoPanic\.raiseRuntime\("call of nil function"\)\)\(\)`,
	)
	if calls := guardedCall.FindAllString(rangeFunction, -1); len(calls) != 1 {
		t.Fatalf(
			"non-constant array range evaluates source %d times, want once:\n%s",
			len(calls),
			rangeFunction,
		)
	}
	switchFunction := targetFunctionText(t, printed, "switchArray")
	bodies := regexp.MustCompile(`return 12n?;`).
		FindAllString(switchFunction, -1)
	if len(bodies) != 1 {
		t.Fatalf(
			"conditional switch body appears %d times, want once:\n%s",
			len(bodies),
			switchFunction,
		)
	}
}

func targetFunctionText(t *testing.T, printed, name string) string {
	t.Helper()
	startMarker := "export function " + name + "("
	start := strings.Index(printed, startMarker)
	if start < 0 {
		t.Fatalf("Wave 4 artifacts lack function %s", name)
	}
	remainder := printed[start+len(startMarker):]
	end := strings.Index(remainder, "\nexport function ")
	artifactEnd := strings.Index(remainder, "\n\n// ")
	if end < 0 || artifactEnd >= 0 && artifactEnd < end {
		end = artifactEnd
	}
	if end < 0 {
		return printed[start:]
	}
	return printed[start : start+len(startMarker)+end]
}

func executeWaveFourGo(t *testing.T, workingDirectory string) string {
	t.Helper()
	modulePath, err := filepath.Abs(waveFourStatementDirectory())
	if err != nil {
		t.Fatal(err)
	}
	runnerDirectory := filepath.Join(workingDirectory, "go-runner-wave4")
	writeProgramFile(t, filepath.Join(runnerDirectory, "go.mod"), fmt.Sprintf(
		`module example.com/runner

go 1.26.4

require example.com/wave4statements v0.0.0

replace example.com/wave4statements => %s
`,
		filepath.ToSlash(modulePath),
	))
	writeProgramFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"

	values "example.com/wave4statements"
)

func main() {
	for index, value := range values.Audit() {
		if index != 0 {
			fmt.Print(" ")
		}
		fmt.Print(value)
	}
	fmt.Println()
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

func waveFourStatementDirectory() string {
	return filepath.Join(
		repositoryRoot(),
		"testdata",
		"constructs",
		"statement",
		"wave4",
	)
}

func TestBlankIdentifierDispositionsPrintTypecheckAndExecuteDifferentially(
	t *testing.T,
) {
	projectDirectory := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(projectDirectory, "go.mod"),
		"module example.com/blankidentifier\n\ngo 1.26.4\n",
	)
	writeProgramFile(
		t,
		filepath.Join(projectDirectory, "source.go"),
		`package blankidentifier

const (
	_ int32 = iota
	one
)

const _ = 1 << 10
type _ int32
func _() {}

type Item struct{}

func (Item) _() {}
func (_ Item) Value() int32 { return 3 }
func (_ Item) Speak(_ int32) int32 { return 4 }

type Speaker interface {
	Speak(_ int32) int32
}

func zero() (_ int32) {
	defer func() {}()
	return
}
func ignore(_ int32, _ int32) int32 { return 5 }
func generic[_ any](value int32) int32 { return value }

func Run() int32 {
	const _ = 5
	type _ int32
	literal := func(_ int32) int32 { return 6 }
	var speaker Speaker = Item{}
	return one +
		Item{}.Value() +
		speaker.Speak(0) +
		zero() +
		ignore(0, 0) +
		generic[string](7) +
		literal(0)
}
`,
	)
	program, err := load.Load(context.Background(), load.Request{
		Directory: projectDirectory,
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	rootObject := program.Roots()[0].Types().Scope().Lookup("Run")
	root, err := emit.NewRoot(rootObject)
	if err != nil {
		t.Fatal(err)
	}
	emission, err := emit.Compile(program, []emit.Root{root})
	if err != nil {
		t.Fatal(err)
	}

	workingDirectory := t.TempDir()
	artifacts := materializeArtifacts(t, emission, workingDirectory)
	blankDefinition := regexp.MustCompile(
		`\b(?:function|const|let|class|interface|type)\s+_\b`,
	)
	if blankDefinition.MatchString(artifacts.printed) {
		t.Fatalf("blank source declaration leaked into TypeScript:\n%s", artifacts.printed)
	}
	for _, expected := range []string{"$0", "$1", "$T0", "$result0"} {
		if !strings.Contains(artifacts.printed, expected) {
			t.Fatalf(
				"target-only blank slot %q is absent:\n%s",
				expected,
				artifacts.printed,
			)
		}
	}

	writeProgramFile(
		t,
		filepath.Join(workingDirectory, "package.json"),
		"{\"type\":\"module\"}\n",
	)
	runnerPath := filepath.Join(workingDirectory, "runner.ts")
	writeProgramFile(
		t,
		runnerPath,
		`import "./program.js";
import { Run } from "`+artifacts.sourceModule+`";

console.log(Run());
`,
	)
	outputDirectory := filepath.Join(workingDirectory, "out")
	arguments := []string{
		"--target", "es2022",
		"--module", "nodenext",
		"--moduleResolution", "nodenext",
		"--strict",
		"--outDir", outputDirectory,
	}
	arguments = append(arguments, artifacts.paths...)
	arguments = append(arguments, runnerPath)
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
	targetOutput := runProgram(
		t,
		workingDirectory,
		"node",
		filepath.Join(outputDirectory, "runner.js"),
	)
	if targetOutput != "26\n" {
		t.Fatalf("TypeScript output = %q, want Go-equivalent 26", targetOutput)
	}

	goTest := filepath.Join(projectDirectory, "source_test.go")
	writeProgramFile(
		t,
		goTest,
		`package blankidentifier

import "testing"

func TestRun(t *testing.T) {
	if got := Run(); got != 26 {
		t.Fatalf("Run() = %d, want 26", got)
	}
}
`,
	)
	commandContext, commandCancel := context.WithTimeout(
		context.Background(),
		2*time.Minute,
	)
	defer commandCancel()
	command := exec.CommandContext(
		commandContext,
		filepath.Join(runtime.GOROOT(), "bin", "go"),
		"test",
		".",
	)
	command.Dir = projectDirectory
	command.Env = append(os.Environ(), "GOMEMLIMIT=1GiB")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("Go differential: %v\n%s", err, output)
	}
}
