package slice_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
	runtimeemission "github.com/tsoniclang/gotots/internal/emit/runtime"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
	corefixture "github.com/tsoniclang/gotots/internal/testfixture/tsoniccore"
)

func TestRuntimeSliceProjectionPreservesBidirectionalAlias(t *testing.T) {
	abi, err := api.NewScalarABI(
		api.IntegerRepresentationNumber,
		api.NativeIntegerWidth64,
	)
	if err != nil {
		t.Fatal(err)
	}
	assembled, err := runtimeemission.AssemblePackage(
		tsgo.NewFactory(),
		abi,
		map[api.RuntimeSymbol]struct{}{
			api.RuntimeSliceProjection:   {},
			api.RuntimeSliceAddress:      {},
			api.RuntimeSliceArrayPointer: {},
		},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	workingDirectory := t.TempDir()
	client, err := tsgo.StartClient(repositoryRoot(), workingDirectory)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if closeErr := client.Close(); closeErr != nil {
			t.Errorf("close TS-Go client: %v", closeErr)
		}
	})
	var targetPaths []string
	var sliceSource string
	for _, file := range assembled.Files() {
		source, printErr := client.PrintNode(
			file.SourceFile(),
			tsgo.PrintOptions{},
		)
		if printErr != nil {
			t.Fatal(printErr)
		}
		targetPath := filepath.Join(
			workingDirectory,
			filepath.FromSlash(file.OutputPath()),
		)
		writeFile(t, targetPath, source)
		targetPaths = append(targetPaths, targetPath)
		if file.OutputPath() == "runtime/slice.ts" {
			sliceSource = source
		}
	}
	for _, fragment := range []string{
		"export class RuntimeSliceProjection<F, T> extends RuntimeSlice<T>",
		"override get(index: number | bigint): T",
		"this.source.set(index, this.toSource(value))",
		"this.source.slice(low, high, max)",
		"this.source.append(this.sourceZero, converted)",
		"$projectedAddress<U>",
		"projectPointer<T, U>",
		"this.source.$projectedAddress<T>",
		"projectPointer<T | undefined, GoArray<T, N>>",
		"projected slice has no contiguous target representation",
	} {
		if !strings.Contains(sliceSource, fragment) {
			t.Fatalf("projected slice runtime lacks %q:\n%s", fragment, sliceSource)
		}
	}
	if strings.Contains(
		sliceSource,
		"projectPointer<F, T>(this.source.address(index)",
	) {
		t.Fatal("projected slice address retained the superseded two-location path")
	}
	runnerPath := filepath.Join(workingDirectory, "runner.ts")
	writeFile(t, runnerPath, `import {
  RuntimeSlice,
  RuntimeSliceProjection,
} from "./runtime/slice.js";
const source = RuntimeSlice.make<bigint>(2, 4, 0n);
source.set(0, 1n);
source.set(1, 2n);
const target = new RuntimeSliceProjection<bigint, number>(
  source,
  (value) => Number(value),
  (value) => BigInt(value),
  0n,
  0,
);
const alias = target.slice(1, null, null);
target.set(1, 7);
console.log(source.get(1), alias.get(0));
source.set(1, 9n);
console.log(target.get(1), alias.get(0));

const reused = target.append(0, [5]);
console.log(reused.length, source.slice(0, 3, null).get(2));
const grown = target.append(0, [5, 6, 7]);
grown.set(0, 11);
console.log(grown.length, grown.capacity, grown.get(4), source.get(0));

const copied = RuntimeSlice.make<number>(2, null, 0);
console.log(RuntimeSlice.copy(copied, target), copied.get(0), copied.get(1));
copied.set(0, 13);
console.log(RuntimeSlice.copy(target, copied), source.get(0), source.get(1));

const nilSource = RuntimeSlice.nil<bigint>();
const nilTarget = new RuntimeSliceProjection<bigint, number>(
  nilSource,
  Number,
  BigInt,
  0n,
  0,
);
console.log(nilTarget.isNil(), nilTarget.length, nilTarget.capacity);
try {
  target.$arrayLocation(1);
} catch {
  console.log("projected-contiguous-unsupported");
}
`)
	targetPaths = append(targetPaths, runnerPath)
	if err := corefixture.InstallResolutionOnly(workingDirectory); err != nil {
		t.Fatal(err)
	}
	writeFile(
		t,
		filepath.Join(workingDirectory, "package.json"),
		"{\"type\":\"module\"}\n",
	)
	typecheck(t, workingDirectory, targetPaths)
	output := run(
		t,
		workingDirectory,
		"node",
		filepath.Join(workingDirectory, "out", "runner.js"),
	)
	const expected = "7n 7\n9 9\n3 5n\n5 8 7 1n\n2 1 9\n2 13n 9n\ntrue 0 0\nprojected-contiguous-unsupported\n"
	if output != expected {
		t.Fatalf("projected slice output = %q, want %q", output, expected)
	}
}
