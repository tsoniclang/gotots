package tsgo

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestProjectSourceFileUsesOfficialASTForPrinting(t *testing.T) {
	directory := t.TempDir()
	sourcePath := filepath.Join(directory, "implementation.ts")
	writeProjectFile(t, sourcePath, `export function twice(value: number): number {
  return value * 2;
}
`)
	writeProjectFile(t, filepath.Join(directory, "tsconfig.json"), `{
  "compilerOptions": {
    "target": "ES2022",
    "module": "NodeNext",
    "moduleResolution": "NodeNext",
    "strict": true,
    "noEmit": true
  },
  "include": ["implementation.ts"]
}
`)

	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	client, err := StartClient(root, directory)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := client.Close(); err != nil {
			t.Error(err)
		}
	}()
	project, err := client.OpenProject(filepath.Join(directory, "tsconfig.json"))
	if err != nil {
		t.Fatal(err)
	}
	source, err := project.SourceFile(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := EncodeSourceFile(source)
	if err != nil {
		t.Fatal(err)
	}
	if len(encoded) == 0 {
		t.Fatal("official source file encoded to no bytes")
	}
	printed, err := client.PrintNode(source, PrintOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(printed, "export function twice(value: number): number") ||
		!strings.Contains(printed, "return value * 2;") {
		t.Fatalf("official AST printed unexpected source:\n%s", printed)
	}
	assertOfficialSourceViewFailsClosed(t, source)
}

func assertOfficialSourceViewFailsClosed(t *testing.T, source SourceFile) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatal("official source exposed a false local statement view")
		}
	}()
	_ = source.Statements()
}
