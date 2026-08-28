package tsgo

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestProjectSourceFileUsesOfficialASTForPrinting(t *testing.T) {
	directory := t.TempDir()
	sourcePath := filepath.Join(directory, "implementation.ts")
	sourceText := `export function twice(value: number): number {
  const label = "λ";
  void label;
  return value * 2;
}
`
	writeProjectFile(t, sourcePath, sourceText)
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

	client, err := StartClientWithTool(selectedTool(t), directory)
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
	evidence, err := project.projectSourceEvidence(filepath.ToSlash(sourcePath))
	if err != nil {
		t.Fatal(err)
	}
	if evidence.text != sourceText {
		t.Fatalf("official immutable source text differs:\n%q\nwant:\n%q", evidence.text, sourceText)
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
