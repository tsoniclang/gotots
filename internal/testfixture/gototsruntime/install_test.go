package gototsruntime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolutionFixturePreservesCanonicalTypesAndTargetsEmittedValues(t *testing.T) {
	root := t.TempDir()
	runtimeRoot := filepath.Join(root, "runtime")
	if err := os.MkdirAll(runtimeRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(runtimeRoot, "slice.ts"),
		[]byte("export class RuntimeSlice<T> {}\n"),
		0o600,
	); err != nil {
		t.Fatal(err)
	}
	outputDirectory := filepath.Join(root, "out")
	if err := InstallResolution(root, outputDirectory); err != nil {
		t.Fatal(err)
	}
	packageRoot := filepath.Join(root, "node_modules", "@gotots", "runtime")
	declaration, err := os.ReadFile(filepath.Join(packageRoot, "slice.d.ts"))
	if err != nil {
		t.Fatal(err)
	}
	value, err := os.ReadFile(filepath.Join(packageRoot, "slice.js"))
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := os.ReadFile(filepath.Join(packageRoot, "package.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(declaration) != `export * from "../../../runtime/slice.js";
` ||
		string(value) != `export * from "../../../out/runtime/slice.js";
` ||
		!strings.Contains(string(manifest), `"name": "@gotots/runtime"`) {
		t.Fatalf(
			"resolution fixture = declaration %q, value %q, manifest %s",
			declaration,
			value,
			manifest,
		)
	}
}
