package emit_test

import (
	"context"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
)

func TestNativeNumericMethodsPreserveEveryCallableForm(t *testing.T) {
	directory := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(directory, "go.mod"),
		"module example.com/numericmethods\n\ngo 1.26.4\n",
	)
	writeProgramFile(t, filepath.Join(directory, "value.go"), `package numericmethods

type Flags uint32

const (
	FlagsOne  Flags = 1
	FlagsLast Flags = 99
)

func (flags Flags) Has(mask Flags) bool { return flags&mask != 0 }
func (flags *Flags) Add(mask Flags) { *flags |= mask }
func (flags Flags) Deferred() (result Flags) {
	defer func() { result |= flags }()
	return 1
}

type Tester interface { Has(Flags) bool }

func PointerExpression() func(*Flags, Flags) { return (*Flags).Add }

func OpenSwitch(flags Flags, values []uint32) uint32 {
	switch flags {
	case FlagsOne:
		return 1
	case FlagsLast:
		return 2
	}
	for _, value := range values {
		return value
	}
	return 0
}

func Audit() uint32 {
	flags := Flags(3)
	score := uint32(0)
	if flags.Has(1) { score += 1 }
	expression := Flags.Has
	if expression(flags, 2) { score += 2 }
	value := flags.Has
	if value(4) { score += 8 }
	var contract Tester = flags
	if contract.Has(2) { score += 4 }
	return score + uint32(flags) + OpenSwitch(Flags(3), []uint32{16})
}
`)
	program, err := load.Load(context.Background(), load.Request{
		Directory: directory,
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
	workingDirectory := t.TempDir()
	artifacts := materializeArtifacts(t, emission, workingDirectory)
	for _, required := range []string{
		"export type Flags = uint32 & {",
		`readonly $goType?: "example.com/numericmethods|Flags";`,
		"export function Flags_Has(flags: Flags, mask: Flags): bool",
		"export function Flags_Add(flags: Pointer<Flags> | undefined, mask: Flags): void",
		"Flags_Has__from_numericmethods(this.$go$value, $argument0)",
	} {
		if !strings.Contains(artifacts.printed, required) {
			t.Fatalf("native numeric method artifact lacks %q:\n%s", required, artifacts.printed)
		}
	}
	for _, forbidden := range []string{
		"export enum Flags",
		"export class Flags",
		"new Flags(",
		".$value",
		"Flags.$goType",
	} {
		if strings.Contains(artifacts.printed, forbidden) {
			t.Fatalf("native numeric method artifact contains %q:\n%s", forbidden, artifacts.printed)
		}
	}
	packageModule := ""
	for _, file := range emission.Files() {
		if file.Kind() == emit.TargetFilePackageAssembly {
			packageModule = "./" + strings.TrimSuffix(file.OutputPath(), ".ts") + ".js"
			break
		}
	}
	if packageModule == "" {
		t.Fatal("native numeric method package assembly is absent")
	}
	runner := filepath.Join(workingDirectory, "runner.ts")
	writeProgramFile(t, runner, `import "./program.js";
import { Audit, Flags_Add, Flags_Has } from "`+packageModule+`";

void Flags_Add;
void Flags_Has;
console.log(String(Audit()));
`)
	writeProgramFile(
		t,
		filepath.Join(workingDirectory, "package.json"),
		"{\"type\":\"module\"}\n",
	)
	waveThreeTypecheck(t, workingDirectory, append(artifacts.paths, runner))
	targetOutput := runProgram(
		t,
		workingDirectory,
		"node",
		filepath.Join(workingDirectory, "out", "runner.js"),
	)
	writeProgramFile(t, filepath.Join(directory, "cmd", "check", "main.go"), `package main

import (
	"fmt"
	"example.com/numericmethods"
)

func main() { fmt.Println(numericmethods.Audit()) }
`)
	goOutput := runProgram(
		t,
		directory,
		filepath.Join(runtime.GOROOT(), "bin", "go"),
		"run",
		"./cmd/check",
	)
	if targetOutput != goOutput {
		t.Fatalf(
			"native numeric method output differs\nTypeScript: %q\nGo: %q",
			targetOutput,
			goOutput,
		)
	}
}
