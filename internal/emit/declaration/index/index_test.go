package index

import (
	"go/ast"
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
	declaration := &ast.FuncDecl{Name: ast.NewIdent("Run")}
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
