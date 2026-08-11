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
	rootSource := readProviderRuntimeRootSource(t, emission, workingDirectory)
	for _, required := range []string{
		".Field(BigInt.asIntN(64, goNumberToBigInt(",
		"globalThis.Number(BigInt.asIntN(64, goInterfaceNonNil<",
		").NumField()))",
	} {
		if !strings.Contains(rootSource, required) {
			t.Fatalf(
				"number-profile interface call lacks scalar boundary %q:\n%s",
				required,
				rootSource,
			)
		}
	}
	reflectionSource := readGeneratedArtifact(
		t,
		workingDirectory,
		output.ReflectionTypeSupportPath,
	)
	for _, required := range []string{".$create(() => ({"} {
		if !strings.Contains(reflectionSource, required) {
			t.Fatalf(
				"reflection metadata lacks lazy constructor %q:\n%s",
				required,
				reflectionSource,
			)
		}
	}
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
	typecheckProviderRunner(
		t,
		workingDirectory,
		artifacts.paths,
		assemblyPath,
		[]string{"ReflectStatic"},
		`const [kind, tag, fields] = ReflectStatic();
console.log(kind + "|" + tag + "|" + fields);
`,
	)
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
	typecheckProviderRunner(
		t,
		workingDirectory,
		artifacts.paths,
		assemblyPath,
		[]string{"ReflectDynamicFacts"},
		`const [kind, absent] = ReflectDynamicFacts();
console.log(kind + "|" + absent);
`,
	)
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
	typecheckProviderRunner(
		t,
		workingDirectory,
		artifacts.paths,
		assemblyPath,
		[]string{"ReflectGenericString"},
		`console.log(ReflectGenericString());
`,
	)
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

func TestSelectedProviderSupportsCertifiedBigIntProfile(t *testing.T) {
	program := loadProviderRuntimeProgram(t)
	root := mustProviderRoot(
		t,
		program.Roots()[0].Types().Scope().Lookup("ReflectStatic"),
	)
	options := emit.DefaultOptions()
	options.IntegerRepresentation = emit.IntegerRepresentationBigInt
	options.StandardLibrary = linkedProviderCertificate(t)
	emission, err := emit.CompileWithOptions(program, []emit.Root{root}, options)
	if err != nil {
		t.Fatal(err)
	}
	workingDirectory := t.TempDir()
	artifacts := materializeArtifacts(t, emission, workingDirectory)
	rootSource := readProviderRuntimeRootSource(t, emission, workingDirectory)
	for _, redundant := range []string{"goNumberToBigInt", "globalThis.Number("} {
		if strings.Contains(rootSource, redundant) {
			t.Fatalf(
				"bigint-profile interface call retained %q conversion:\n%s",
				redundant,
				rootSource,
			)
		}
	}
	for _, required := range []string{
		".Field(__gotots_argument_0)",
		".NumField()",
	} {
		if !strings.Contains(rootSource, required) {
			t.Fatalf(
				"bigint-profile interface call lacks direct shape %q:\n%s",
				required,
				rootSource,
			)
		}
	}
	if !strings.Contains(artifacts.printed, "export type int = bigint;") ||
		!strings.Contains(artifacts.printed, "export type int64 = bigint;") {
		t.Fatalf("bigint provider profile lacks exact scalar aliases:\n%s", artifacts.printed)
	}
	reflectionSource := readGeneratedArtifact(
		t,
		workingDirectory,
		output.ReflectionTypeSupportPath,
	)
	if !strings.Contains(reflectionSource, "kind: 25n") ||
		!strings.Contains(reflectionSource, "size: 16n") ||
		!strings.Contains(reflectionSource, "align: 8n") {
		t.Fatalf("bigint reflection metadata lacks provider scalar ABI:\n%s", reflectionSource)
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
		t.Fatal("bigint provider runtime package assembly is absent")
	}
	typecheckProviderRunner(
		t,
		workingDirectory,
		artifacts.paths,
		assemblyPath,
		[]string{"ReflectStatic"},
		`const [kind, tag, fields] = ReflectStatic();
console.log(kind + "|" + tag + "|" + fields);
`,
	)
}

