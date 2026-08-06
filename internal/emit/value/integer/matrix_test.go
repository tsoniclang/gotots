package integer_test

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

var matrixTypes = []struct {
	name     string
	unsigned bool
	bitwise  bool
}{
	{"Int8", false, true},
	{"Int16", false, true},
	{"Int32", false, true},
	{"Int64", false, false},
	{"Uint8", true, true},
	{"Uint16", true, true},
	{"Uint32", true, true},
	{"Uint64", true, false},
}

func TestEveryAdmittedIntegerOperatorAndWidthExecutesDifferentially(t *testing.T) {
	for _, representation := range []emit.IntegerRepresentation{
		emit.IntegerRepresentationNumber,
		emit.IntegerRepresentationBigInt,
	} {
		t.Run(representation.String(), func(t *testing.T) {
			directory := t.TempDir()
			writeFile(t, filepath.Join(directory, "go.mod"), `module example.com/integermatrix

go 1.26.4
`)
			writeFile(t, filepath.Join(directory, "source.go"), integerMatrixSource())
			loaded, err := load.One(context.Background(), load.Request{
				Directory: directory,
				Pattern:   ".",
			})
			if err != nil {
				t.Fatal(err)
			}
			options := emit.DefaultOptions()
			options.IntegerRepresentation = representation
			roots := matrixRoots(t, loaded, representation)
			emission, err := emit.CompileWithOptions(loaded.Program(), roots, options)
			if err != nil {
				t.Fatal(err)
			}
			workingDirectory := t.TempDir()
			goOutput := executeMatrixGo(
				t,
				directory,
				workingDirectory,
				representation,
			)
			targetOutput := executeMatrixTypeScript(
				t,
				emission,
				workingDirectory,
				representation,
			)
			if targetOutput != goOutput {
				t.Fatalf(
					"%s TypeScript output = %q, Go output = %q",
					representation,
					targetOutput,
					goOutput,
				)
			}
		})
	}
}

func integerMatrixSource() string {
	var source strings.Builder
	source.WriteString("package integermatrix\n\n")
	for _, target := range matrixTypes {
		sourceType := strings.ToLower(target.name)
		fmt.Fprintf(
			&source,
			"func Number%s(left, right %s) ",
			target.name,
			sourceType,
		)
		if target.bitwise && !target.unsigned {
			fmt.Fprintf(
				&source,
				"(%[1]s, %[1]s, %[1]s, %[1]s, %[1]s, %[1]s, %[1]s, %[1]s, %[1]s, %[1]s, %[1]s, %[1]s, bool) {\n"+
					"\treturn left + right, left - right, left * right, left & right, left | right, left ^ right, left &^ right, left << 1, left >> 1, +left, -left, ^left, left < right\n}\n\n",
				sourceType,
			)
		} else if target.bitwise {
			fmt.Fprintf(
				&source,
				"(%[1]s, %[1]s, %[1]s, %[1]s, %[1]s, %[1]s, %[1]s, %[1]s, %[1]s, %[1]s, %[1]s, bool) {\n"+
					"\treturn left + right, left - right, left * right, left & right, left | right, left ^ right, left &^ right, left << 1, left >> 1, +left, ^left, left < right\n}\n\n",
				sourceType,
			)
		} else if target.unsigned {
			fmt.Fprintf(
				&source,
				"(%[1]s, %[1]s, %[1]s, %[1]s, bool) {\n"+
					"\treturn left + right, left - right, left * right, +left, left < right\n}\n\n",
				sourceType,
			)
		} else {
			fmt.Fprintf(
				&source,
				"(%[1]s, %[1]s, %[1]s, %[1]s, %[1]s, bool) {\n"+
					"\treturn left + right, left - right, left * right, +left, -left, left < right\n}\n\n",
				sourceType,
			)
		}
		bigResults := strings.TrimSuffix(strings.Repeat(sourceType+", ", 14), ", ") + ", bool"
		bigValues := "left + right, left - right, left * right, left / right, left % right, left & right, left | right, left ^ right, left &^ right, left << 1, left >> 1, +left, -left, ^left, left < right"
		if target.unsigned {
			bigResults = strings.TrimSuffix(strings.Repeat(sourceType+", ", 13), ", ") + ", bool"
			bigValues = "left + right, left - right, left * right, left / right, left % right, left & right, left | right, left ^ right, left &^ right, left << 1, left >> 1, +left, ^left, left < right"
		}
		fmt.Fprintf(
			&source,
			"func Big%s(left, right %s) (%s) {\n\treturn %s\n}\n\n",
			target.name,
			sourceType,
			bigResults,
			bigValues,
		)
	}
	return source.String()
}

