package tsgo

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestProjectExportsUseTSGoSymbolsAndDeclarationOwners(t *testing.T) {
	projectDirectory := t.TempDir()
	writeProjectFile(t, filepath.Join(projectDirectory, "tsconfig.json"), `{
  "compilerOptions": {
    "target": "ES2022",
    "module": "NodeNext",
    "moduleResolution": "NodeNext",
    "strict": true
  },
  "include": ["*.ts"]
}
`)
	implementationPath := filepath.Join(projectDirectory, "implementation.ts")
	writeProjectFile(t, implementationPath, `export function Value(input: string): number {
  return input.length;
}

export class Box {
  static Make(value: number): Box {
    return new Box(value);
  }

  private constructor(readonly value: number) {}
}

const Hidden = 1;
void Hidden;
`)
	entryPath := filepath.Join(projectDirectory, "entry.ts")
	writeProjectFile(t, entryPath, `export { Box, Value } from "./implementation.js";
export const state: { Count: number } = { Count: 0 };
`)
	renamedPath := filepath.Join(projectDirectory, "renamed.ts")
	writeProjectFile(t, renamedPath, `export { Value as Other } from "./implementation.js";
`)

	client, err := StartClient(repositoryRoot(), projectDirectory)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("close client: %v", err)
		}
	})
	project, err := client.OpenProject(
		filepath.Join(projectDirectory, "tsconfig.json"),
	)
	if err != nil {
		t.Fatal(err)
	}
	exports, err := project.Exports(entryPath)
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, len(exports))
	for index, selected := range exports {
		names[index] = selected.Name()
		if selected.TypeString() == "" ||
			selected.Flags() == 0 ||
			len(selected.Declarations()) == 0 {
			t.Fatalf("invalid export %#v", selected)
		}
		if selected.Name() == "state" {
			if !slices.Equal(
				selected.Declarations(),
				[]string{filepath.ToSlash(entryPath)},
			) {
				t.Fatalf(
					"state declarations = %v",
					selected.Declarations(),
				)
			}
			continue
		}
		if !slices.Equal(
			selected.Declarations(),
			[]string{filepath.ToSlash(implementationPath)},
		) {
			t.Fatalf(
				"%s declarations = %v",
				selected.Name(),
				selected.Declarations(),
			)
		}
	}
	if !slices.Equal(names, []string{"Box", "Value", "state"}) {
		t.Fatalf("exports = %v", names)
	}
	renamed, err := project.Exports(renamedPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(renamed) != 1 ||
		renamed[0].Name() != "Other" ||
		renamed[0].TypeString() != "(input: string) => number" {
		t.Fatalf("renamed exports = %#v", renamed)
	}
}

func TestDeclarationPathRejectsMalformedHandles(t *testing.T) {
	for _, handle := range []string{"", "one", "one.two", "one.two."} {
		if path, ok := declarationPath(handle); ok || path != "" {
			t.Fatalf("declarationPath(%q) = %q, %t", handle, path, ok)
		}
	}
	if path, ok := declarationPath("12.34./project/source.ts"); !ok || path != "/project/source.ts" {
		t.Fatalf("declaration path = %q, %t", path, ok)
	}
}

func writeProjectFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
