package constant_test

import (
	"context"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
)

func TestDefinedPackageConstantSurvivesSourceModuleCycle(t *testing.T) {
	directory := t.TempDir()
	writeFile(t, filepath.Join(directory, "go.mod"), "module example.com/constantcycle\n\ngo 1.26.4\n")
	writeFile(t, filepath.Join(directory, "compact.go"), `package constantcycle

type ID uint16

func (id ID) Value() uint16 { return uint16(id) }

func Use() uint16 { return Value.Value() }
`)
	writeFile(t, filepath.Join(directory, "tables.go"), `package constantcycle

const Value ID = 7
`)

	loaded, err := load.One(context.Background(), load.Request{
		Directory: directory,
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	roots, err := emit.ExportedAPIRoots(loaded)
	if err != nil {
		t.Fatal(err)
	}
	emission, err := emit.Compile(loaded.Program(), roots)
	if err != nil {
		t.Fatal(err)
	}
	printed := printConstantFamily(t, emission)
	for _, expected := range []string{
		"export function Value$constant(): ID",
		"return Value$constant().Value()",
		"export let Value: ReturnType<typeof Value$constant>",
		"Value = Value$constant()",
	} {
		if !strings.Contains(printed, expected) {
			t.Fatalf("constant-cycle artifact lacks %q:\n%s", expected, printed)
		}
	}
	if strings.Contains(printed, "export const Value") {
		t.Fatalf("constant-cycle artifact retained eager construction:\n%s", printed)
	}

	workingDirectory := t.TempDir()
	artifacts := materializeConstantFamily(t, emission, workingDirectory)
	packageModule := ""
	for _, file := range emission.Files() {
		if file.Kind() == emit.TargetFilePackageAssembly {
			packageModule = "./" + strings.TrimSuffix(file.OutputPath(), ".ts") + ".js"
			break
		}
	}
	if packageModule == "" {
		t.Fatal("constant-cycle package assembly is absent")
	}
	runnerPath := filepath.Join(workingDirectory, "runner.ts")
	writeFile(t, runnerPath, `import "./program.js";
import { Use, Value } from "`+packageModule+`";

console.log(String(Use()));
console.log(String(Value.Value()));
`)
	targetOutput := executeMaterializedTypeScript(
		t,
		workingDirectory,
		artifacts,
		runnerPath,
	)
	writeFile(t, filepath.Join(directory, "cmd", "check", "main.go"), `package main

import (
	"fmt"

	values "example.com/constantcycle"
)

func main() {
	fmt.Println(values.Use())
	fmt.Println(values.Value.Value())
}
`)
	goOutput := run(
		t,
		directory,
		filepath.Join(runtime.GOROOT(), "bin", "go"),
		"run",
		"./cmd/check",
	)
	if targetOutput != goOutput {
		t.Fatalf(
			"constant-cycle TypeScript output = %q, Go output = %q",
			targetOutput,
			goOutput,
		)
	}
}

func TestCrossPackageDefinedConstantUsesCertifiedPackageThunk(t *testing.T) {
	directory := t.TempDir()
	writeFile(t, filepath.Join(directory, "go.mod"), "module example.com/constantthunk\n\ngo 1.26.4\n")
	writeFile(t, filepath.Join(directory, "defs", "defs.go"), `package defs

type ID uint16

func (id ID) Value() uint16 { return uint16(id) }

const Number ID = 9
`)
	writeFile(t, filepath.Join(directory, "api", "api.go"), `package api

import "example.com/constantthunk/defs"

func Read() uint16 { return defs.Number.Value() }
`)

	program, err := load.Load(context.Background(), load.Request{
		Directory: directory,
		Pattern:   "./api",
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
	printed := printConstantFamily(t, emission)
	for _, expected := range []string{
		"export function Number$constant(): ID",
		"Number$constant as Number$constant__from_defs",
		"return Number$constant__from_defs().Value()",
		"export let Number: ReturnType<typeof Number$constant>",
		"Number = Number$constant()",
	} {
		if !strings.Contains(printed, expected) {
			t.Fatalf("cross-package constant artifact lacks %q:\n%s", expected, printed)
		}
	}

	workingDirectory := t.TempDir()
	artifacts := materializeConstantFamily(t, emission, workingDirectory)
	apiModule := ""
	for _, file := range emission.Files() {
		if file.Kind() == emit.TargetFilePackageAssembly &&
			file.PackageName() == "api" {
			apiModule = "./" + strings.TrimSuffix(file.OutputPath(), ".ts") + ".js"
			break
		}
	}
	if apiModule == "" {
		t.Fatal("constant-thunk API assembly is absent")
	}
	runnerPath := filepath.Join(workingDirectory, "runner.ts")
	writeFile(t, runnerPath, `import "./program.js";
import { Read } from "`+apiModule+`";

console.log(String(Read()));
`)
	targetOutput := executeMaterializedTypeScript(
		t,
		workingDirectory,
		artifacts,
		runnerPath,
	)
	writeFile(t, filepath.Join(directory, "cmd", "check", "main.go"), `package main

import (
	"fmt"

	"example.com/constantthunk/api"
)

func main() {
	fmt.Println(api.Read())
}
`)
	goOutput := run(
		t,
		directory,
		filepath.Join(runtime.GOROOT(), "bin", "go"),
		"run",
		"./cmd/check",
	)
	if targetOutput != goOutput {
		t.Fatalf(
			"cross-package constant TypeScript output = %q, Go output = %q",
			targetOutput,
			goOutput,
		)
	}
}
