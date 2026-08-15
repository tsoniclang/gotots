package load_test

import (
	"context"
	"go/ast"
	"os"
	"path/filepath"
	"testing"

	"github.com/tsoniclang/gotots/internal/load"
)

func TestSelectedSyntaxHasExactParentRelations(t *testing.T) {
	project := t.TempDir()
	writeSyntaxParentFile(t, filepath.Join(project, "go.mod"), `module example.com/parents

go 1.26.4
`)
	writeSyntaxParentFile(t, filepath.Join(project, "source.go"), `package parents

func Run(left, right int) int { return left + right }
`)
	program, err := load.Load(context.Background(), load.Request{
		Directory: project,
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	source := program.Roots()[0]
	function := source.Files()[0].Syntax().Decls[0].(*ast.FuncDecl)
	result := function.Body.List[0].(*ast.ReturnStmt).Results[0].(*ast.BinaryExpr)
	if parent, ok := source.SyntaxParent(result.X); !ok || parent != result {
		t.Fatalf("left parent = (%T, %v), want exact binary expression", parent, ok)
	}
	if parent, ok := source.SyntaxParent(result.Y); !ok || parent != result {
		t.Fatalf("right parent = (%T, %v), want exact binary expression", parent, ok)
	}
	if _, ok := source.SyntaxParent(ast.NewIdent("detached")); ok {
		t.Fatal("detached syntax acquired a fabricated parent")
	}
}

func writeSyntaxParentFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
