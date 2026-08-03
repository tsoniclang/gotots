package provider_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/output"
)

func TestReflectTypeForUsesCanonicalGeneratedMetadata(t *testing.T) {
	program := loadProviderRuntimeProgram(t)
	root := mustProviderRoot(
		t,
		program.Roots()[0].Types().Scope().Lookup("ReflectStatic"),
	)
	options := emit.DefaultOptions()
	options.StandardLibrary = linkedProviderCertificate(t)
	emission, err := emit.CompileWithOptions(program, []emit.Root{root}, options)
	if err != nil {
		t.Fatal(err)
	}
	workingDirectory := t.TempDir()
	artifacts := materializeArtifacts(t, emission, workingDirectory)
	reflectionSource := readGeneratedArtifact(
		t,
		workingDirectory,
		output.ReflectionTypeSupportPath,
	)
	if strings.Contains(artifacts.printed, ".TypeFor<") {
		t.Fatalf("TypeFor retained an erased TypeScript generic call:\n%s", artifacts.printed)
	}
	if !strings.Contains(artifacts.printed, "$goReflectType_") {
		t.Fatalf("TypeFor emitted no canonical runtime-type reference:\n%s", artifacts.printed)
	}
	for _, redundant := range []string{
		"assignableTo:",
		"convertibleTo:",
		"implements:",
		"fieldAlign:",
		"index: [0]",
	} {
		if strings.Contains(reflectionSource, redundant) {
			t.Fatalf(
				"reflection metadata retained redundant %q:\n%s",
				redundant,
				reflectionSource,
			)
		}
	}
	assemblyPath := ""
	for _, file := range emission.Files() {
		if file.Kind() == emit.TargetFilePackageAssembly &&
			file.PackageName() == "providerruntime" {
			assemblyPath = file.OutputPath()
			break
		}
	}
	if assemblyPath == "" {
		t.Fatal("provider runtime package assembly is absent")
	}
	targetOutput := executeProviderTypeScript(
		t,
		workingDirectory,
		artifacts.paths,
		assemblyPath,
		[]string{"ReflectStatic"},
		`const [kind, tag, fields] = ReflectStatic();
console.log(kind + "|" + tag + "|" + fields);
`,
	)
	if targetOutput != "struct|name|1\n" {
		t.Fatalf("reflection differential = %q, want %q", targetOutput, "struct|name|1\n")
	}
}

func TestReflectTypeOfUsesRegisteredCanonicalDynamicType(t *testing.T) {
	program := loadProviderRuntimeProgram(t)
	root := mustProviderRoot(
		t,
		program.Roots()[0].Types().Scope().Lookup("ReflectDynamicFacts"),
	)
	options := emit.DefaultOptions()
	options.StandardLibrary = linkedProviderCertificate(t)
	emission, err := emit.CompileWithOptions(program, []emit.Root{root}, options)
	if err != nil {
		t.Fatal(err)
	}
	workingDirectory := t.TempDir()
	artifacts := materializeArtifacts(t, emission, workingDirectory)
	if strings.Contains(artifacts.printed, ".TypeOf(") {
		t.Fatalf("TypeOf retained a provider placeholder call:\n%s", artifacts.printed)
	}
	if !strings.Contains(
		artifacts.printed,
		`import "../../../support/reflection-types.js";`,
	) {
		t.Fatalf("TypeOf emitted no static metadata initialization import:\n%s", artifacts.printed)
	}
	reflectionSource := readGeneratedArtifact(
		t,
		workingDirectory,
		output.ReflectionTypeSupportPath,
	)
	if strings.Contains(reflectionSource, "providerruntime.Unrelated") {
		t.Fatalf(
			"unrelated interface adapter acquired reflection metadata:\n%s",
			reflectionSource,
		)
	}
	assemblyPath := ""
	for _, file := range emission.Files() {
		if file.Kind() == emit.TargetFilePackageAssembly &&
			file.PackageName() == "providerruntime" {
			assemblyPath = file.OutputPath()
			break
		}
	}
	if assemblyPath == "" {
		t.Fatal("provider runtime package assembly is absent")
	}
	targetOutput := executeProviderTypeScript(
		t,
		workingDirectory,
		artifacts.paths,
		assemblyPath,
		[]string{"ReflectDynamicFacts"},
		`const [kind, absent] = ReflectDynamicFacts();
console.log(kind + "|" + absent);
`,
	)
	if targetOutput != "string|true\n" {
		t.Fatalf("dynamic reflection differential = %q", targetOutput)
	}
}

func TestReflectTypeForOpenGenericUsesPrivateCapability(t *testing.T) {
	program := loadProviderRuntimeProgram(t)
	root := mustProviderRoot(
		t,
		program.Roots()[0].Types().Scope().Lookup("ReflectGenericString"),
	)
	options := emit.DefaultOptions()
	options.StandardLibrary = linkedProviderCertificate(t)
	emission, err := emit.CompileWithOptions(program, []emit.Root{root}, options)
	if err != nil {
		t.Fatal(err)
	}
	workingDirectory := t.TempDir()
	artifacts := materializeArtifacts(t, emission, workingDirectory)
	if strings.Contains(artifacts.printed, ".TypeFor<") {
		t.Fatalf("open TypeFor retained an erased TypeScript generic call:\n%s", artifacts.printed)
	}
	if !strings.Contains(
		artifacts.printed,
		"export function ReflectGenericString(): gostring",
	) || !strings.Contains(
		artifacts.printed,
		"export function ReflectGeneric$kernel<T>($go$reflection_type_",
	) {
		t.Fatalf("generic reflection capability crossed its private kernel boundary:\n%s", artifacts.printed)
	}
	assemblyPath := ""
	for _, file := range emission.Files() {
		if file.Kind() == emit.TargetFilePackageAssembly &&
			file.PackageName() == "providerruntime" {
			assemblyPath = file.OutputPath()
			break
		}
	}
	if assemblyPath == "" {
		t.Fatal("provider runtime package assembly is absent")
	}
	targetOutput := executeProviderTypeScript(
		t,
		workingDirectory,
		artifacts.paths,
		assemblyPath,
		[]string{"ReflectGenericString"},
		`console.log(ReflectGenericString());
`,
	)
	if targetOutput != "string\n" {
		t.Fatalf("open generic reflection differential = %q", targetOutput)
	}
}

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

type ReflectedEntry struct { Name string `+"`json:\"name\"`"+` }

func ReflectStatic() (string, string, int) {
	typ := reflect.TypeFor[ReflectedEntry]()
	return typ.Kind().String(), typ.Field(0).Tag.Get("json"), typ.NumField()
}

type Named interface { Name() string }
type Unrelated struct{}
func (Unrelated) Name() string { return "unrelated" }
func useNamed(value Named) bool { return value.Name() == "" }

func ReflectDynamicFacts() (string, bool) {
	var observed any = "value"
	var absent any
	return reflect.TypeOf(observed).Kind().String(), reflect.TypeOf(absent) == nil && !useNamed(Unrelated{})
}

func ReflectGeneric[T any]() string {
	return reflect.TypeFor[T]().Kind().String()
}

func ReflectGenericString() string { return ReflectGeneric[string]() }

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

func readGeneratedArtifact(
	t *testing.T,
	workingDirectory string,
	outputPath string,
) string {
	t.Helper()
	source, err := os.ReadFile(filepath.Join(
		workingDirectory,
		filepath.FromSlash(outputPath),
	))
	if err != nil {
		t.Fatal(err)
	}
	return string(source)
}
