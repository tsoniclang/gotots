package emit

import (
	"go/ast"
	"go/token"
	"go/types"
	"testing"

	"github.com/tsoniclang/gotots/internal/load"
)

func TestSchedulerDeduplicatesPendingAndCycleReferences(t *testing.T) {
	sourcePackage := types.NewPackage("example.com/schedule", "schedule")
	first := types.NewFunc(
		token.Pos(1),
		sourcePackage,
		"First",
		types.NewSignatureType(nil, nil, nil, nil, nil, false),
	)
	second := types.NewFunc(
		token.Pos(2),
		sourcePackage,
		"Second",
		types.NewSignatureType(nil, nil, nil, nil, nil, false),
	)
	scheduler := newScheduler()
	scheduler.enqueue(first)
	scheduler.enqueue(first)
	if object, ok := scheduler.next(); !ok || object != first {
		t.Fatalf("first scheduled object = %v, %v", object, ok)
	}
	scheduler.enqueue(second)
	scheduler.enqueue(first)
	if object, ok := scheduler.next(); !ok || object != second {
		t.Fatalf("second scheduled object = %v, %v", object, ok)
	}
	if object, ok := scheduler.next(); ok || object != nil {
		t.Fatalf("duplicate cycle target was re-enqueued: %v", object)
	}
}

func TestDeclarationIndexRejectsDuplicateObjectOwnership(t *testing.T) {
	sourcePackage := types.NewPackage("example.com/schedule", "schedule")
	object := types.NewFunc(
		token.Pos(1),
		sourcePackage,
		"Run",
		types.NewSignatureType(nil, nil, nil, nil, nil, false),
	)
	sites := make(map[types.Object]declarationSite)
	declaration := &ast.FuncDecl{Name: ast.NewIdent("Run")}
	if err := addDeclarationSite(
		sites,
		object,
		nil,
		load.File{},
		declaration,
		"modules/key/schedule/run.ts",
	); err != nil {
		t.Fatal(err)
	}
	if err := addDeclarationSite(
		sites,
		object,
		nil,
		load.File{},
		declaration,
		"modules/key/schedule/other.ts",
	); err == nil {
		t.Fatal("duplicate declaration ownership was accepted")
	}
}
