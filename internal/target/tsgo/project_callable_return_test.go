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
type RenamedResult = Result;
export class Operations {
  static create(): Result { return new Result(); }
  static renamed(): RenamedResult { return new Result(); }
  static optional(): Result | undefined { return undefined; }
  static nullable(): Result | null { return null; }
  static accept(value: RenamedResult | undefined): void {}
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
		identity.Matches(projectExportByName(t, exports, "Other")) || identity.IncludesNullish() {
		t.Fatal("callable return identity did not exact-join its export")
	}
	for _, name := range []string{"renamed", "optional", "nullable"} {
		member, ok := operations.ValueMember(name)
		if !ok {
			t.Fatalf("Operations.%s is absent", name)
		}
		identity, err := project.CallableReturnTypeIdentity(member)
		if err != nil {
			t.Fatal(err)
		}
		if !identity.Matches(projectExportByName(t, exports, "Result")) || identity.IncludesNullish() != (name != "renamed") {
			t.Fatalf("%s lost its exact declaration identity or nullability", name)
		}
	}
	accept, ok := operations.ValueMember("accept")
	if !ok {
		t.Fatal("Operations.accept is absent")
	}
	parameter, err := project.CallableParameterTypeIdentity(accept, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !parameter.Matches(projectExportByName(t, exports, "Result")) || !parameter.IncludesNullish() {
		t.Fatal("parameter nullability was erased from its declared identity")
	}
}
