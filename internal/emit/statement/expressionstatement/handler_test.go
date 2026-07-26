package expressionstatement_test

import (
	"context"
	"errors"
	"go/ast"
	"path/filepath"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/load"
)

func TestExpressionStatementRejectsNonCallMutation(t *testing.T) {
	project := filepath.Join(
		"..", "..", "..", "..",
		"testdata", "projects", "void-calls",
	)
	loaded, err := load.One(context.Background(), load.Request{
		Directory: project,
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	runFunction := loaded.Files()[0].Syntax().Decls[2].(*ast.FuncDecl)
	statement := runFunction.Body.List[0].(*ast.ExprStmt)
	statement.X = &ast.Ident{Name: "notACall"}

	_, err = emit.CompileFile(loaded, loaded.Files()[0].Syntax())
	var unsupported *api.UnsupportedError
	if !errors.As(err, &unsupported) {
		t.Fatalf("error = %v, want *api.UnsupportedError", err)
	}
	if unsupported.Category != api.CategoryStatement ||
		unsupported.Construct != "*ast.ExprStmt" ||
		unsupported.Role != api.RoleBlockStatement {
		t.Fatalf("unsupported error = %#v", unsupported)
	}
}
