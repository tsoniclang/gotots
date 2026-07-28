package maprepresentation_test

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/output"
)

func TestAggregateMapsCompileAndExecuteDifferentially(t *testing.T) {
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
			loaded := loadAggregateMapProject(t)
			roots, err := emit.ExportedAPIRoots(loaded)
			if err != nil {
				t.Fatal(err)
			}
			emission, err := emit.CompileWithOptions(
				loaded.Program(),
				roots,
				testCase.options,
			)
			if err != nil {
				t.Fatal(err)
			}
			workingDirectory := t.TempDir()
			artifacts := materialize(t, emission, workingDirectory)
			assertAggregateMapArtifacts(t, emission, artifacts)
			goOutput := executeAggregateMapGo(t, workingDirectory)
			targetOutput := executeAggregateMapTypeScript(
				t,
				artifacts,
				workingDirectory,
			)
			if targetOutput != goOutput {
				t.Fatalf(
					"TypeScript output differs from Go\nTypeScript:\n%s\nGo:\n%s",
					targetOutput,
					goOutput,
				)
			}
		})
	}
}

func TestAggregateMapArtifactsIgnoreRootOrderAndUnreachableShapes(t *testing.T) {
	loaded := loadAggregateMapProject(t)
	roots, err := emit.ExportedAPIRoots(loaded)
	if err != nil {
		t.Fatal(err)
	}
	first, err := emit.CompileWithOptions(
		loaded.Program(),
		roots,
		emit.DefaultOptions(),
	)
	if err != nil {
		t.Fatal(err)
	}
	reversed := slices.Clone(roots)
	slices.Reverse(reversed)
	second, err := emit.CompileWithOptions(
		loaded.Program(),
		reversed,
		emit.DefaultOptions(),
	)
	if err != nil {
		t.Fatal(err)
	}
	assertSameArtifacts(
		t,
		materializedContents(t, first),
		materializedContents(t, second),
		"root order",
	)

	reachedDirectory := t.TempDir()
	for _, name := range []string{"go.mod", "source.go"} {
		writeFile(
			t,
			filepath.Join(reachedDirectory, name),
			readFile(t, filepath.Join(aggregateMapProjectDirectory(), name)),
		)
	}
	reached, err := load.One(context.Background(), load.Request{
		Directory: reachedDirectory,
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	reachedRoots, err := emit.ExportedAPIRoots(reached)
	if err != nil {
		t.Fatal(err)
	}
	withoutUnreachable, err := emit.CompileWithOptions(
		reached.Program(),
		reachedRoots,
		emit.DefaultOptions(),
	)
	if err != nil {
		t.Fatal(err)
	}
	assertSameArtifacts(
		t,
		materializedContents(t, first),
		materializedContents(t, withoutUnreachable),
		"unreachable map shape",
	)
}

func assertAggregateMapArtifacts(
	t *testing.T,
	emission emit.ProgramEmission,
	artifacts materialized,
) {
	t.Helper()
	generated := 0
	totalBytes := 0
	largestBytes := 0
	largestPath := ""
	for _, file := range emission.Files() {
		if !strings.HasPrefix(file.OutputPath(), "support/maps/") {
			continue
		}
		generated++
		if file.Kind() != emit.TargetFileSupport {
			t.Fatalf(
				"map specialization %s has target-file kind %d",
				file.OutputPath(),
				file.Kind(),
			)
		}
		source := readFile(t, artifacts.file(t, file.OutputPath()))
		if strings.Count(source, "export class $goMap_") != 1 {
			t.Fatalf(
				"map specialization %s does not own exactly one class:\n%s",
				file.OutputPath(),
				source,
			)
		}
		totalBytes += len(source)
		if len(source) > largestBytes {
			largestBytes = len(source)
			largestPath = file.OutputPath()
		}
		if len(source) > 7_000 {
			t.Fatalf(
				"map specialization %s = %d bytes, want at most 7000",
				file.OutputPath(),
				len(source),
			)
		}
		for _, forbidden := range []string{
			": any",
			": unknown",
			".call(",
			".apply(",
			".bind(",
			"private readonly hash",
			"private readonly equal",
			"private readonly copyKey",
		} {
			if strings.Contains(source, forbidden) {
				t.Fatalf(
					"map specialization %s contains %q:\n%s",
					file.OutputPath(),
					forbidden,
					source,
				)
			}
		}
	}
	if generated != 5 {
		t.Fatalf(
			"map specialization artifacts = %d, want five exact reached shapes",
			generated,
		)
	}
	anonymous := readFile(
		t,
		artifacts.file(t, output.AnonymousStructSupportPath),
	)
	if strings.Count(anonymous, "export class $goStruct_") != 2 ||
		strings.Count(anonymous, "static $hash(") != 1 {
		t.Fatalf(
			"anonymous map dependency owns the wrong classes/hash surface:\n%s",
			anonymous,
		)
	}
	t.Logf(
		"map specializations=%d total-bytes=%d largest=%s/%d",
		generated,
		totalBytes,
		largestPath,
		largestBytes,
	)
}

func executeAggregateMapGo(t *testing.T, workingDirectory string) string {
	t.Helper()
	modulePath, err := filepath.Abs(aggregateMapProjectDirectory())
	if err != nil {
		t.Fatal(err)
	}
	runnerDirectory := filepath.Join(workingDirectory, "go-runner")
	writeFile(t, filepath.Join(runnerDirectory, "go.mod"), fmt.Sprintf(`module example.com/runner

go 1.26.4

require example.com/aggregatemap v0.0.0

replace example.com/aggregatemap => %s
`, filepath.ToSlash(modulePath)))
	writeFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"

	values "example.com/aggregatemap"
)

func main() {
	fmt.Println(values.NamedValueLifecycle())
	fmt.Println(values.ArrayKeyLifecycle())
	fmt.Println(values.StructKeyLifecycle())
	fmt.Println(values.AnonymousShapeLifecycle())
	fmt.Println(values.CollisionEquality())
	fmt.Println(values.LiteralOrder())
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

func executeAggregateMapTypeScript(
	t *testing.T,
	artifacts materialized,
	workingDirectory string,
) string {
	t.Helper()
	runnerPath := filepath.Join(workingDirectory, "runner.ts")
	writeFile(t, runnerPath, `import {
    AnonymousShapeLifecycle,
    ArrayKeyLifecycle,
    CollisionEquality,
    LiteralOrder,
    NamedValueLifecycle,
    StructKeyLifecycle,
} from "`+artifacts.module(t, "source.ts")+`";
import "./program.js";

function show(value: number | bigint | boolean): string {
    return typeof value === "bigint" ? value.toString() : String(value);
}
function print(...values: Array<number | bigint | boolean>): void {
    console.log(...values.map(show));
}

print(...NamedValueLifecycle());
print(...ArrayKeyLifecycle());
print(...StructKeyLifecycle());
print(...AnonymousShapeLifecycle());
print(...CollisionEquality());
print(LiteralOrder());
`)
	writeFile(
		t,
		filepath.Join(workingDirectory, "package.json"),
		"{\"type\":\"module\"}\n",
	)
	strictTypecheckWithRunner(t, artifacts, workingDirectory, runnerPath)
	return run(
		t,
		workingDirectory,
		"node",
		filepath.Join(workingDirectory, "out", "runner.js"),
	)
}

func loadAggregateMapProject(t *testing.T) *load.Package {
	t.Helper()
	loaded, err := load.One(context.Background(), load.Request{
		Directory: aggregateMapProjectDirectory(),
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	return loaded
}

func aggregateMapProjectDirectory() string {
	return filepath.Join(
		repositoryRoot(),
		"testdata",
		"constructs",
		"value",
		"map",
		"aggregate",
	)
}

func materializedContents(
	t *testing.T,
	emission emit.ProgramEmission,
) map[string]string {
	t.Helper()
	root := t.TempDir()
	artifacts := materialize(t, emission, root)
	result := make(map[string]string, len(artifacts.paths))
	for _, targetPath := range artifacts.paths {
		relative, err := filepath.Rel(root, targetPath)
		if err != nil {
			t.Fatal(err)
		}
		result[filepath.ToSlash(relative)] = readFile(t, targetPath)
	}
	return result
}

func assertSameArtifacts(
	t *testing.T,
	left map[string]string,
	right map[string]string,
	reason string,
) {
	t.Helper()
	if len(left) != len(right) {
		t.Fatalf(
			"%s changed artifact count: %d != %d",
			reason,
			len(left),
			len(right),
		)
	}
	for targetPath, content := range left {
		if right[targetPath] != content {
			t.Fatalf("%s changed target artifact %s", reason, targetPath)
		}
	}
}
