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

func TestProgramRetainsOneCoherentSourceAvailableDependencyGraph(t *testing.T) {
	projectDirectory := filepath.Join("..", "..", "testdata", "projects", "demand-program")
	loaded, err := Load(context.Background(), Request{
		Directory: projectDirectory,
		Pattern:   "./api",
	})
	if err != nil {
		t.Fatal(err)
	}
	packages := loaded.Packages()
	if len(packages) != 3 {
		t.Fatalf("source-available packages = %d, want api, mathx, service", len(packages))
	}
	for index, path := range []string{
		"example.com/demand/api",
		"example.com/demand/mathx",
		"example.com/demand/service",
	} {
		if packages[index].Path() != path {
			t.Fatalf("package %d = %q, want %q", index, packages[index].Path(), path)
		}
		if packages[index].ModulePath() != "example.com/demand" ||
			packages[index].ModuleVersion() != "" {
			t.Fatalf(
				"package %s module = %s@%s",
				path,
				packages[index].ModulePath(),
				packages[index].ModuleVersion(),
			)
		}
	}
	if roots := loaded.Roots(); len(roots) != 1 ||
		roots[0].Path() != "example.com/demand/api" {
		t.Fatalf("roots = %v, want api", roots)
	}

	apiPackage := loaded.PackageByPath("example.com/demand/api")
	servicePackage := loaded.PackageByPath("example.com/demand/service")
	if apiPackage == nil || servicePackage == nil {
		t.Fatal("api or service package is absent")
	}
	var runDeclaration *ast.FuncDecl
	for _, declaration := range apiPackage.Files()[0].Syntax().Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if ok && function.Name.Name == "Run" {
			runDeclaration = function
			break
		}
	}
	if runDeclaration == nil {
		t.Fatal("Run declaration not found")
	}
	result := runDeclaration.Body.List[0].(*ast.ReturnStmt).Results[0].(*ast.BinaryExpr)
	call := result.X.(*ast.CallExpr)
	selector := call.Fun.(*ast.SelectorExpr)
	if apiPackage.TypesInfo().Uses[selector.Sel] !=
		servicePackage.Types().Scope().Lookup("Compute") {
		t.Fatal("cross-package selector does not share the loaded go/types object")
	}
	if loaded.PackageForTypes(servicePackage.Types()) != servicePackage {
		t.Fatal("types.Package reverse lookup did not preserve package identity")
	}
}
