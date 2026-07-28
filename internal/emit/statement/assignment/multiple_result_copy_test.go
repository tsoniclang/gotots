package assignment_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestMultipleResultDestinationsUseTheRHSOwnershipEvidence(t *testing.T) {
	directory := t.TempDir()
	writeFile(
		t,
		filepath.Join(directory, "go.mod"),
		"module example.com/resultcopy\n\ngo 1.26.4\n",
	)
	writeFile(t, filepath.Join(directory, "source.go"), `package resultcopy

type Box struct { Value int32 }

func Pair(value int32) (Box, bool) {
	return Box{Value: value}, true
}

func Use(value int32) int32 {
	box, ok := Pair(value)
	if !ok {
		return 0
	}
	return box.Value
}
`)
	loaded, err := load.One(context.Background(), load.Request{
		Directory: directory,
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	roots, err := emit.ExportedAPIRoots(loaded)
	if err != nil {
		t.Fatal(err)
	}
	emission, err := emit.Compile(loaded.Program(), roots)
	if err != nil {
		t.Fatal(err)
	}
	var use tsgo.FunctionDeclaration
	for _, file := range emission.Files() {
		if file.Kind() != emit.TargetFileSource {
			continue
		}
		for _, statement := range file.SourceFile().Statements() {
			function, ok := statement.(tsgo.FunctionDeclaration)
			if ok && function.Name().Text() == "Use" {
				use = function
			}
		}
	}
	if use == nil {
		t.Fatal("Use target function is absent")
	}
	statements := use.Body().(tsgo.Block).Statements()
	if len(statements) < 2 {
		t.Fatalf("Use statements = %d, want tuple capture and declaration", len(statements))
	}
	declaration := statements[1].(tsgo.VariableStatement).
		DeclarationList().
		Declarations()[0]
	element, ok := declaration.Initializer().(tsgo.ElementAccessExpression)
	if !ok {
		t.Fatalf(
			"owned tuple destination initializer = %T, want direct element",
			declaration.Initializer(),
		)
	}
	if _, ok := element.Expression().(tsgo.Identifier); !ok {
		t.Fatalf("owned tuple source = %T, want captured identifier", element.Expression())
	}
}