func matrixRoots(
	t *testing.T,
	loaded *load.Package,
	representation emit.IntegerRepresentation,
) []emit.Root {
	t.Helper()
	prefix := "Number"
	if representation == emit.IntegerRepresentationBigInt {
		prefix = "Big"
	}
	roots := make([]emit.Root, 0, len(matrixTypes))
	for _, sourceType := range matrixTypes {
		object := loaded.Types().Scope().Lookup(prefix + sourceType.name)
		root, err := emit.NewRoot(object)
		if err != nil {
			t.Fatal(err)
		}
		roots = append(roots, root)
	}
	return roots
}

func executeMatrixTypeScript(
	t *testing.T,
	emission emit.ProgramEmission,
	workingDirectory string,
	representation emit.IntegerRepresentation,
) string {
	t.Helper()
	artifacts := materializeIntegerFamily(t, emission, workingDirectory)
	prefix := "Number"
	if representation == emit.IntegerRepresentationBigInt {
		prefix = "Big"
	}
	var runner strings.Builder
	fmt.Fprintf(
		&runner,
		"import * as values from %q;\n\n"+
			"const show = (value: number | bigint | boolean): string => value.toString();\n"+
			"const row = (value: readonly (number | bigint | boolean)[]): string => value.map(show).join(\" \");\n",
		artifacts.module(t, "source.ts"),
	)
	for _, sourceType := range matrixTypes {
		suffix := ""
		if representation == emit.IntegerRepresentationBigInt &&
			(sourceType.name == "Int64" || sourceType.name == "Uint64") {
			suffix = "n"
		}
		left := "12"
		if !sourceType.unsigned {
			left = "-12"
		}
		fmt.Fprintf(
			&runner,
			"console.log(row(values.%s%s(%s%s, 5%s)));\n",
			prefix,
			sourceType.name,
			left,
			suffix,
			suffix,
		)
	}
	runnerPath := filepath.Join(workingDirectory, "runner.ts")
	writeFile(t, runnerPath, runner.String())
	return executeMaterializedTypeScript(t, workingDirectory, artifacts, runnerPath)
}

func executeMatrixGo(
	t *testing.T,
	moduleDirectory string,
	workingDirectory string,
	representation emit.IntegerRepresentation,
) string {
	t.Helper()
	prefix := "Number"
	if representation == emit.IntegerRepresentationBigInt {
		prefix = "Big"
	}
	runnerDirectory := filepath.Join(workingDirectory, "go-runner")
	writeFile(t, filepath.Join(runnerDirectory, "go.mod"), fmt.Sprintf(`module example.com/runner

go 1.26.4

require example.com/integermatrix v0.0.0

replace example.com/integermatrix => %s
`, filepath.ToSlash(moduleDirectory)))
	var calls strings.Builder
	for _, sourceType := range matrixTypes {
		left := "12"
		if !sourceType.unsigned {
			left = "-12"
		}
		fmt.Fprintf(
			&calls,
			"\tfmt.Println(values.%s%s(%s, 5))\n",
			prefix,
			sourceType.name,
			left,
		)
	}
	writeFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"
	values "example.com/integermatrix"
)

func main() {
`+calls.String()+`}
`)
	return run(
		t,
		runnerDirectory,
		filepath.Join(runtime.GOROOT(), "bin", "go"),
		"run",
		".",
	)
}
