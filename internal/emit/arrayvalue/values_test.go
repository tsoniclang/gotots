package arrayvalue_test

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestArrayValuesCompileThroughTheDirectEmitter(t *testing.T) {
	compileArrayFixture(t)
}

func arrayValuesDirectory() string {
	return filepath.Join(repositoryRoot(), "testdata", "projects", "array-values")
}

func repositoryRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		panic("resolve test source path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}
