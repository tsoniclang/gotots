package maprepresentation_test

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/output"
)

func TestWideNativeMapBigIntCarrierProfilesExecuteDifferentially(t *testing.T) {
	goOutput := executeWideNativeMapGo(t, t.TempDir())
	for _, profile := range []struct {
		name           string
		representation emit.IntegerRepresentation
	}{
		{"fixed64-bigint", emit.IntegerRepresentationFixed64BigInt},
		{"bigint", emit.IntegerRepresentationBigInt},
	} {
		t.Run(profile.name, func(t *testing.T) {
			emission := compileAggregateMapProfile(t, profile.representation)
			workingDirectory := t.TempDir()
			artifacts := materialize(t, emission, workingDirectory)
			assertWideNativeMapCarrier(
				t,
				emission,
				artifacts,
				"bigint",
				true,
			)
			targetOutput := executeWideNativeMapTypeScript(
				t,
				artifacts,
				workingDirectory,
			)
			if targetOutput != goOutput {
				t.Fatalf(
					"%s wide native map TypeScript/Go = %q/%q",
					profile.name,
					targetOutput,
					goOutput,
				)
			}
		})
	}
}

func TestWideNativeMapNumberProfileExecutesDeclaredCollisionTradeoff(
	t *testing.T,
) {
	emission := compileAggregateMapProfile(
		t,
		emit.IntegerRepresentationNumber,
	)
	workingDirectory := t.TempDir()
	artifacts := materialize(t, emission, workingDirectory)
	assertWideNativeMapCarrier(t, emission, artifacts, "number", false)
	targetOutput := executeWideNativeMapTypeScript(
		t,
		artifacts,
		workingDirectory,
	)
	const selectedNumberOutput = "90 20 21 20 30 25 true true true false false false 0\n"
	if targetOutput != selectedNumberOutput {
		t.Fatalf(
			"number-carrier wide native map output = %q, want declared collision envelope %q",
			targetOutput,
			selectedNumberOutput,
		)
	}
	goOutput := executeWideNativeMapGo(t, t.TempDir())
	if targetOutput == goOutput {
		t.Fatalf("number carrier unexpectedly claimed exact uint64 map output %q", targetOutput)
	}
	t.Logf(
		"number uint64 collision tradeoff: TypeScript=%q Go=%q",
		strings.TrimSpace(targetOutput),
		strings.TrimSpace(goOutput),
	)
}

func compileAggregateMapProfile(
	t *testing.T,
	representation emit.IntegerRepresentation,
) emit.ProgramEmission {
	t.Helper()
	loaded := loadAggregateMapProject(t)
	roots, err := emit.ExportedAPIRoots(loaded)
	if err != nil {
		t.Fatal(err)
	}
	options := mapNumberOptions()
	options.IntegerRepresentation = representation
	emission, err := emit.CompileWithOptions(loaded.Program(), roots, options)
	if err != nil {
		t.Fatal(err)
	}
	return emission
}

func assertWideNativeMapCarrier(
	t *testing.T,
	emission emit.ProgramEmission,
	artifacts materialized,
	carrier string,
	exact bool,
) {
	t.Helper()
	scalars := readFile(t, artifacts.file(t, "runtime/scalars.ts"))
	wantCarrier := carrier
	if carrier == "bigint" {
		wantCarrier = "$go$core$uint64"
	}
	if !strings.Contains(scalars, "export type uint64 = "+wantCarrier+";") {
		t.Fatalf("uint64 carrier is not %s:\n%s", carrier, scalars)
	}
	wideDefinitions := 0
	for _, file := range emission.Files() {
		if file.OutputPath() != output.MapSpecializationSupportPath {
			continue
		}
		source := readFile(t, artifacts.file(t, file.OutputPath()))
		if !strings.Contains(source, "Map<uint64, Box__from_aggregatemap>") {
			continue
		}
		for _, classSource := range mapClassSources(source) {
			if !strings.Contains(classSource, "Map<uint64, Box__from_aggregatemap>") {
				continue
			}
			wideDefinitions++
			for _, forbidden := range []string{
				"$hash(",
				"$equal(",
				"$find(",
				"GoDenseIndex",
				"Map<uint64, [",
			} {
				if strings.Contains(classSource, forbidden) {
					t.Fatalf("wide native map contains %q:\n%s", forbidden, classSource)
				}
			}
		}
	}
	if wideDefinitions != 1 {
		t.Fatalf("wide native map definitions = %d, want one", wideDefinitions)
	}
	program := readFile(t, artifacts.file(t, "source.ts"))
	for _, boundary := range []string{
		"9007199254740993",
		"18446744073709551615",
	} {
		hasBigInt := strings.Contains(program, boundary+"n")
		if hasBigInt != exact {
			t.Fatalf(
				"uint64 boundary %s bigint literal = %t, want %t:\n%s",
				boundary,
				hasBigInt,
				exact,
				program,
			)
		}
	}
}

func executeWideNativeMapTypeScript(
	t *testing.T,
	artifacts materialized,
	workingDirectory string,
) string {
	t.Helper()
	runnerPath := filepath.Join(workingDirectory, "wide-runner.ts")
	writeFile(t, runnerPath, `import { WideKeyLifecycle } from "`+
		artifacts.module(t, "source.ts")+`";
import "./program.js";

const show = (value: number | bigint | boolean): string => String(value);
console.log(...WideKeyLifecycle().map(show));
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
		filepath.Join(workingDirectory, "out", "wide-runner.js"),
	)
}

func executeWideNativeMapGo(t *testing.T, workingDirectory string) string {
	t.Helper()
	modulePath, err := filepath.Abs(aggregateMapProjectDirectory())
	if err != nil {
		t.Fatal(err)
	}
	runnerDirectory := filepath.Join(workingDirectory, "wide-go-runner")
	writeFile(t, filepath.Join(runnerDirectory, "go.mod"), fmt.Sprintf(`module example.com/widerunner

go 1.26.4

require example.com/aggregatemap v0.0.0

replace example.com/aggregatemap => %s
`, filepath.ToSlash(modulePath)))
	writeFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"
	values "example.com/aggregatemap"
)

func main() { fmt.Println(values.WideKeyLifecycle()) }
`)
	return run(
		t,
		runnerDirectory,
		filepath.Join(runtime.GOROOT(), "bin", "go"),
		"run",
		".",
	)
}
