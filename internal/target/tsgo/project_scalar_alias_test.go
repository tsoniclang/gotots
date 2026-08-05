package tsgo

import (
	"path/filepath"
	"slices"
	"testing"
)

func TestCallableScalarAliasesPreserveDeclaredAliasIdentities(t *testing.T) {
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
	scalarPath := filepath.Join(directory, "scalars.ts")
	writeProjectFile(t, scalarPath, `
export type int = bigint;
export type int64 = bigint;
export type uint8 = number;
`)
	sourcePath := filepath.Join(directory, "source.ts")
	writeProjectFile(t, sourcePath, `
import type { int, int64, uint8 } from "./scalars.js";

export function Exact(
  value: int,
  nested: readonly [int64, (value: uint8) => Promise<int>],
): Promise<[int64, int]> {
  return Promise.resolve([nested[0], value]);
}

export function Wrong(value: int64): int64 {
  return value;
}

export function InferredVoid() {}
export function InferredBool() { return true; }
`)
	project := openScalarAliasTestProject(t, directory)
	exports, err := project.Exports(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	aliases := map[string]string{
		"int":   scalarPath,
		"int64": scalarPath,
		"uint8": scalarPath,
	}
	exact, err := project.CallableScalarAliases(
		projectExportByName(t, exports, "Exact"),
		aliases,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(exact.Parameters) != 2 ||
		!slices.Equal(exact.Parameters[0], []string{"int"}) ||
		!slices.Equal(
			exact.Parameters[1],
			[]string{"int64", "uint8", "int"},
		) ||
		!slices.Equal(exact.Results, []string{"int64", "int"}) {
		t.Fatalf("exact scalar aliases = %#v", exact)
	}
	wrong, err := project.CallableScalarAliases(
		projectExportByName(t, exports, "Wrong"),
		aliases,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(wrong.Parameters) != 1 ||
		!slices.Equal(wrong.Parameters[0], []string{"int64"}) ||
		!slices.Equal(wrong.Results, []string{"int64"}) {
		t.Fatalf("wrong scalar aliases = %#v", wrong)
	}
	for _, name := range []string{"InferredVoid", "InferredBool"} {
		inferred, err := project.CallableScalarAliases(
			projectExportByName(t, exports, name),
			aliases,
		)
		if err != nil {
			t.Fatal(err)
		}
		if len(inferred.Parameters) != 0 || len(inferred.Results) != 0 {
			t.Fatalf("%s scalar aliases = %#v", name, inferred)
		}
	}
}

func TestCallableScalarAliasesRejectSameNamedForeignAlias(t *testing.T) {
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
	scalarPath := filepath.Join(directory, "scalars.ts")
	writeProjectFile(t, scalarPath, `export type int = bigint;
`)
	writeProjectFile(t, filepath.Join(directory, "foreign.ts"), `export type int = bigint;
`)
	sourcePath := filepath.Join(directory, "source.ts")
	writeProjectFile(t, sourcePath, `
import type { int } from "./foreign.js";
export function Wrong(value: int): int { return value; }
`)
	project := openScalarAliasTestProject(t, directory)
	exports, err := project.Exports(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := project.CallableScalarAliases(
		projectExportByName(t, exports, "Wrong"),
		map[string]string{"int": scalarPath},
	); err == nil {
		t.Fatal("same-named foreign scalar alias was accepted")
	}
}

func openScalarAliasTestProject(
	t *testing.T,
	directory string,
) *ProjectInspection {
	t.Helper()
	client, err := StartClient(repositoryRoot(), directory)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("close client: %v", err)
		}
	})
	project, err := client.OpenProject(filepath.Join(directory, "tsconfig.json"))
	if err != nil {
		t.Fatal(err)
	}
	return project
}
