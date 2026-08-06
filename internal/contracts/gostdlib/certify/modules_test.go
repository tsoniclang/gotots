package certify

import (
	"os"
	"path/filepath"
	"testing"
)

func TestModuleMapIsClosedAndOrdered(t *testing.T) {
	path := filepath.Join(t.TempDir(), "modules.json")
	writeModuleMap(t, path, `{
  "schemaVersion": 1,
  "modules": [
    {
      "goImportPath": "strings",
      "specifier": "@gotots/gostdlib/strings.js",
      "sourcePath": "src/strings.ts"
    }
  ]
}`)
	seeds, err := readModuleSeeds(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(seeds) != 1 || seeds[0].GoImportPath != "strings" {
		t.Fatalf("module seeds = %#v", seeds)
	}
	writeModuleMap(t, path, `{"schemaVersion":1,"modules":[],"extra":true}`)
	if _, err := readModuleSeeds(path); err == nil {
		t.Fatal("unknown module-map field passed")
	}
}

func writeModuleMap(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
