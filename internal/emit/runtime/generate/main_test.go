package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSynchronizeChecksAndRepairsExactGeneratedSurface(t *testing.T) {
	directory := t.TempDir()
	expected := map[string][]byte{
		"package.json": []byte("{}\n"),
		"slice.ts":     []byte("export class RuntimeSlice {}\n"),
	}
	if err := synchronize(directory, expected, false); err != nil {
		t.Fatal(err)
	}
	if err := synchronize(directory, expected, true); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(directory, "slice.ts"),
		[]byte("export class Changed {}\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}
	if err := synchronize(directory, expected, true); err == nil {
		t.Fatal("changed generated runtime passed exact check")
	}
	if err := os.WriteFile(
		filepath.Join(directory, "stale.ts"),
		[]byte("export const stale = true;\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}
	if err := synchronize(directory, expected, false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(directory, "stale.ts")); !os.IsNotExist(err) {
		t.Fatalf("stale generated runtime remains: %v", err)
	}
	if err := synchronize(directory, expected, true); err != nil {
		t.Fatal(err)
	}
}
