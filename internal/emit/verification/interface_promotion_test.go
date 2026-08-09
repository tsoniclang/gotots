package emit_test

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestWaveSixInterfacesCompileThroughThePublicPipeline(t *testing.T) {
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
				Directory: waveSixInterfaceDirectory(),
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
			if artifacts.bytes > 110_000 || artifacts.largest > 40_000 {
				t.Fatalf(
					"Wave 6 artifact bounds exceeded: total=%d largest=%d",
					artifacts.bytes,
					artifacts.largest,
				)
			}
			assertWaveSixShape(t, artifacts.printed)
			waveThreeTypecheck(t, workingDirectory, artifacts.paths)
			t.Logf(
				"Wave 6 artifacts total=%d largest=%d",
				artifacts.bytes,
				artifacts.largest,
			)
			for index, artifact := range artifacts.sizes {
				if index == 20 {
					break
				}
				t.Logf(
					"Wave 6 artifact rank=%d path=%s bytes=%d",
					index+1,
					artifact.path,
					artifact.bytes,
				)
			}
		})
	}
}

func assertWaveSixShape(t *testing.T, printed string) {
	t.Helper()
	for _, required := range []string{
		"abstract class GoInterfaceValue",
		"readonly $go$type",
		"readonly $go$methods",
		"$go$implements(",
		"$go$equal(",
		"$go$hash(",
		"export interface DerivedReader",
		"readonly $go$value",
		"Object.freeze(",
		"switch (true)",
		".$is(",
		"$go$type === $goDynamicType_",
	} {
		if !strings.Contains(printed, required) {
			t.Fatalf("Wave 6 artifacts lack %q:\n%s", required, printed)
		}
	}
	if strings.Contains(printed, "instanceof $goInterfaceAdapter_") {
		t.Fatalf(
			"Wave 6 artifacts use constructor identity for Go dynamic types:\n%s",
			printed,
		)
	}
	if got := strings.Count(printed, "    static format"); got != 5 {
		t.Fatalf(
			"Wave 6 format truth is not shared exactly once per value class: %d",
			got,
		)
	}
	if got := strings.Count(printed, "if (verb === \"T\")"); got != 6 {
		t.Fatalf("Wave 6 contains duplicated format decision bodies: %d", got)
	}
	for _, shared := range []string{
		"GoInterfaceFormat.formatOther",
		"GoInterfaceFormat.formatBoolean",
		"GoInterfaceFormat.formatString",
		"GoInterfaceFormat.formatInteger",
		"GoInterfaceFormat.formatFloat",
	} {
		if !strings.Contains(printed, shared) {
			t.Fatalf("Wave 6 artifacts do not consume shared format owner %q", shared)
		}
	}
}

func executeWaveSixGo(t *testing.T, workingDirectory string) string {
	t.Helper()
	modulePath, err := filepath.Abs(waveSixInterfaceDirectory())
	if err != nil {
		t.Fatal(err)
	}
	runnerDirectory := filepath.Join(workingDirectory, "go-runner-wave6")
	writeProgramFile(t, filepath.Join(runnerDirectory, "go.mod"), fmt.Sprintf(
		`module example.com/runner

go 1.26.4

require example.com/wave6interfaces v0.0.0

replace example.com/wave6interfaces => %s
`,
		filepath.ToSlash(modulePath),
	))
	writeProgramFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"

	values "example.com/wave6interfaces"
)

func panics(action func()) (result bool) {
	defer func() {
		result = recover() != nil
	}()
	action()
	return false
}

