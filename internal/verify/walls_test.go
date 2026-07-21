package verify

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// repoRoot resolves the repository root from this test file's location:
// internal/verify/walls_test.go -> ../.. is the module root.
func repoRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate walls_test.go")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", ".."))
}

// productImport reports the module-relative package directory of a production
// (non-test) Go file, or "" if the file should be skipped.
func productGoFiles(t *testing.T, root string, visit func(pkgDir string, imports []string)) {
	t.Helper()
	fset := token.NewFileSet()
	for _, base := range []string{"internal", "cmd"} {
		dir := filepath.Join(root, base)
		err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			file, perr := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
			if perr != nil {
				t.Errorf("parse %s: %v", path, perr)
				return nil
			}
			pkgDir, _ := filepath.Rel(root, filepath.Dir(path))
			imports := make([]string, 0, len(file.Imports))
			for _, spec := range file.Imports {
				imports = append(imports, strings.Trim(spec.Path.Value, `"`))
			}
			visit(filepath.ToSlash(pkgDir), imports)
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", dir, err)
		}
	}
}

// TestASTImportsAreWalled proves go/ast and go/types are imported only by the
// two packages the architecture permits: internal/source and
// internal/language/analyze.
func TestASTImportsAreWalled(t *testing.T) {
	root := repoRoot(t)
	allowed := map[string]bool{
		"internal/source":           true,
		"internal/language/analyze": true,
	}
	productGoFiles(t, root, func(pkgDir string, imports []string) {
		for _, imp := range imports {
			if imp == "go/ast" || imp == "go/types" {
				if !allowed[pkgDir] {
					t.Errorf("%s imports %s; only internal/source and internal/language/analyze may", pkgDir, imp)
				}
			}
		}
	})
}

// TestVerifyIsNotImportedByProduction proves no production package imports the
// verification layer.
func TestVerifyIsNotImportedByProduction(t *testing.T) {
	root := repoRoot(t)
	const verifyPkg = "github.com/tsoniclang/gotots/internal/verify"
	productGoFiles(t, root, func(pkgDir string, imports []string) {
		for _, imp := range imports {
			if imp == verifyPkg {
				t.Errorf("%s imports the verification layer %s", pkgDir, verifyPkg)
			}
		}
	})
}
