package scalarsupport_test

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/emit/api"
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
	sourceFacts := 0
	for _, file := range emission.Files() {
		if file.Kind() != emit.TargetFileSupport {
			continue
		}
		switch file.OutputPath() {
		case output.ScalarSupportPath:
			if support.SourceFile() != nil {
				t.Fatal("program emitted more than one scalar support module")
			}
			support = file
		case sourceFactRuntimePath(t):
			sourceFacts++
		default:
			t.Fatalf("unexpected support path %q", file.OutputPath())
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
	if sourceFacts != 1 {
		t.Fatalf("source-fact support modules = %d, want one", sourceFacts)
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
	aliases := make([]tsgo.TypeAliasDeclaration, 0, 1)
	for _, statement := range statements {
		if alias, ok := statement.(tsgo.TypeAliasDeclaration); ok {
			aliases = append(aliases, alias)
		}
	}
	if len(aliases) != 1 {
		t.Fatalf("support aliases = %d, want one requested alias", len(aliases))
	}
	alias := aliases[0]
	if alias.Name().Text() != "int32" ||
		alias.Type().Kind() != tsgo.SyntaxKindTypeReference {
		t.Fatalf(
			"support alias = %s/%d, want int32/shared-reference",
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
	sourceFacts := 0
	for _, file := range emission.Files() {
		switch file.Kind() {
		case emit.TargetFileSource,
			emit.TargetFilePackageState,
			emit.TargetFilePackageAssembly:
			packages[file.PackageName()] = struct{}{}
		case emit.TargetFileSupport:
			switch file.OutputPath() {
			case output.ScalarSupportPath:
				support++
			case sourceFactRuntimePath(t):
				sourceFacts++
			default:
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
	if len(packages) != 3 || initialization != 1 || support != 1 || sourceFacts != 1 {
		t.Fatalf(
			"complete files = packages %v, program %d, scalar support %d, source facts %d",
			packages,
			initialization,
			support,
			sourceFacts,
		)
	}
}

func TestIntegerRepresentationDefaultsToLosslessProfileWithNarrowCarriers(t *testing.T) {
	if got := emit.DefaultOptions().IntegerRepresentation; got != emit.IntegerRepresentationBigInt {
		t.Fatalf("default integer representation = %v, want bigint", got)
	}
	program := loadDemandProgram(t)
	roots, err := emit.ExportedAPIRoots(program.Roots()[0])
	if err != nil {
		t.Fatal(err)
	}
	emission, err := emit.Compile(program, roots)
	if err != nil {
		t.Fatal(err)
	}
	assertIntegerCarrier(t, emission)
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
	assertIntegerCarrier(t, emission)
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

func TestEvaluationOrderDefaultsToLosslessEmission(t *testing.T) {
	if got := emit.DefaultOptions().EvaluationOrder; got != emit.EvaluationOrderPreserveGo {
		t.Fatalf("default evaluation order = %v, want preserve-go", got)
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
) {
	t.Helper()
	found := false
	for _, file := range emission.Files() {
		if file.Kind() != emit.TargetFileSupport ||
			file.OutputPath() != output.ScalarSupportPath {
			continue
		}
		for _, statement := range file.SourceFile().Statements() {
			alias, ok := statement.(tsgo.TypeAliasDeclaration)
			if !ok || alias.Name().Text() == "bool" {
				continue
			}
			found = true
			reference, ok := alias.Type().(tsgo.TypeReferenceNode)
			if !ok {
				t.Fatalf("%s carrier = %T, want shared primitive reference", alias.Name().Text(), alias.Type())
			}
			name, ok := reference.TypeName().(tsgo.Identifier)
			want := "$go$core$" + alias.Name().Text()
			if !ok || name.Text() != want {
				t.Fatalf("%s carrier = %T %v, want %s", alias.Name().Text(), reference.TypeName(), reference.TypeName(), want)
			}
		}
	}
	if !found {
		t.Fatal("emission contains no integer support alias")
	}
}

func sourceFactRuntimePath(t *testing.T) string {
	t.Helper()
	contract, err := api.RuntimeContract(api.RuntimeSourceCompilationFact)
	if err != nil {
		t.Fatal(err)
	}
	return contract.OutputPath()
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
