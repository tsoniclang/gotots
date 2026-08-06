package certify

import (
	"go/token"
	"path/filepath"
	"testing"
)

func TestSelectedGoSourceLocationRejectsSyntheticBuildFiles(t *testing.T) {
	root := t.TempDir()
	files := token.NewFileSet()
	selected := files.AddFile(
		filepath.Join(root, "src", "net", "net.go"),
		-1,
		8,
	)
	selected.SetLines([]int{0, 4})
	location, admitted, err := selectedGoSourceLocation(
		root,
		files,
		selected.Pos(4),
	)
	if err != nil || !admitted || location != "net/net.go:2:1" {
		t.Fatalf("selected location = %q, %v, %v", location, admitted, err)
	}

	synthetic := files.AddFile(
		filepath.Join(t.TempDir(), "_cgo_gotypes.go"),
		-1,
		8,
	)
	if location, admitted, err := selectedGoSourceLocation(
		root,
		files,
		synthetic.Pos(1),
	); err != nil || admitted || location != "" {
		t.Fatalf("synthetic location = %q, %v, %v", location, admitted, err)
	}
}
