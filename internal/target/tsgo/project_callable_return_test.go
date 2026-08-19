package tsgo

import (
	"path/filepath"
	"testing"
)

func TestCallableReturnTypeIdentityExactJoinsExport(t *testing.T) {
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
	entryPath := filepath.Join(projectDirectory, "entry.ts")
	writeProjectFile(t, entryPath, `
export class Result {}
export class Other {}
export class Operations {
  static create(): Result { return new Result(); }
}
`)
	client, err := StartClientWithTool(selectedTool(t), projectDirectory)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("close client: %v", err)
		}
	})
	project, err := client.OpenProject(filepath.Join(
		projectDirectory,
		"tsconfig.json",
	))
	if err != nil {
		t.Fatal(err)
	}
	exports, err := project.Exports(entryPath)
	if err != nil {
		t.Fatal(err)
	}
	operations := projectExportByName(t, exports, "Operations")
	create, ok := operations.ValueMember("create")
	if !ok {
		t.Fatal("Operations.create is absent")
	}
	identity, err := project.CallableReturnTypeIdentity(create)
	if err != nil {
		t.Fatal(err)
	}
	if !identity.Matches(projectExportByName(t, exports, "Result")) ||
		identity.Matches(projectExportByName(t, exports, "Other")) {
		t.Fatal("callable return identity did not exact-join its export")
	}
}
