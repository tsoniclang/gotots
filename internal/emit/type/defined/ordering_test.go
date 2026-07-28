package defined_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestAccessorCompoundPreserveGoOrderExecutesDifferentially(t *testing.T) {
	loaded, err := load.One(context.Background(), load.Request{
		Directory: definedFixtureDirectory(),
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	roots, err := emit.ExportedAPIRoots(loaded)
	if err != nil {
		t.Fatal(err)
	}
	emission, err := emit.CompileWithOptions(
		loaded.Program(),
		roots,
		emit.Options{
			IntegerRepresentation: emit.IntegerRepresentationNumber,
			EvaluationOrder:       emit.EvaluationOrderPreserveGo,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	workingDirectory := t.TempDir()
	artifacts := printDefined(t, workingDirectory, emission)
	assertCompoundArtifactShape(t, artifacts.printed)
	goOutput := executeCompoundOrderGo(t, workingDirectory)
	targetOutput := executeCompoundOrderTypeScript(
		t,
		workingDirectory,
		artifacts,
	)
	if targetOutput != goOutput {
		t.Fatalf(
			"compound update order differs\nTypeScript: %q\nGo: %q",
			targetOutput,
			goOutput,
		)
	}
}

func assertCompoundArtifactShape(t *testing.T, printed string) {
	t.Helper()
	const startMarker = "export function CountArrayCompoundOrder"
	start := strings.Index(printed, startMarker)
	if start < 0 {
		t.Fatalf("defined artifact has no %s", startMarker)
	}
	body := printed[start:]
	if next := strings.Index(body[len(startMarker):], "\nexport function "); next >= 0 {
		body = body[:len(startMarker)+next]
	}
	if got := strings.Count(body, "const __gotots_store_"); got != 2 {
		t.Fatalf("accessor compound location captures = %d, want 2:\n%s", got, body)
	}
	if got := strings.Count(body, "const __gotots_assign_"); got != 1 {
		t.Fatalf("accessor compound right captures = %d, want 1:\n%s", got, body)
	}
	right := strings.Index(body, "const __gotots_assign_")
	rightEnd := -1
	if right >= 0 {
		rightEnd = strings.Index(body[right:], ";")
	}
	if right < 0 ||
		rightEnd < 0 ||
		!strings.Contains(body[right:right+rightEnd], "= __gotots_callee_") {
		t.Fatalf("accessor compound has no captured guarded-call result:\n%s", body)
	}
	compound := body[right:]
	read := strings.Index(compound, ".get(")
	store := strings.Index(compound, ".set(")
	if read < 0 || store < 0 || store > read {
		t.Fatalf("accessor compound setter does not consume one getter:\n%s", body)
	}
}

func executeCompoundOrderGo(
	t *testing.T,
	workingDirectory string,
) string {
	t.Helper()
	runnerDirectory := filepath.Join(workingDirectory, "order-go")
	writeDefinedFile(t, filepath.Join(runnerDirectory, "go.mod"), `module example.com/order

go 1.26.4

require example.com/definedbasic v0.0.0

replace example.com/definedbasic => `+
		filepath.ToSlash(definedFixtureDirectory())+"\n")
	writeDefinedFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"
	values "example.com/definedbasic"
)

func main() {
	fmt.Println(values.IntFromCount(values.CountArrayCompoundOrder()))
}
`)
	return runDefinedCommand(t, runnerDirectory, "go", "run", ".")
}

func executeCompoundOrderTypeScript(
	t *testing.T,
	workingDirectory string,
	artifacts printedDefined,
) string {
	t.Helper()
	runnerPath := filepath.Join(workingDirectory, "order.ts")
	writeDefinedFile(t, runnerPath, `import * as values from "`+
		artifacts.sourceModule+`";
console.log(String(values.IntFromCount(values.CountArrayCompoundOrder())));
`)
	writeDefinedFile(
		t,
		filepath.Join(workingDirectory, "package.json"),
		"{\"type\":\"module\"}\n",
	)
	outputDirectory := filepath.Join(workingDirectory, "order-out")
	arguments := []string{
		"--target", "es2022",
		"--module", "nodenext",
		"--moduleResolution", "nodenext",
		"--strict",
		"--outDir", outputDirectory,
	}
	arguments = append(arguments, artifacts.paths...)
	arguments = append(arguments, runnerPath)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := tsgo.Compile(
		ctx,
		repositoryRoot(),
		workingDirectory,
		arguments,
	); err != nil {
		t.Fatal(err)
	}
	return runDefinedCommand(
		t,
		workingDirectory,
		"node",
		filepath.Join(outputDirectory, "order.js"),
	)
}
