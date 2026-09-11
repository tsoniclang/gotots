package tsoniccore

import (
	"encoding/json"
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
	for _, name := range []string{"@tsonic/core", "@gotots/abi"} {
		bytes, err := os.ReadFile(filepath.Join(root, "node_modules", name, "package.json"))
		if err != nil {
			t.Fatal(err)
		}
		var manifest struct {
			Name             string            `json:"name"`
			Version          string            `json:"version"`
			Private          bool              `json:"private"`
			PeerDependencies map[string]string `json:"peerDependencies"`
		}
		if err := json.Unmarshal(bytes, &manifest); err != nil {
			t.Fatal(err)
		}
		if manifest.Name != name || manifest.Version != "0.0.0" || !manifest.Private {
			t.Fatalf("resolution package metadata = %#v", manifest)
		}
		if name == "@gotots/abi" && manifest.PeerDependencies["@tsonic/core"] != "0.0.0" {
			t.Fatalf("ABI resolution dependency = %#v", manifest.PeerDependencies)
		}
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
		"toRawPointer",
		"reinterpretRawPointer",
		"offsetRawPointer",
		"memoryLayout",
		"memoryArrayLayout",
		"defaultValue",
		"memoryField",
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
		"interface FixedArray<T, TLength extends number | bigint>",
		"readonly length: TLength",
		"type int32 = number",
		"type int64 = bigint",
		"type float64 = number",
	} {
		if !strings.Contains(string(types), declaration) {
			t.Fatalf("resolution fixture lacks %s", declaration)
		}
	}
}
