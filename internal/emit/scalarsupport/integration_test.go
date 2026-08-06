package scalarsupport_test

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/output"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestProgramEmitsRequestedPrimitiveAliasesOnce(t *testing.T) {
	program := loadDemandProgram(t)
	roots, err := emit.ExportedAPIRoots(program.Roots()[0])
	if err != nil {
		t.Fatal(err)
	}
	emission, err := emit.Compile(program, roots)
	if err != nil {
		t.Fatal(err)
	}

	var support emit.TargetFile
	for _, file := range emission.Files() {
		if file.Kind() == emit.TargetFileSupport {
			if support.SourceFile() != nil {
				t.Fatal("program emitted more than one scalar support module")
			}
			support = file
		}
	}
	if support.SourceFile() == nil {
		t.Fatal("program emitted no scalar support module")
	}
	if support.OutputPath() != output.ScalarSupportPath {
		t.Fatalf(
			"support path = %q, want %q",
			support.OutputPath(),
			output.ScalarSupportPath,
		)
	}
	runtimePackage, ok := emission.RuntimePackage()
	if !ok {
		t.Fatal("program emitted no canonical runtime package")
	}
	if runtimePackage.Name() != "@gotots/runtime" ||
		runtimePackage.RootPath() != "runtime" ||
		runtimePackage.ManifestPath() != "runtime/package.json" ||
		len(runtimePackage.Manifest()) == 0 {
		t.Fatalf(
			"runtime package = %q/%q/%q manifest=%d",
			runtimePackage.Name(),
			runtimePackage.RootPath(),
			runtimePackage.ManifestPath(),
			len(runtimePackage.Manifest()),
		)
	}
	statements := support.SourceFile().Statements()
	if len(statements) != 1 {
		t.Fatalf("support statements = %d, want one requested alias", len(statements))
	}
	alias, ok := statements[0].(tsgo.TypeAliasDeclaration)
	if !ok {
		t.Fatalf("support statement = %T, want tsgo.TypeAliasDeclaration", statements[0])
	}
	if alias.Name().Text() != "int32" ||
		alias.Type().Kind() != tsgo.SyntaxKindNumberKeyword {
		t.Fatalf(
			"support alias = %s/%d, want int32/number",
			alias.Name().Text(),
			alias.Type().Kind(),
		)
	}
}

func TestCompileFileReturnsCompleteStandaloneEmission(t *testing.T) {
	program := loadDemandProgram(t)
	root := program.Roots()[0]
	emission, err := emit.CompileFile(root, root.Files()[0].Syntax())
	if err != nil {
		t.Fatal(err)
	}
	packages := make(map[string]struct{})
	initialization := 0
	support := 0
	for _, file := range emission.Files() {
		switch file.Kind() {
		case emit.TargetFileSource,
			emit.TargetFilePackageState,
			emit.TargetFilePackageAssembly:
			packages[file.PackageName()] = struct{}{}
		case emit.TargetFileSupport:
			support++
			if file.OutputPath() != output.ScalarSupportPath {
				t.Fatalf("support path = %q", file.OutputPath())
			}
		case emit.TargetFileProgramInitialization:
			initialization++
			if file.OutputPath() != output.ProgramInitializationPath {
				t.Fatalf("program initialization path = %q", file.OutputPath())
			}
		default:
			t.Fatalf("target file %s has invalid kind %d", file.OutputPath(), file.Kind())
		}
	}
	for _, packageName := range []string{"api", "mathx", "service"} {
		if _, ok := packages[packageName]; !ok {
			t.Fatalf("file-root emission dropped dependency package %s", packageName)
		}
	}
	if len(packages) != 3 || initialization != 1 || support != 1 {
		t.Fatalf(
			"complete files = packages %v, program %d, support %d",
			packages,
			initialization,
			support,
		)
	}
}

func TestIntegerRepresentationDefaultsToDirectNumberSyntax(t *testing.T) {
	program := loadDemandProgram(t)
	roots, err := emit.ExportedAPIRoots(program.Roots()[0])
	if err != nil {
		t.Fatal(err)
	}
	emission, err := emit.Compile(program, roots)
	if err != nil {
		t.Fatal(err)
	}
	assertIntegerCarrier(t, emission, false)
	assertEmissionHasNoIntegerNoise(t, emission)
}

func TestBigIntProfilePreservesNarrowIntegerCarriers(t *testing.T) {
	program := loadDemandProgram(t)
	roots, err := emit.ExportedAPIRoots(program.Roots()[0])
	if err != nil {
		t.Fatal(err)
	}
	emission, err := emit.CompileWithOptions(program, roots, emit.Options{
		IntegerRepresentation: emit.IntegerRepresentationBigInt,
		EvaluationOrder:       emit.EvaluationOrderDirect,
	})
	if err != nil {
		t.Fatal(err)
	}
	assertIntegerCarrier(t, emission, true)
	printed := printIntegerEmission(t, emission)
	if strings.Contains(printed, "0n") || strings.Contains(printed, "1n") {
		t.Fatalf("narrow-only BigInt-profile emission contains BigInt syntax:\n%s", printed)
	}
	assertNoIntegerNoise(t, printed)
}

func TestInvalidIntegerRepresentationFailsAtCompilationEntry(t *testing.T) {
	program := loadDemandProgram(t)
	roots, err := emit.ExportedAPIRoots(program.Roots()[0])
	if err != nil {
		t.Fatal(err)
	}
	_, err = emit.CompileWithOptions(program, roots, emit.Options{})
	var optionsError *emit.OptionsError
	if !errors.As(err, &optionsError) {
		t.Fatalf("error = %#v, want *emit.OptionsError", err)
	}
}

