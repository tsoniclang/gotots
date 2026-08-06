package tsgo

import (
	"path/filepath"
	"slices"
	"testing"
)

func TestCallableParameterNamesFollowDeclaredOrder(t *testing.T) {
	directory := t.TempDir()
	writeProjectFile(t, filepath.Join(directory, "tsconfig.json"), `{
  "compilerOptions": {
    "target": "ES2022",
    "module": "NodeNext",
    "moduleResolution": "NodeNext",
    "strict": true
  },
  "include": ["*.ts"]
}
`)
	sourcePath := filepath.Join(directory, "source.ts")
	writeProjectFile(t, sourcePath, `
export interface Base { Base(): void }
export interface Target extends Base { Target(): void }
export interface InterfaceView<From extends Base, To extends From> {
  (value: From | undefined): To | undefined;
}
export function Consume(
  source: object,
  eof: Error,
  unexpectedEOF: Error,
  noProgress: Error,
  view: InterfaceView<Base, Target>,
): void {
  void source;
  void eof;
  void unexpectedEOF;
  void noProgress;
  void view;
}
`)
	project := openScalarAliasTestProject(t, directory)
	exports, err := project.Exports(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	names, err := project.CallableParameterNames(
		projectExportByName(t, exports, "Consume"),
	)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"source", "eof", "unexpectedEOF", "noProgress", "view"}
	if !slices.Equal(names, want) {
		t.Fatalf("parameter names = %q, want %q", names, want)
	}
	arguments, err := project.CallableParameterTypeArguments(
		projectExportByName(t, exports, "Consume"),
		4,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(arguments) != 2 ||
		!arguments[0].Matches(projectExportByName(t, exports, "Base")) ||
		!arguments[1].Matches(projectExportByName(t, exports, "Target")) {
		t.Fatalf("view type arguments = %#v", arguments)
	}
}
