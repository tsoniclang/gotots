package unsafeintrinsic_test

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
)

func TestUnsafeRegionIntrinsicsPrintTypecheckAndMatchGo(t *testing.T) {
	for _, testCase := range []struct {
		name       string
		options    emit.Options
		byteValues string
		one        string
		nonzero    string
		negative   string
	}{
		{
			name:       "number",
			options:    emit.DefaultOptions(),
			byteValues: "255, 65",
			one:        "1",
			nonzero:    "1",
			negative:   "-1",
		},
		{
			name: "bigint",
			options: emit.Options{
				IntegerRepresentation: emit.IntegerRepresentationBigInt,
				EvaluationOrder:       emit.EvaluationOrderPreserveGo,
			},
			byteValues: "255, 65",
			one:        "1",
			nonzero:    "1n",
			negative:   "-1n",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			project := writeUnsafeIntrinsicProject(t)
			program, err := load.Load(context.Background(), load.Request{
				Directory: project,
				Pattern:   ".",
			})
			if err != nil {
				t.Fatal(err)
			}
			scope := program.Roots()[0].Types().Scope()
			rootNames := []string{
				"BuildString",
				"AliasSlice",
				"StringDataFirst",
				"SliceDataAlias",
				"EmptyString",
				"NilNonzero",
				"NegativeLength",
				"NilComparisons",
			}
			roots := make([]emit.Root, 0, len(rootNames))
			for _, name := range rootNames {
				root, rootErr := emit.NewRoot(scope.Lookup(name))
				if rootErr != nil {
					t.Fatal(rootErr)
				}
				roots = append(roots, root)
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
			artifacts := materializeArtifacts(t, emission, workingDirectory)
			assertUnsafeRuntimeShape(t, artifacts.printed)

			runner := filepath.Join(workingDirectory, "runner.ts")
			writeProgramFile(t, runner, unsafeTargetRunner(
				artifacts.sourceModule,
				testCase.byteValues,
				testCase.one,
				testCase.nonzero,
				testCase.negative,
			))
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
			goOutput := executeUnsafeIntrinsicGo(t, project, workingDirectory)
			if targetOutput != goOutput {
				t.Fatalf(
					"unsafe intrinsic output differs\nTypeScript:\n%s\nGo:\n%s",
					targetOutput,
					goOutput,
				)
			}
		})
	}
}

func writeUnsafeIntrinsicProject(t *testing.T) string {
	t.Helper()
	project := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(project, "go.mod"),
		"module example.com/unsaferegion\n\ngo 1.26.4\n",
	)
	writeProgramFile(t, filepath.Join(project, "source.go"), `package unsaferegion

import "unsafe"

func BuildString(bytes []byte) string {
	return unsafe.String(&bytes[0], len(bytes))
}

func AliasSlice(bytes []byte) byte {
	view := unsafe.Slice(&bytes[0], len(bytes))
	view[1] = 90
	return bytes[1]
}

func StringDataFirst() byte {
	value := string([]byte{0xff, 65})
	return *unsafe.StringData(value)
}

func SliceDataAlias(bytes []byte) byte {
	pointer := unsafe.SliceData(bytes)
	*pointer = 88
	return bytes[0]
}

func EmptyString() string {
	return unsafe.String(nil, 0)
}

func NilNonzero(length int) string {
	return unsafe.String(nil, length)
}

func NegativeLength(bytes []byte, length int) []byte {
	return unsafe.Slice(&bytes[0], length)
}

func NilComparisons() (bool, bool) {
	var zero unsafe.Pointer
	return zero == nil, nil != zero
}
`)
	return project
}

func assertUnsafeRuntimeShape(t *testing.T, printed string) {
	t.Helper()
	for _, required := range []string{
		"export function goUnsafeString<",
		"export function goUnsafeSlice<",
		"export function goUnsafeStringData<",
		"export function goUnsafeSliceData<",
		"private readonly $go$region",
		"globalThis.Number",
	} {
		if !strings.Contains(printed, required) {
			t.Fatalf("unsafe runtime lacks %q:\n%s", required, printed)
		}
	}
	for _, forbidden := range []string{
		"export declare function String(",
		"export declare function Slice(",
		"export declare function StringData(",
		"export declare function SliceData(",
	} {
		if strings.Contains(printed, forbidden) {
			t.Fatalf("unsafe intrinsic leaked ambient declaration %q", forbidden)
		}
	}
}

func unsafeTargetRunner(
	sourceModule string,
	byteValues string,
	one string,
	nonzero string,
	negative string,
) string {
	return `import { RuntimeSlice } from "./runtime/slice.js";
import {
    AliasSlice,
    BuildString,
    EmptyString,
    NegativeLength,
    NilComparisons,
    SliceDataAlias,
    StringDataFirst,
	NilNonzero,
} from "` + sourceModule + `";

function bytes(value: string): string {
    const result: string[] = [];
    for (let index = 0; index < value.length; index++) {
        result.push(value.charCodeAt(index).toString(16).padStart(2, "0"));
    }
    return result.join("");
}

let nilNonzero = false;
try {
    NilNonzero(` + nonzero + `);
} catch {
    nilNonzero = true;
}
let negativeLength = false;
try {
    NegativeLength(RuntimeSlice.literal([` + byteValues + `]), ` + negative + `);
} catch {
    negativeLength = true;
}
const nilComparisons = NilComparisons();
console.log([
    bytes(BuildString(RuntimeSlice.literal([` + byteValues + `]))),
    String(AliasSlice(RuntimeSlice.literal([` + one + `, ` + one + `]))),
    String(StringDataFirst()),
    String(SliceDataAlias(RuntimeSlice.literal([` + one + `, ` + one + `]))),
    JSON.stringify(EmptyString()),
	String(nilNonzero),
    String(negativeLength),
    String(nilComparisons[0]),
    String(nilComparisons[1]),
].join("|"));
`
}

func executeUnsafeIntrinsicGo(
	t *testing.T,
	project string,
	workingDirectory string,
) string {
	t.Helper()
	runnerDirectory := filepath.Join(workingDirectory, "go-runner-unsafe")
	writeProgramFile(t, filepath.Join(runnerDirectory, "go.mod"), fmt.Sprintf(
		`module example.com/runner

go 1.26.4

require example.com/unsaferegion v0.0.0

replace example.com/unsaferegion => %s
`,
		filepath.ToSlash(project),
	))
	writeProgramFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"

	values "example.com/unsaferegion"
)

func panics(call func()) (result bool) {
	defer func() { result = recover() != nil }()
	call()
	return false
}

func main() {
	text := values.BuildString([]byte{0xff, 65})
	alias := values.AliasSlice([]byte{1, 1})
	stringData := values.StringDataFirst()
	sliceData := values.SliceDataAlias([]byte{1, 1})
	nilNonzero := panics(func() { values.NilNonzero(1) })
	negative := panics(func() { values.NegativeLength([]byte{0xff, 65}, -1) })
	nilZero, nilNonzeroPointer := values.NilComparisons()
	fmt.Printf("%x|%d|%d|%d|%q|%t|%t|%t|%t\n",
		text, alias, stringData, sliceData, values.EmptyString(), nilNonzero, negative,
		nilZero, nilNonzeroPointer)
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