func TestInvalidEvaluationOrderFailsAtCompilationEntry(t *testing.T) {
	program := loadDemandProgram(t)
	roots, err := emit.ExportedAPIRoots(program.Roots()[0])
	if err != nil {
		t.Fatal(err)
	}
	options := emit.DefaultOptions()
	options.EvaluationOrder = emit.EvaluationOrderInvalid
	_, err = emit.CompileWithOptions(program, roots, options)
	var optionsError *emit.OptionsError
	if !errors.As(err, &optionsError) ||
		optionsError.Field != "evaluation order" {
		t.Fatalf("error = %#v, want evaluation-order OptionsError", err)
	}
}

func TestIntegerRepresentationSelectionParsesOnlyClosedProfiles(t *testing.T) {
	for source, want := range map[string]emit.IntegerRepresentation{
		"number":         emit.IntegerRepresentationNumber,
		"fixed64-bigint": emit.IntegerRepresentationFixed64BigInt,
		"bigint":         emit.IntegerRepresentationBigInt,
	} {
		got, err := emit.ParseIntegerRepresentation(source)
		if err != nil {
			t.Fatalf("parse %q: %v", source, err)
		}
		if got != want {
			t.Fatalf("parse %q = %v, want %v", source, got, want)
		}
	}
	got, err := emit.ParseIntegerRepresentation("int32")
	var optionsError *emit.OptionsError
	if got != emit.IntegerRepresentationInvalid ||
		!errors.As(err, &optionsError) {
		t.Fatalf("parse invalid = (%v, %#v), want invalid OptionsError", got, err)
	}
}

func TestEvaluationOrderDefaultsToDirectEmission(t *testing.T) {
	if got := emit.DefaultOptions().EvaluationOrder; got != emit.EvaluationOrderDirect {
		t.Fatalf("default evaluation order = %v, want direct", got)
	}
}

func TestEvaluationOrderSelectionParsesOnlyClosedProfiles(t *testing.T) {
	for source, want := range map[string]emit.EvaluationOrder{
		"direct":      emit.EvaluationOrderDirect,
		"preserve-go": emit.EvaluationOrderPreserveGo,
	} {
		got, err := emit.ParseEvaluationOrder(source)
		if err != nil {
			t.Fatalf("parse %q: %v", source, err)
		}
		if got != want {
			t.Fatalf("parse %q = %v, want %v", source, got, want)
		}
	}
	got, err := emit.ParseEvaluationOrder("automatic")
	var optionsError *emit.OptionsError
	if got != emit.EvaluationOrderInvalid ||
		!errors.As(err, &optionsError) {
		t.Fatalf("parse invalid = (%v, %#v), want invalid OptionsError", got, err)
	}
}

func assertIntegerCarrier(
	t *testing.T,
	emission emit.ProgramEmission,
	exact bool,
) {
	t.Helper()
	found := false
	for _, file := range emission.Files() {
		if file.Kind() != emit.TargetFileSupport {
			continue
		}
		for _, statement := range file.SourceFile().Statements() {
			alias, ok := statement.(tsgo.TypeAliasDeclaration)
			if !ok || alias.Name().Text() == "bool" {
				continue
			}
			found = true
			want := tsgo.SyntaxKindNumberKeyword
			if exact && (alias.Name().Text() == "int64" ||
				alias.Name().Text() == "uint64") {
				want = tsgo.SyntaxKindBigIntKeyword
			}
			if alias.Type().Kind() != want {
				t.Fatalf("%s carrier = %d, want %d", alias.Name().Text(), alias.Type().Kind(), want)
			}
		}
	}
	if !found {
		t.Fatal("emission contains no integer support alias")
	}
}

func assertEmissionHasNoIntegerNoise(t *testing.T, emission emit.ProgramEmission) {
	t.Helper()
	assertNoIntegerNoise(t, printIntegerEmission(t, emission))
}

func assertNoIntegerNoise(t *testing.T, printed string) {
	t.Helper()
	for _, forbidden := range []string{
		" as int32",
		" as int64",
		"Math.imul",
		") | 0",
	} {
		if strings.Contains(printed, forbidden) {
			t.Fatalf("generated TypeScript contains %q:\n%s", forbidden, printed)
		}
	}
}

func printIntegerEmission(t *testing.T, emission emit.ProgramEmission) string {
	t.Helper()
	client, err := tsgo.StartClient(repositoryRoot(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("close TS-Go client: %v", err)
		}
	})
	var result strings.Builder
	for _, file := range emission.Files() {
		if file.Kind() != emit.TargetFileSource {
			continue
		}
		printed, err := client.PrintNode(file.SourceFile(), tsgo.PrintOptions{})
		if err != nil {
			t.Fatal(err)
		}
		result.WriteString(printed)
	}
	return result.String()
}

func loadDemandProgram(t *testing.T) *load.Program {
	t.Helper()
	program, err := load.Load(context.Background(), load.Request{
		Directory: filepath.Join(
			repositoryRoot(),
			"testdata",
			"projects",
			"demand-program",
		),
		Pattern: "./api",
	})
	if err != nil {
		t.Fatal(err)
	}
	return program
}

func repositoryRoot() string {
	return filepath.Join("..", "..", "..")
}
