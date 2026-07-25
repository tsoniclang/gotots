package load

import (
	"context"
	"go/ast"
	"go/types"
	"path/filepath"
	"testing"
)

func TestOneReturnsOneCoherentSyntaxAndTypeGraph(t *testing.T) {
	projectDirectory := filepath.Join("..", "..", "testdata", "projects", "single-package")
	loaded, err := One(context.Background(), Request{
		Directory: projectDirectory,
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Path() != "example.com/add" {
		t.Fatalf("package path = %q, want example.com/add", loaded.Path())
	}

	files := loaded.Files()
	if len(files) != 1 {
		t.Fatalf("syntax files = %d, want 1", len(files))
	}
	if filepath.Base(files[0].Path()) != "add.go" {
		t.Fatalf("source path = %q, want add.go", files[0].Path())
	}
	function, ok := files[0].Syntax().Decls[0].(*ast.FuncDecl)
	if !ok {
		t.Fatalf("first declaration = %T, want *ast.FuncDecl", files[0].Syntax().Decls[0])
	}
	signature, ok := loaded.TypesInfo().Defs[function.Name].Type().(*types.Signature)
	if !ok {
		t.Fatalf("definition for %s is not a function signature", function.Name.Name)
	}
	binary := function.Body.List[0].(*ast.ReturnStmt).Results[0].(*ast.BinaryExpr)
	leftObject := loaded.TypesInfo().Uses[binary.X.(*ast.Ident)]
	rightObject := loaded.TypesInfo().Uses[binary.Y.(*ast.Ident)]
	if leftObject != signature.Params().At(0) || rightObject != signature.Params().At(1) {
		t.Fatal("parameter definitions and body uses do not share one go/types graph")
	}
	if width := loaded.TypesSizes().Sizeof(types.Typ[types.Int]); width != 4 && width != 8 {
		t.Fatalf("int width = %d bytes, want 4 or 8", width)
	}
}

func TestOneFailsClosedOnTypeErrors(t *testing.T) {
	projectDirectory := filepath.Join("..", "..", "testdata", "projects", "type-error")
	_, err := One(context.Background(), Request{
		Directory: projectDirectory,
		Pattern:   ".",
	})
	if err == nil {
		t.Fatal("load succeeded for a package with a type error")
	}
	if _, ok := err.(*Error); !ok {
		t.Fatalf("error = %T, want *load.Error", err)
	}
}
