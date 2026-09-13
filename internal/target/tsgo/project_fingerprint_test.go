package tsgo

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestComputedMemberFingerprintRetainsPinnedLibraryIdentity(test *testing.T) {
	directory := test.TempDir()
	writeProjectFile(test, filepath.Join(directory, "tsconfig.json"), `{"compilerOptions":{"target":"ES2022","strict":true},"include":["*.ts"]}`)
	implementation := filepath.Join(directory, "implementation.ts")
	writeProjectFile(test, implementation, `export interface Values { [Symbol.iterator](): Iterator<number>; }`)
	client, err := StartClientWithTool(selectedTool(test), directory)
	if err != nil {
		test.Fatal(err)
	}
	test.Cleanup(func() {
		if err := client.Close(); err != nil {
			test.Error(err)
		}
	})
	project, err := client.OpenProject(filepath.Join(directory, "tsconfig.json"))
	if err != nil {
		test.Fatal(err)
	}
	exports, err := project.Exports(implementation)
	if err != nil {
		test.Fatal(err)
	}
	members := projectExportByName(test, exports, "Values").TypeMembers()
	if len(members) != 1 || members[0].Fingerprint() == "" {
		test.Fatal("iterator has no exact fingerprint")
	}
	key, selected, err := project.projectComputedMemberKey(members[0].handles[0])
	if err != nil || !selected || len(key.Owners) != 1 || !strings.HasPrefix(key.Owners[0], "tsgo:"+pinnedSchemaRevision+":bundled:///libs/") {
		test.Fatalf("iterator key = %#v, selected=%t, error=%v", key, selected, err)
	}
}

func TestComputedKeyOwnerDoesNotAdmitForeignImplementations(test *testing.T) {
	root := test.TempDir()
	for _, owner := range []string{filepath.Join(root, "..", "foreign.ts"), "bundled:///libs/../foreign.ts", "bundled:///libs/", "bundled:///elsewhere/library.d.ts"} {
		if _, err := projectComputedKeyOwnerKeys([]string{owner}, root); err == nil {
			test.Fatalf("admitted foreign owner %q", owner)
		}
	}
	if _, err := projectOwnerKeys([]string{"bundled:///libs/lib.es2015.iterable.d.ts"}, root); err == nil {
		test.Fatal("bundled key identity admitted as project implementation")
	}
}

func TestProjectVariableOwnerIsNotItsNominalValueType(test *testing.T) {
	directory := test.TempDir()
	writeProjectFile(test, filepath.Join(directory, "tsconfig.json"), `{"compilerOptions":{"target":"ES2022","strict":true},"include":["*.ts"]}`)
	writeProjectFile(test, filepath.Join(directory, "value.ts"), `export class Value { readonly count = 3; }`)
	implementation := filepath.Join(directory, "implementation.ts")
	writeProjectFile(test, implementation, `import { Value } from "./value.js"; export const first = new Value(); export let second: Value = first;`)
	client, err := StartClientWithTool(selectedTool(test), directory)
	if err != nil {
		test.Fatal(err)
	}
	test.Cleanup(func() {
		if err := client.Close(); err != nil {
			test.Error(err)
		}
	})
	project, err := client.OpenProject(filepath.Join(directory, "tsconfig.json"))
	if err != nil {
		test.Fatal(err)
	}
	exports, err := project.Exports(implementation)
	if err != nil {
		test.Fatal(err)
	}
	for _, name := range []string{"first", "second"} {
		owners := projectExportByName(test, exports, name).ImplementationOwners()
		if len(owners) != 1 || owners[0] != "implementation.ts" {
			test.Fatalf("%s implementation owners = %v", name, owners)
		}
	}
}
