package index

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"github.com/tsoniclang/gotots/internal/load"
)

func TestAddSiteRejectsDuplicateObjectOwnership(t *testing.T) {
	sourcePackage := types.NewPackage("example.com/index", "index")
	object := types.NewFunc(
		token.Pos(1),
		sourcePackage,
		"Run",
		types.NewSignatureType(nil, nil, nil, nil, nil, false),
	)
	sites := make(map[types.Object]Site)
	parsed, err := parser.ParseFile(token.NewFileSet(), "source.go", "package index\nfunc Run() {}\n", 0)
	if err != nil {
		t.Fatal(err)
	}
	declaration, ok := parsed.Decls[0].(*ast.FuncDecl)
	if !ok {
		t.Fatalf("declaration = %T", parsed.Decls[0])
	}
	if err := addSite(
		sites,
		object,
		nil,
		load.File{},
		declaration,
		"modules/key/index/run.ts",
	); err != nil {
		t.Fatal(err)
	}
	if err := addSite(
		sites,
		object,
		nil,
		load.File{},
		declaration,
		"modules/key/index/other.ts",
	); err == nil {
		t.Fatal("duplicate declaration ownership was accepted")
	}
}