func main() {
	for index, value := range values.Audit() {
		if index != 0 {
			fmt.Print(" ")
		}
		fmt.Print(value)
	}
	for _, action := range []func(){
		values.FailedAssertion,
		values.UncomparableEquality,
		values.UnhashableMapKey,
	} {
		if panics(action) {
			fmt.Print(" panic")
		} else {
			fmt.Print(" no-panic")
		}
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

func waveSixInterfaceDirectory() string {
	return filepath.Join(
		repositoryRoot(),
		"testdata",
		"constructs",
		"interface",
		"wave6",
	)
}

type waveSixScale struct {
	implementers int
	adapters     int
	callBytes    int
	printedBytes int
	wireBytes    int
}

func TestWaveSixInterfaceDispatchIsIndependentOfImplementerCount(
	t *testing.T,
) {
	client, err := tsgo.StartClient(repositoryRoot(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("close TS-Go client: %v", err)
		}
	})

	var measurements []waveSixScale
	for _, count := range []int{4, 8, 16} {
		measurement := measureWaveSixScale(t, client, count)
		t.Logf(
			"implementers=%d adapters=%d call=%dB printed=%dB wire=%dB",
			measurement.implementers,
			measurement.adapters,
			measurement.callBytes,
			measurement.printedBytes,
			measurement.wireBytes,
		)
		if measurement.adapters != count {
			t.Fatalf(
				"%d implementers emitted %d adapters",
				count,
				measurement.adapters,
			)
		}
		measurements = append(measurements, measurement)
	}
	smallestCall := measurements[0].callBytes
	largestCall := measurements[0].callBytes
	for _, measurement := range measurements[1:] {
		smallestCall = min(smallestCall, measurement.callBytes)
		largestCall = max(largestCall, measurement.callBytes)
	}
	if largestCall-smallestCall > 4 {
		t.Fatalf(
			"interface call size depends on implementers: %v",
			measurements,
		)
	}
	assertWaveSixLinearGrowth(
		t,
		"printed TypeScript",
		measurements,
		func(value waveSixScale) int { return value.printedBytes },
	)
	assertWaveSixLinearGrowth(
		t,
		"encoded TS-Go AST",
		measurements,
		func(value waveSixScale) int { return value.wireBytes },
	)
}

func measureWaveSixScale(
	t *testing.T,
	client *tsgo.Client,
	implementers int,
) waveSixScale {
	t.Helper()
	projectDirectory := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(projectDirectory, "go.mod"),
		"module example.com/wave6scale\n\ngo 1.26.4\n",
	)
	var source strings.Builder
	source.WriteString("package wave6scale\n\n")
	source.WriteString("type Reader interface { Read() int32 }\n\n")
	for index := range implementers {
		fmt.Fprintf(&source, "type V%d struct{}\n", index)
		fmt.Fprintf(
			&source,
			"func (value V%d) Read() int32 { return %d }\n",
			index,
			index,
		)
	}
	source.WriteString(
		"\nfunc Call(value Reader) int32 { return value.Read() }\n",
	)
	source.WriteString("func Audit() int32 { return ")
	for index := range implementers {
		if index != 0 {
			source.WriteString(" + ")
		}
		fmt.Fprintf(&source, "Call(V%d{})", index)
	}
	source.WriteString(" }\n")
	writeProgramFile(
		t,
		filepath.Join(projectDirectory, "source.go"),
		source.String(),
	)
	program, err := load.Load(context.Background(), load.Request{
		Directory: projectDirectory,
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	roots, err := emit.ExportedAPIRoots(program.Roots()[0])
	if err != nil {
		t.Fatal(err)
	}
	emission, err := emit.Compile(program, roots)
	if err != nil {
		t.Fatal(err)
	}
	result := waveSixScale{implementers: implementers}
	var printed strings.Builder
	for _, file := range emission.Files() {
		wire, err := tsgo.EncodeSourceFile(file.SourceFile())
		if err != nil {
			t.Fatal(err)
		}
		result.wireBytes += len(wire)
		target, err := client.PrintNode(file.SourceFile(), tsgo.PrintOptions{})
		if err != nil {
			t.Fatal(err)
		}
		result.printedBytes += len(target)
		printed.WriteString(target)
		printed.WriteByte('\n')
	}
	target := printed.String()
	result.adapters = strings.Count(
		target,
		"export class $goInterfaceAdapter_",
	)
	call := targetFunctionText(t, target, "Call")
	result.callBytes = len(call)
	if strings.Contains(call, "switch") ||
		strings.Contains(call, "$is(") ||
		!strings.Contains(call, ".Read(") {
		t.Fatalf("interface call is not direct O(1) dispatch:\n%s", call)
	}
	return result
}

func assertWaveSixLinearGrowth(
	t *testing.T,
	name string,
	values []waveSixScale,
	measure func(waveSixScale) int,
) {
	t.Helper()
	first := measure(values[1]) - measure(values[0])
	second := measure(values[2]) - measure(values[1])
	if first <= 0 || second <= 0 || second > first*23/10 {
		t.Fatalf(
			"%s growth is not linear: deltas %d, %d",
			name,
			first,
			second,
		)
	}
}

func TestPromotedPointerInterfaceAdapterPreservesReceiverAddress(
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
				Directory: promotedPointerInterfaceDirectory(),
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
			if !strings.Contains(
				artifacts.printed,
				"$goInterfaceAdapter_",
			) || !strings.Contains(
				artifacts.printed,
				"Inner__from_promotedpointer.Increment(",
			) || !strings.Contains(
				artifacts.printed,
				"loadPointer<Outer__from_promotedpointer>",
			) || !strings.Contains(
				artifacts.printed,
				"addressOf<Inner__from_promotedpointer>(__gotots_store_0.Inner)",
			) {
				t.Fatalf(
					"promoted pointer adapter lacks direct typed receiver projection:\n%s",
					artifacts.printed,
				)
			}
			if strings.Contains(artifacts.printed, "GoPointer") ||
				strings.Contains(artifacts.printed, "runtime/pointer") {
				t.Fatalf(
					"promoted stable receiver retained a carrier projection:\n%s",
					artifacts.printed,
				)
			}
			writeProgramFile(
				t,
				filepath.Join(workingDirectory, "package.json"),
				"{\"type\":\"module\"}\n",
			)
			waveThreeTypecheck(
				t,
				workingDirectory,
				artifacts.paths,
			)
			if artifacts.bytes > 48_000 || artifacts.largest > 24_000 {
				t.Fatalf(
					"promoted pointer artifacts exceed bounds: total=%d largest=%d",
					artifacts.bytes,
					artifacts.largest,
				)
			}
		})
	}
}

func executePromotedPointerGo(
	t *testing.T,
	workingDirectory string,
) string {
	t.Helper()
	modulePath, err := filepath.Abs(promotedPointerInterfaceDirectory())
	if err != nil {
		t.Fatal(err)
	}
	runnerDirectory := filepath.Join(
		workingDirectory,
		"go-runner-promoted-pointer",
	)
	writeProgramFile(
		t,
		filepath.Join(runnerDirectory, "go.mod"),
		fmt.Sprintf(
			`module example.com/runner

go 1.26.4

require example.com/promotedpointer v0.0.0

replace example.com/promotedpointer => %s
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

	values "example.com/promotedpointer"
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

func promotedPointerInterfaceDirectory() string {
	return filepath.Join(
		repositoryRoot(),
		"testdata",
		"constructs",
		"interface",
		"promoted-pointer",
	)
}
