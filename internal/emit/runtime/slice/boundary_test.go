package slice_test

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

func TestScalarSliceNeighborsFailAtTypedOwners(t *testing.T) {
	tests := []struct {
		name      string
		source    string
		category  api.Category
		construct string
	}{
		{
			name: "range",
			source: `package boundary
func F(values []int32) int32 {
	var total int32
	for _, value := range values { total += value }
	return total
}
`,
			category:  api.CategoryStatement,
			construct: "*ast.RangeStmt",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := compileBoundary(t, test.source)
			var unsupported *api.UnsupportedError
			if !errors.As(err, &unsupported) {
				t.Fatalf("error = %#v, want typed unsupported boundary", err)
			}
			if unsupported.Category != test.category ||
				unsupported.Construct != test.construct {
				t.Fatalf(
					"unsupported = (%s, %s, %s), want (%s, %s)",
					unsupported.Category,
					unsupported.Construct,
					unsupported.Role,
					test.category,
					test.construct,
				)
			}
		})
	}
}

func TestNestedSliceTypeUsesRecursiveDescriptor(t *testing.T) {
	err := compileBoundary(t, `package boundary
func F(values [][]int32) int { return len(values) }
`)
	if err != nil {
		t.Fatalf("nested slice type was rejected: %v", err)
	}
}

func compileBoundary(t *testing.T, source string) error {
	t.Helper()
	directory := t.TempDir()
	if err := os.WriteFile(
		filepath.Join(directory, "go.mod"),
		[]byte("module example.com/boundary\n\ngo 1.26.4\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(directory, "source.go"),
		[]byte(source),
		0o644,
	); err != nil {
		t.Fatal(err)
	}
	program, err := load.Load(context.Background(), load.Request{
		Directory: directory,
		Pattern:   ".",
	})
	if err != nil {
		t.Fatalf("boundary source was not valid Go: %v", err)
	}
	root, err := emit.NewRoot(program.Roots()[0].Types().Scope().Lookup("F"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = emit.Compile(program, []emit.Root{root})
	return err
}
