package emit_test

import (
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
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
	support := 0
	for _, file := range emission.Files() {
		switch file.Kind() {
		case emit.TargetFileSource:
			packages[file.PackageName()] = struct{}{}
		case emit.TargetFileSupport:
			support++
			if file.OutputPath() != output.ScalarSupportPath {
				t.Fatalf("support path = %q", file.OutputPath())
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
	if len(packages) != 3 || support != 1 {
		t.Fatalf("complete files = packages %v, support %d", packages, support)
	}
}
