package emit_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
)

func TestDemandCompilerRejectsUnsupportedDeclarationRoot(t *testing.T) {
	projectDirectory := projectPath("constructs", "declaration", "unsupported", "variable")
	loaded, err := load.One(context.Background(), load.Request{
		Directory: projectDirectory,
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = emit.CompileFile(loaded, loaded.Files()[0].Syntax())
	var unsupported *emit.ScheduleError
	if !errors.As(err, &unsupported) ||
		unsupported.Object != "Value" ||
		unsupported.Reason != "object has no supported source declaration" {
		t.Fatalf("error = %#v, want exact unsupported declaration obligation", err)
	}
}

func projectPath(elements ...string) string {
	parts := append([]string{"..", "..", "testdata"}, elements...)
	return filepath.Join(parts...)
}
