package emit_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestPackageInitializerArtifactsOwnDistinctLexicalScopes(t *testing.T) {
	projectDirectory := filepath.Join(
		repositoryRoot(),
		"testdata",
		"projects",
		"package-initialization",
	)
	program, err := load.Load(context.Background(), load.Request{
		Directory: projectDirectory,
		Pattern:   "./api",
	})
	if err != nil {
		t.Fatal(err)
	}
	roots, err := emit.ExportedAPIRoots(program.Roots()[0])
	if err != nil {
		t.Fatal(err)
	}
	emission, err := emit.Compile(program, roots)
	if err != nil {
		t.Fatal(err)
	}
	wantBlocks := map[string]int{"api": 1, "sideeffect": 2, "sink": 0}
	seen := make(map[string]bool)
	for _, file := range emission.Files() {
		if file.Kind() != emit.TargetFilePackageAssembly {
			continue
		}
		for _, statement := range file.SourceFile().Statements() {
			function, ok := statement.(tsgo.FunctionDeclaration)
			if !ok || function.Name().Text() != "$initialize" {
				continue
			}
			blocks := 0
			for _, bodyStatement := range function.Body().(tsgo.Block).Statements() {
				if _, ok := bodyStatement.(tsgo.Block); ok {
					blocks++
				}
			}
			if blocks != wantBlocks[file.PackageName()] {
				t.Fatalf(
					"%s initializer artifact scopes = %d, want %d",
					file.PackageName(),
					blocks,
					wantBlocks[file.PackageName()],
				)
			}
			seen[file.PackageName()] = true
		}
	}
	if len(seen) != len(wantBlocks) {
		t.Fatalf("initializer assemblies = %v, want %v", seen, wantBlocks)
	}
}