func TestProviderStructFieldsProjectEveryValueOperation(t *testing.T) {
	program := loadProviderRuntimeProgram(t)
	root := mustProviderRoot(
		t,
		program.Roots()[0].Types().Scope().Lookup("ProviderStructFields"),
	)
	options := emit.DefaultOptions()
	options.StandardLibrary = linkedProviderCertificate(t)
	emission, err := emit.CompileWithOptions(program, []emit.Root{root}, options)
	if err != nil {
		t.Fatal(err)
	}
	workingDirectory := t.TempDir()
	artifacts := materializeArtifacts(t, emission, workingDirectory)
	rootSource := readProviderRuntimeRootSource(t, emission, workingDirectory)
	for _, required := range []string{
		"UnicodeRangeTableOperations.$make(",
		"BigInt.asIntN(64, goNumberToBigInt(5))",
		"BigInt.asUintN(64, goNumberToBigInt($productValue))",
		"globalThis.Number(BigInt.asUintN(64,",
		"projectPointer<",
	} {
		if !strings.Contains(rootSource, required) &&
			!strings.Contains(artifacts.printed, required) {
			t.Fatalf("provider struct projection lacks %q:\n%s", required, artifacts.printed)
		}
	}
	for _, forbidden := range []string{
		"RuntimeSliceProjection<metrics__from_gostdlib.Description",
		"RuntimeMetricsDescriptionOperations.$zero()",
		"RuntimeSliceProjection<unicode__from_gostdlib.Range16",
		"RuntimeSliceProjection<unicode__from_gostdlib.Range32",
	} {
		if strings.Contains(rootSource, forbidden) {
			t.Fatalf("equal provider struct representation emitted %q:\n%s", forbidden, rootSource)
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
	typecheckProviderRunner(
		t,
		workingDirectory,
		artifacts.paths,
		assemblyPath,
		[]string{"ProviderStructFields"},
		`const [allocation, latinOffset, name] = ProviderStructFields();
console.log(allocation + "|" + latinOffset + "|" + name);
`,
	)
}

func TestProviderVariadicSliceProjectsDirectNamedPointers(t *testing.T) {
	program := loadProviderRuntimeProgram(t)
	root := mustProviderRoot(
		t,
		program.Roots()[0].Types().Scope().Lookup("UnicodeDecimal"),
	)
	options := emit.DefaultOptions()
	options.StandardLibrary = linkedProviderCertificate(t)
	emission, err := emit.CompileWithOptions(program, []emit.Root{root}, options)
	if err != nil {
		t.Fatal(err)
	}
	workingDirectory := t.TempDir()
	artifacts := materializeArtifacts(t, emission, workingDirectory)
	rootSource := readProviderRuntimeRootSource(t, emission, workingDirectory)
	if !strings.Contains(
		rootSource,
		"RuntimeSliceProjection<Pointer<unicode__from_gostdlib.RangeTable> | undefined, unicode__from_gostdlib.RangeTable | undefined>",
	) {
		t.Fatalf("provider named-pointer slice element retained the product pointer carrier:\n%s", rootSource)
	}
	waveThreeTypecheck(t, workingDirectory, artifacts.paths)
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

import (
	"reflect"
	"runtime"
	"runtime/metrics"
	"unicode"
)

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

func ProviderStructFields() (uint64, int, string) {
	var stats runtime.MemStats
	stats.Alloc = 7
	allocation := &stats.Alloc
	*allocation = 9
	table := unicode.RangeTable{LatinOffset: 5}
	descriptions := metrics.All()
	descriptions[0].Name = "changed"
	return stats.Alloc, table.LatinOffset, descriptions[0].Name
}

func UnicodeDecimal(value rune) bool {
	return unicode.In(value, unicode.Nd)
}
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

func readProviderRuntimeRootSource(
	t *testing.T,
	emission emit.ProgramEmission,
	workingDirectory string,
) string {
	t.Helper()
	for _, file := range emission.Files() {
		if strings.HasSuffix(file.OutputPath(), "/_root/source.ts") {
			return readGeneratedArtifact(
				t,
				workingDirectory,
				file.OutputPath(),
			)
		}
	}
	t.Fatal("provider runtime root source is absent")
	return ""
}
