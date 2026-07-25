package emit_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/load"
)

func TestDeclarationDispatcherRejectsUnsupportedNode(t *testing.T) {
	projectDirectory := projectPath("constructs", "declaration", "unsupported", "variable")
	loaded, err := load.One(context.Background(), load.Request{
		Directory: projectDirectory,
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}

	compiler := emit.New(loaded)
	_, err = compiler.EmitFile(loaded.Files()[0].Syntax(), filepath.Join(t.TempDir(), "variable.ts"))
	var unsupported *api.UnsupportedError
	if !errors.As(err, &unsupported) {
		t.Fatalf("error = %v, want *api.UnsupportedError", err)
	}
	if unsupported.Category != api.CategoryDeclaration {
		t.Fatalf("category = %s, want %s", unsupported.Category, api.CategoryDeclaration)
	}
	if unsupported.Construct != "*ast.GenDecl" {
		t.Fatalf("construct = %q, want *ast.GenDecl", unsupported.Construct)
	}
	if unsupported.Role != api.RoleFileDeclaration {
		t.Fatalf("role = %s, want %s", unsupported.Role, api.RoleFileDeclaration)
	}
}

func projectPath(elements ...string) string {
	parts := append([]string{"..", "..", "testdata"}, elements...)
	return filepath.Join(parts...)
}
