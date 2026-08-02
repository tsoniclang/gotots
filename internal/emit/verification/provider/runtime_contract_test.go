package provider_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
)

func TestSelectedProviderContributesCertifiedRuntimeClosure(t *testing.T) {
	program := loadProviderRuntimeProgram(t)
	root := mustProviderRoot(
		t,
		program.Roots()[0].Types().Scope().Lookup("ReflectType"),
	)
	certificate := linkedProviderCertificate(t)
	options := emit.DefaultOptions()
	options.StandardLibrary = certificate
	emission, err := emit.CompileWithOptions(program, []emit.Root{root}, options)
	if err != nil {
		t.Fatal(err)
	}
	workingDirectory := t.TempDir()
	artifacts := materializeArtifacts(t, emission, workingDirectory)
	waveThreeTypecheck(t, workingDirectory, artifacts.paths)
	for _, required := range []string{
		"runtime/complex.ts",
		"export class GoComplex128",
		"export type int32 = number;",
		"export type float64 = number;",
		"reflect__from_gostdlib.Type",
	} {
		if !strings.Contains(artifacts.printed, required) {
			t.Fatalf("provider runtime closure lacks %q:\n%s", required, artifacts.printed)
		}
	}
	if strings.Contains(artifacts.printed, "$goProviderInterfaceBridge") {
		t.Fatal("sealed reflect.Type emitted an open-interface bridge")
	}
}

func TestUnselectedProviderDoesNotExpandRuntimeClosure(t *testing.T) {
	program := loadProviderRuntimeProgram(t)
	root := mustProviderRoot(
		t,
		program.Roots()[0].Types().Scope().Lookup("Identity"),
	)
	options := emit.DefaultOptions()
	options.StandardLibrary = linkedProviderCertificate(t)
	emission, err := emit.CompileWithOptions(program, []emit.Root{root}, options)
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range emission.Files() {
		if file.OutputPath() == "runtime/complex.ts" {
			t.Fatal("unselected provider expanded the runtime closure")
		}
	}
}

func TestSelectedProviderRejectsUncertifiedIntegerProfile(t *testing.T) {
	program := loadProviderRuntimeProgram(t)
	root := mustProviderRoot(
		t,
		program.Roots()[0].Types().Scope().Lookup("ReflectType"),
	)
	options := emit.DefaultOptions()
	options.IntegerRepresentation = emit.IntegerRepresentationBigInt
	options.StandardLibrary = linkedProviderCertificate(t)
	if _, err := emit.CompileWithOptions(program, []emit.Root{root}, options); err == nil {
		t.Fatal("provider accepted an uncertified integer profile")
	}
}

func loadProviderRuntimeProgram(t *testing.T) *load.Program {
	t.Helper()
	project := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(project, "go.mod"),
		"module example.com/providerruntime\n\ngo 1.26.4\n",
	)
	writeProgramFile(t, filepath.Join(project, "source.go"), `package providerruntime

import "reflect"

func ReflectType(value any) reflect.Type { return reflect.TypeOf(value) }
func Identity(value int) int { return value }
`)
	program, err := load.Load(context.Background(), load.Request{
		Directory:    project,
		Pattern:      ".",
		BuildProfile: linkedProviderBuildProfile(t),
	})
	if err != nil {
		t.Fatal(err)
	}
	return program
}
