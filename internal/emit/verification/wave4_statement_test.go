package emit_test

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"testing"

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
	paths        []string
	sourceModule string
	bytes        int
	largest      int
	sizes        []artifactSize
	printed      string
}

type artifactSize struct {
	path  string
	bytes int
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
	result := waveFourArtifacts{}
	for _, file := range emission.Files() {
		printed, err := client.PrintNode(file.SourceFile(), tsgo.PrintOptions{})
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
		result.printed += "\n// " + file.OutputPath() + "\n" + printed
		result.sizes = append(result.sizes, artifactSize{
			path:  file.OutputPath(),
			bytes: len(printed),
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
		"outer:",
		"continue outer;",
	} {
		if !strings.Contains(printed, required) {
			t.Fatalf("Wave 4 artifacts lack %q:\n%s", required, printed)
		}
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
	calls := regexp.MustCompile(`__gotots_callee_[0-9]+\(\)`).
		FindAllString(rangeFunction, -1)
	if len(calls) != 1 {
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
