package localdeclaration_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestInitializedLocalDeclarationBuildsTypedVariableList(t *testing.T) {
	projectDirectory, err := filepath.Abs(
		filepath.Join("..", "..", "..", "..", "testdata", "projects", "local-variables"),
	)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := load.One(context.Background(), load.Request{
		Directory: projectDirectory,
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	target, err := emit.New(loaded).EmitFile(
		loaded.Files()[0].Syntax(),
		filepath.Join(t.TempDir(), "local-variables.ts"),
	)
	if err != nil {
		t.Fatal(err)
	}
	compute := target.Statements()[1].(tsgo.FunctionDeclaration)
	inner := compute.Body().(tsgo.Block).Statements()[1].(tsgo.Block)
	pair := inner.Statements()[1].(tsgo.VariableStatement)
	if pair.DeclarationList().Flags() != tsgo.NodeFlagsLet {
		t.Fatalf("declaration flags = %d, want let", pair.DeclarationList().Flags())
	}
	declarations := pair.DeclarationList().Declarations()
	if len(declarations) != 2 {
		t.Fatalf("declarations = %d, want 2", len(declarations))
	}
	if declarations[0].Name().(tsgo.Identifier).Text() != "left" ||
		declarations[1].Name().(tsgo.Identifier).Text() != "right" {
		t.Fatal("ValueSpec declaration order was not preserved")
	}
	if declarations[0].Type() == nil || declarations[1].Type() == nil ||
		declarations[0].Initializer() == nil || declarations[1].Initializer() == nil {
		t.Fatal("typed initialized declarations were not constructed")
	}
}
