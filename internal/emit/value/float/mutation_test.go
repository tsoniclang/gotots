package float_test

import (
	"context"
	"go/ast"
	"go/token"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

// TestFloatConstantSpellingIsIgnored proves float materialization reads the
// checker's canonical go/constant value, not the source spelling: rewriting a
// float literal to a different spelling of the same value (2.5 -> 2.50e0) leaves
// the emitted artifact byte-identical.
func TestFloatConstantSpellingIsIgnored(t *testing.T) {
	baseline := printFloatProgram(t, loadFloat(t))

	mutated := loadFloat(t)
	literalNode := floatLiteral(t, mutated, "Literal")
	literalNode.Value = "2.50e0"
	after := printFloatProgram(t, mutated)

	if baseline != after {
		t.Fatalf("artifact changed after an alternate float spelling (same checker value):\n%s\n---\n%s", baseline, after)
	}
}

func loadFloat(t *testing.T) *load.Package {
	t.Helper()
	loaded, err := load.One(context.Background(), load.Request{Directory: floatDirectory(), Pattern: "."})
	if err != nil {
		t.Fatal(err)
	}
	return loaded
}

func floatLiteral(t *testing.T, loaded *load.Package, function string) *ast.BasicLit {
	t.Helper()
	for _, file := range loaded.Files() {
		for _, declaration := range file.Syntax().Decls {
			decl, ok := declaration.(*ast.FuncDecl)
			if !ok || decl.Name.Name != function {
				continue
			}
			literal, ok := decl.Body.List[0].(*ast.ReturnStmt).Results[0].(*ast.BasicLit)
			if ok && literal.Kind == token.FLOAT {
				return literal
			}
		}
	}
	t.Fatalf("no float literal in function %s", function)
	return nil
}

func printFloatProgram(t *testing.T, loaded *load.Package) string {
	t.Helper()
	roots, err := emit.ExportedAPIRoots(loaded)
	if err != nil {
		t.Fatal(err)
	}
	emission, err := emit.Compile(loaded.Program(), roots)
	if err != nil {
		t.Fatal(err)
	}
	client, err := tsgo.StartClient(repositoryRoot(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = client.Close() })
	result := ""
	for _, file := range emission.Files() {
		printed, err := client.PrintNode(file.SourceFile(), tsgo.PrintOptions{})
		if err != nil {
			t.Fatal(err)
		}
		result += printed
	}
	return result
}
