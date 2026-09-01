package localdeclaration_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/output"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestInitializedLocalDeclarationBuildsInferredVariableList(t *testing.T) {
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
	source := loaded.Files()[0]
	emission, err := emit.CompileFile(loaded, source.Syntax())
	if err != nil {
		t.Fatal(err)
	}
	expectedPath, err := output.SourcePath(loaded, source)
	if err != nil {
		t.Fatal(err)
	}
	var target tsgo.SourceFile
	for _, file := range emission.Files() {
		if file.Kind() == emit.TargetFileSource && file.OutputPath() == expectedPath {
			target = file.SourceFile()
			break
		}
	}
	if target == nil {
		t.Fatalf("complete emission has no source artifact %s", expectedPath)
	}
	compute := localDeclarationFunctionByName(t, target, "Compute")
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
	if declarations[0].Type() != nil || declarations[1].Type() != nil ||
		declarations[0].Initializer() == nil || declarations[1].Initializer() == nil {
		t.Fatal("initialized declarations were not left to exact target inference")
	}
}

func localDeclarationFunctionByName(
	t *testing.T,
	source tsgo.SourceFile,
	name string,
) tsgo.FunctionDeclaration {
	t.Helper()
	for _, statement := range source.Statements() {
		function, ok := statement.(tsgo.FunctionDeclaration)
		if ok && function.Name().Text() == name {
			return function
		}
	}
	t.Fatalf("target function %q is absent", name)
	return nil
}
