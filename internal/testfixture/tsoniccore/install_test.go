package tsoniccore

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolutionFixtureIsComplete(t *testing.T) {
	root := t.TempDir()
	if err := InstallResolutionOnly(root); err != nil {
		t.Fatal(err)
	}
	module := filepath.Join(root, "node_modules", "@tsonic", "core")
	declarations, err := os.ReadFile(filepath.Join(module, "lang.d.ts"))
	if err != nil {
		t.Fatal(err)
	}
	runtime, err := os.ReadFile(filepath.Join(module, "lang.js"))
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{
		"addressOf",
		"allocatePointer",
		"loadPointer",
		"storePointer",
		"equalPointer",
		"hashPointer",
		"projectPointer",
		"bindPointer",
		"bindRawPointer",
		"equalRawPointer",
		"hashRawPointer",
	} {
		if !strings.Contains(string(declarations), "function "+name) ||
			!strings.Contains(string(runtime), `unsupported("`+name+`")`) {
			t.Fatalf("resolution fixture lacks %s", name)
		}
	}
	types, err := os.ReadFile(filepath.Join(module, "types.d.ts"))
	if err != nil {
		t.Fatal(err)
	}
	for _, declaration := range []string{
		"interface RawPointer",
		"type int32 = number",
		"type int64 = bigint",
		"type float64 = number",
	} {
		if !strings.Contains(string(types), declaration) {
			t.Fatalf("resolution fixture lacks %s", declaration)
		}
	}
}
