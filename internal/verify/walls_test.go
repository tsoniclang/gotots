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

const modulePath = "github.com/tsoniclang/gotots"

// pkgInfo is one production package: its module-relative directory and the
// deduplicated set of import paths across its non-test files.
type pkgInfo struct {
	dir     string
	imports []string
}

// repoRoot resolves the module root from this test file's location:
// internal/verify/walls_test.go -> ../.. is the module root.
func repoRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate walls_test.go")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", ".."))
}

// productPackages walks the entire module tree and returns every production
// (non-test) package with its imports. Walking the whole tree — not a fixed
// list of directories — means a leaking package added anywhere is still gated.
func productPackages(t *testing.T) []pkgInfo {
	t.Helper()
	root := repoRoot(t)
	fset := token.NewFileSet()
	byDir := map[string]map[string]bool{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if path != root && (strings.HasPrefix(name, ".") || name == "testdata" || name == "vendor" || name == "node_modules") {
				return fs.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, perr := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if perr != nil {
			t.Errorf("parse %s: %v", path, perr)
			return nil
		}
		dir, _ := filepath.Rel(root, filepath.Dir(path))
		dir = filepath.ToSlash(dir)
		if byDir[dir] == nil {
			byDir[dir] = map[string]bool{}
		}
		for _, spec := range file.Imports {
			byDir[dir][strings.Trim(spec.Path.Value, `"`)] = true
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk module tree: %v", err)
	}
	pkgs := make([]pkgInfo, 0, len(byDir))
	for dir, set := range byDir {
		imports := make([]string, 0, len(set))
		for imp := range set {
			imports = append(imports, imp)
		}
		pkgs = append(pkgs, pkgInfo{dir: dir, imports: imports})
	}
	return pkgs
}

// TestWallGateSeesTheTree self-checks the gate: if the walk found no packages
// or missed a known one, the wall rules would pass vacuously.
func TestWallGateSeesTheTree(t *testing.T) {
	pkgs := productPackages(t)
	seen := map[string]bool{}
	for _, p := range pkgs {
		seen[p.dir] = true
	}
	for _, required := range []string{"internal/language/catalog", "internal/language/analyze", "internal/source", "internal/compiler", "cmd/gotots"} {
		if !seen[required] {
			t.Errorf("wall gate did not observe package %s; the walk is incomplete", required)
		}
	}
}

// TestASTImportsAreWalled (Rule 3) proves go/ast and go/types are imported only
// by internal/source and internal/language/analyze, anywhere in the tree.
func TestASTImportsAreWalled(t *testing.T) {
	allowed := map[string]bool{
		"internal/source":           true,
		"internal/language/analyze": true,
	}
	for _, p := range productPackages(t) {
		for _, imp := range p.imports {
			if (imp == "go/ast" || imp == "go/types") && !allowed[p.dir] {
				t.Errorf("Rule 3: %s imports %s; only internal/source and internal/language/analyze may", p.dir, imp)
			}
		}
	}
}

// TestVerifyIsNotImportedByProduction (Rule 6) proves no production package
// imports the verification layer.
func TestVerifyIsNotImportedByProduction(t *testing.T) {
	verifyPkg := modulePath + "/internal/verify"
	for _, p := range productPackages(t) {
		if p.dir == "internal/verify" {
			continue
		}
		for _, imp := range p.imports {
			if imp == verifyPkg {
				t.Errorf("Rule 6: %s imports the verification layer", p.dir)
			}
		}
	}
}

// TestNoCorpusOrIntegrationImports (Rule 2) proves no production package imports
// the acceptance corpus or integration harness.
func TestNoCorpusOrIntegrationImports(t *testing.T) {
	integrationPrefix := modulePath + "/integration"
	for _, p := range productPackages(t) {
		for _, imp := range p.imports {
			if strings.HasPrefix(imp, integrationPrefix) || strings.Contains(imp, "typescript-go") {
				t.Errorf("Rule 2: %s imports acceptance/integration path %s", p.dir, imp)
			}
		}
	}
}

// layerRank ranks the packages that exist so far. An import edge must go from a
// higher rank to a strictly lower one; equal-or-reverse edges are layering
// violations (Rule 1). Unranked packages are skipped until they are placed.
var layerRank = map[string]int{
	"internal/language/catalog": 10,
	"internal/source":           30,
	"internal/language/analyze": 40,
	"internal/compiler":         80,
	"cmd/gotots":                90,
	"internal/verify":           100,
}

// TestNoReverseLayerImports (Rule 1) proves lower layers never import higher
// ones. It covers every internal edge between two ranked packages.
func TestNoReverseLayerImports(t *testing.T) {
	for _, p := range productPackages(t) {
		importerRank, ranked := layerRank[p.dir]
		if !ranked {
			continue
		}
		for _, imp := range p.imports {
			if !strings.HasPrefix(imp, modulePath+"/") {
				continue
			}
			importedDir := strings.TrimPrefix(imp, modulePath+"/")
			importedRank, ranked := layerRank[importedDir]
			if !ranked {
				continue
			}
			if importerRank <= importedRank {
				t.Errorf("Rule 1: %s (rank %d) imports %s (rank %d); imports must go to a strictly lower layer",
					p.dir, importerRank, importedDir, importedRank)
			}
		}
	}
}
