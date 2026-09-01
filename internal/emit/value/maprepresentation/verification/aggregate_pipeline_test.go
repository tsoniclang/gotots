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
		{name: "number", options: mapNumberOptions()},
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
		mapNumberOptions(),
	)
	if err != nil {
		t.Fatal(err)
	}
	reversed := slices.Clone(roots)
	slices.Reverse(reversed)
	second, err := emit.CompileWithOptions(
		loaded.Program(),
		reversed,
		mapNumberOptions(),
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
		mapNumberOptions(),
	)
	if err != nil {
		t.Fatal(err)
	}
	assertSameArtifacts(
		t,
		withoutSourceFactApplicationsByPath(materializedContents(t, first)),
		withoutSourceFactApplicationsByPath(materializedContents(t, withoutUnreachable)),
		"unreachable map shape",
	)
}

func assertAggregateMapArtifacts(
	t *testing.T,
	emission emit.ProgramEmission,
	artifacts materialized,
) {
	t.Helper()
	generatedFiles := 0
	generatedDefinitions := 0
	nativeDefinitions := 0
	hashedDefinitions := 0
	hashedBucketLoops := 0
	totalBytes := 0
	largestBytes := 0
	largestPath := ""
	for _, file := range emission.Files() {
		if file.OutputPath() != output.MapSpecializationSupportPath {
			continue
		}
		generatedFiles++
		if file.Kind() != emit.TargetFileSupport {
			t.Fatalf(
				"map specialization %s has target-file kind %d",
				file.OutputPath(),
				file.Kind(),
			)
		}
		source := readFile(t, artifacts.file(t, file.OutputPath()))
		definitions := strings.Count(source, "export class $goMap$")
		if definitions == 0 {
			t.Fatalf("map specialization shard %s is empty", file.OutputPath())
		}
		generatedDefinitions += definitions
		nativeDefinitions += strings.Count(
			source,
			"private readonly values: Map<",
		)
		hashedDefinitions += strings.Count(
			source,
			"private readonly buckets: Map<number,",
		)
		hashedBucketLoops += strings.Count(source, "for (const entry of bucket)")
		totalBytes += len(source)
		if len(source) > largestBytes {
			largestBytes = len(source)
			largestPath = file.OutputPath()
		}
		for name, classSource := range mapClassSources(source) {
			if payloadBytes := mapClassPayloadBytes(name, classSource); payloadBytes > 7_200 {
				t.Fatalf(
					"map specialization %s#%s = %d semantic payload bytes (%d actual), want at most 7200 payload bytes",
					file.OutputPath(),
					name,
					payloadBytes,
					len(classSource),
				)
			}
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
			"GoDenseIndex",
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
	if generatedDefinitions != 7 {
		t.Fatalf(
			"map specialization definitions = %d, want seven exact reached shapes",
			generatedDefinitions,
		)
	}
	if nativeDefinitions != 3 ||
		hashedDefinitions != 4 ||
		hashedBucketLoops != hashedDefinitions*3 {
		t.Fatalf(
			"map storage shapes = native:%d hashed:%d typed-bucket-loops:%d, want 3/4/12",
			nativeDefinitions,
			hashedDefinitions,
			hashedBucketLoops,
		)
	}
	assertProjectedPrimitiveMap(t, artifacts)
	anonymous := readFile(
		t,
		artifacts.file(t, output.AnonymousStructSupportPath),
	)
	if strings.Count(anonymous, "export class $goStruct$") != 2 ||
		strings.Count(anonymous, "static $hash(") != 1 {
		t.Fatalf(
			"anonymous map dependency owns the wrong classes/hash surface:\n%s",
			anonymous,
		)
	}
	t.Logf(
		"map specialization-files=%d definitions=%d total-bytes=%d largest=%s/%d",
		generatedFiles,
		generatedDefinitions,
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
	fmt.Println(values.NamedKeyLifecycle())
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
    NamedKeyLifecycle,
    NamedValueLifecycle,
    StructKeyLifecycle,
} from "`+artifacts.module(t, "source.ts")+`";
import "./program.js";

function show(value: number | bigint | boolean | string): string {
    return typeof value === "bigint" ? value.toString() : String(value);
}
function print(...values: Array<number | bigint | boolean | string>): void {
    console.log(...values.map(show));
}

print(...NamedValueLifecycle());
print(...ArrayKeyLifecycle());
print(...StructKeyLifecycle());
print(...AnonymousShapeLifecycle());
print(...CollisionEquality());
print(...NamedKeyLifecycle());
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

func assertProjectedPrimitiveMap(t *testing.T, artifacts materialized) {
	t.Helper()
	source := readFile(t, artifacts.file(t, output.MapSpecializationSupportPath))
	var selected string
	for name, classSource := range mapClassSources(source) {
		if name == "$goMap$MapOf_Named_aggregatemap$Label_To_int32" {
			selected = classSource
			break
		}
	}
	if selected == "" {
		var names []string
		for name := range mapClassSources(source) {
			names = append(names, name)
		}
		slices.Sort(names)
		t.Fatalf("defined-string key map specialization is absent; classes=%v", names)
	}
	for _, required := range []string{
		"private readonly values: Map<gostring, int32>",
		"private static $projectKey($key: Label__from_aggregatemap): gostring",
		"private static $reifyKey($storageKey: gostring): Label__from_aggregatemap",
		"values.get(storageKey)",
		"values.set(storageKey, ",
		"values.has(storageKey)",
		"values.delete(storageKey)",
		"result.push(",
	} {
		if !strings.Contains(selected, required) {
			t.Fatalf("defined-string native map lacks %q:\n%s", required, selected)
		}
	}
	for _, forbidden := range []string{
		"$hash(",
		"$equal(",
		"$find(",
		"buckets",
		"Map<gostring, [",
		"values.set(storageKey, [",
	} {
		if strings.Contains(selected, forbidden) {
			t.Fatalf("defined-string native map contains %q:\n%s", forbidden, selected)
		}
	}
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

func withoutSourceFactApplicationsByPath(contents map[string]string) map[string]string {
	result := make(map[string]string, len(contents))
	for path, content := range contents {
		result[path] = withoutSourceFactApplications(content)
	}
	return result
}

func withoutSourceFactApplications(source string) string {
	var result strings.Builder
	for _, line := range strings.SplitAfter(source, "\n") {
		if strings.HasPrefix(line, "attribute<") {
			continue
		}
		result.WriteString(line)
	}
	return result.String()
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
