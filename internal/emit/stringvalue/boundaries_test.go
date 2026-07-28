package stringvalue_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/load"
)

func TestStringFamilyRejectsDeferredNeighboringConstructs(t *testing.T) {
	tests := map[string]string{
		"integer to string conversion": `package boundary

func Convert(value int32) string {
	return string(value)
}
`,
		"rune to string conversion": `package boundary

func Convert(value rune) string {
	return string(value)
}
`,
		"rune iteration": `package boundary

func Count(value string) int {
	count := 0
	for range value {
		count++
	}
	return count
}
`,
	}
	for name, source := range tests {
		t.Run(name, func(t *testing.T) {
			directory := t.TempDir()
			writeBoundaryFile(t, filepath.Join(directory, "go.mod"), "module example.com/boundary\n\ngo 1.26.4\n")
			writeBoundaryFile(t, filepath.Join(directory, "source.go"), source)
			program, err := load.Load(context.Background(), load.Request{
				Directory: directory,
				Pattern:   ".",
			})
			if err != nil {
				t.Fatal(err)
			}
			roots, err := emit.ExportedAPIRoots(program.Roots()[0])
			if err != nil {
				t.Fatal(err)
			}
			_, err = emit.Compile(program, roots)
			var unsupported *api.UnsupportedError
			if !errors.As(err, &unsupported) {
				t.Fatalf("error = %#v, want typed unsupported boundary", err)
			}
		})
	}
}

func writeBoundaryFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
