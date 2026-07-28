package arrayvalue_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/load"
)

func TestArrayFamilyRejectsDeferredNeighborsAtTypedBoundaries(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		category api.Category
	}{
		{
			name: "aggregate element",
			source: `package boundary

type Box struct { Value int32 }

func Use(value [2]Box) int32 { return value[0].Value }
`,
			category: api.CategoryType,
		},
		{
			name: "range",
			source: `package boundary

func Sum(value [2]int32) int32 {
	var total int32
	for _, element := range value {
		total += element
	}
	return total
}
`,
			category: api.CategoryStatement,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := compileBoundary(t, test.source)
			var unsupported *api.UnsupportedError
			if !errors.As(err, &unsupported) ||
				unsupported.Category != test.category {
				t.Fatalf(
					"compile error = %#v, want %s UnsupportedError",
					err,
					test.category,
				)
			}
		})
	}
}

func compileBoundary(t *testing.T, source string) error {
	t.Helper()
	directory := t.TempDir()
	writeFile(
		t,
		filepath.Join(directory, "go.mod"),
		"module example.com/boundary\n\ngo 1.26.4\n",
	)
	writeFile(t, filepath.Join(directory, "source.go"), source)
	program, err := load.Load(context.Background(), load.Request{
		Directory: directory,
		Pattern:   ".",
	})
	if err != nil {
		return err
	}
	roots, err := emit.ExportedAPIRoots(program.Roots()[0])
	if err != nil {
		return err
	}
	_, err = emit.Compile(program, roots)
	return err
}
