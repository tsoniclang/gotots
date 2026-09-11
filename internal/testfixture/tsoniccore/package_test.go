package tsoniccore

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolutionPackageRefusesLinkedDirectory(test *testing.T) {
	root := test.TempDir()
	outside := test.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "node_modules", "@tsonic"), 0o755); err != nil {
		test.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "node_modules", "@tsonic", "core")); err != nil {
		test.Fatal(err)
	}
	if err := InstallResolutionOnly(root); err == nil {
		test.Fatal("resolution fixture followed an external package link")
	}
	entries, err := os.ReadDir(outside)
	if err != nil || len(entries) != 0 {
		test.Fatalf("external package was modified: entries=%d error=%v", len(entries), err)
	}
}

func TestResolutionPackageRefusesLinkedFileBeforeWriting(test *testing.T) {
	root := test.TempDir()
	directory := filepath.Join(root, "node_modules", "@tsonic", "core")
	if err := os.MkdirAll(directory, 0o755); err != nil {
		test.Fatal(err)
	}
	external := filepath.Join(test.TempDir(), "types.d.ts")
	if err := os.WriteFile(external, []byte("external"), 0o600); err != nil {
		test.Fatal(err)
	}
	if err := os.Symlink(external, filepath.Join(directory, "types.d.ts")); err != nil {
		test.Fatal(err)
	}
	if err := InstallResolutionOnly(root); err == nil {
		test.Fatal("resolution fixture overwrote a linked declaration")
	}
	content, err := os.ReadFile(external)
	if err != nil || string(content) != "external" {
		test.Fatalf("external declaration changed: %q, %v", content, err)
	}
	if _, err := os.Stat(filepath.Join(directory, "lang.js")); !os.IsNotExist(err) {
		test.Fatalf("fixture wrote before admitting every output file: %v", err)
	}
}
