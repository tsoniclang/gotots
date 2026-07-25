package stagecheck

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/source"
)

func TestSourceUniverseRejectsSameNamedImportTargetMutation(t *testing.T) {
	directory := t.TempDir()
	writeTopologySource(
		t, directory, "go.mod",
		"module example.com/topologycheck\n\ngo 1.26.0\n",
	)
	writeTopologySource(
		t, directory, "alpha/value.go",
		"package shared\n\nconst Value = 1\n",
	)
	writeTopologySource(
		t, directory, "omega/value.go",
		"package shared\n\nconst Value = 1\n",
	)
	writeTopologySource(
		t, directory, "bridge/bridge.go",
		"package bridge\n\nimport (\n\talpha \"example.com/topologycheck/alpha\"\n\tomega \"example.com/topologycheck/omega\"\n)\n\nconst Value = alpha.Value + omega.Value\n",
	)
	mainPath := writeTopologySource(
		t, directory, "cmd/app/main.go",
		"package main\n\nimport (\n\tshared \"example.com/topologycheck/alpha\"\n\t\"example.com/topologycheck/bridge\"\n)\n\nvar _ = shared.Value + bridge.Value\nfunc main() {}\n",
	)
	request := source.Request{
		Dir: directory, Patterns: []string{"./cmd/app"},
	}
	universe, err := source.ResolveUniverse(request)
	if err != nil {
		t.Fatal(err)
	}
	workspace, err := source.FinalizeResolved(universe)
	if err != nil {
		t.Fatal(err)
	}
	if err := VerifySourceUniverse(workspace, request); err != nil {
		t.Fatalf("unmodified topology failed: %v", err)
	}
	if err := os.WriteFile(
		mainPath,
		[]byte("package main\n\nimport (\n\tshared \"example.com/topologycheck/omega\"\n\t\"example.com/topologycheck/bridge\"\n)\n\nvar _ = shared.Value + bridge.Value\nfunc main() {}\n"),
		0o600,
	); err != nil {
		t.Fatal(err)
	}
	err = VerifySourceUniverse(workspace, request)
	if err == nil ||
		!strings.Contains(err.Error(), "holds import") ||
		!strings.Contains(err.Error(), "misses toolchain import") {
		t.Fatalf(
			"same-named import-target mutation error = %v, want both edge residuals",
			err,
		)
	}
}

func writeTopologySource(
	t *testing.T,
	directory string,
	name string,
	content string,
) string {
	t.Helper()
	path := filepath.Join(directory, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
