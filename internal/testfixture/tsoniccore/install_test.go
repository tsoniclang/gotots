package tsoniccore

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolutionFixturePinsStructuralInertnessAndRejectsPointerExecution(
	t *testing.T,
) {
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
	for _, name := range []string{"struct", "field"} {
		if !strings.Contains(string(declarations), "function "+name) ||
			strings.Contains(string(runtime), `unsupported("`+name+`")`) ||
			!strings.Contains(string(runtime), "export const "+name) {
			t.Fatalf("resolution fixture lacks inert structural marker %s", name)
		}
	}
	types, err := os.ReadFile(filepath.Join(module, "types.d.ts"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(types), "interface RawPointer") {
		t.Fatal("resolution fixture lacks RawPointer")
	}
}
