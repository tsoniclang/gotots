package verify

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"
)

const modulePath = "github.com/tsoniclang/gotots"

// pkgInfo is one production package: its module-relative directory and the
// deduplicated, sorted set of import paths across its non-test files.
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
// (non-test) package with its imports, in deterministic order. Import paths
// are decoded as Go string literals (strconv.Unquote), never trimmed
// textually, so raw-string imports cannot bypass the walls.
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
			decoded, uerr := strconv.Unquote(spec.Path.Value)
			if uerr != nil {
				t.Errorf("%s: undecodable import literal %s: %v", path, spec.Path.Value, uerr)
				continue
			}
			byDir[dir][decoded] = true
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk module tree: %v", err)
	}
	dirs := make([]string, 0, len(byDir))
	for dir := range byDir {
		dirs = append(dirs, dir)
	}
	sort.Strings(dirs)
	pkgs := make([]pkgInfo, 0, len(dirs))
	for _, dir := range dirs {
		imports := make([]string, 0, len(byDir[dir]))
		for imp := range byDir[dir] {
			imports = append(imports, imp)
		}
		sort.Strings(imports)
		pkgs = append(pkgs, pkgInfo{dir: dir, imports: imports})
	}
	return pkgs
}

// hasSegment reports whether one exact path segment equals name — exact
// segment-boundary matching, never substring matching.
func hasSegment(importPath, name string) bool {
	for _, segment := range strings.Split(importPath, "/") {
		if segment == name {
			return true
		}
	}
	return false
}

// TestWallGateSeesTheTree self-checks the gate: if the walk found no packages
// or missed a known one, the wall rules would pass vacuously.
func TestWallGateSeesTheTree(t *testing.T) {
	pkgs := productPackages(t)
	seen := map[string]bool{}
	for _, p := range pkgs {
		seen[p.dir] = true
	}
	for _, required := range []string{
		"internal/identity", "internal/language/catalog", "internal/language/analyze",
		"internal/source", "internal/scope", "internal/stagecheck", "internal/compiler", "cmd/gotots",
	} {
		if !seen[required] {
			t.Errorf("wall gate did not observe package %s; the walk is incomplete", required)
		}
	}
}

// TestASTImportsAreWalled (Rule 3) proves go/ast and go/types are imported
// only by the two analysis owners and the independent stage verifiers
// (which require toolchain extraction that bypasses the producers).
func TestASTImportsAreWalled(t *testing.T) {
	allowed := map[string]bool{
		"internal/source":           true,
		"internal/language/analyze": true,
		"internal/language/typeset": true,
		"internal/stagecheck":       true,
	}
	for _, p := range productPackages(t) {
		for _, imp := range p.imports {
			if (imp == "go/ast" || imp == "go/types") && !allowed[p.dir] {
				t.Errorf("Rule 3: %s imports %s; only source, analyze, and stagecheck may", p.dir, imp)
			}
		}
	}
}

// TestVerifyIsNotImportedByProduction (Rule 6) proves no production package
// imports the gate layer.
func TestVerifyIsNotImportedByProduction(t *testing.T) {
	verifyPkg := modulePath + "/internal/verify"
	for _, p := range productPackages(t) {
		if p.dir == "internal/verify" {
			continue
		}
		for _, imp := range p.imports {
			if imp == verifyPkg {
				t.Errorf("Rule 6: %s imports the verification gate layer", p.dir)
			}
		}
	}
}

// TestNoCorpusOrIntegrationImports (Rule 2) proves no production package
// imports the acceptance corpus or integration harness, by exact segment.
func TestNoCorpusOrIntegrationImports(t *testing.T) {
	integrationPrefix := modulePath + "/integration"
	for _, p := range productPackages(t) {
		for _, imp := range p.imports {
			if imp == integrationPrefix || strings.HasPrefix(imp, integrationPrefix+"/") ||
				hasSegment(imp, "typescript-go") {
				t.Errorf("Rule 2: %s imports acceptance/integration path %s", p.dir, imp)
			}
		}
	}
}

// layerRank is the total layer registry: every production package has exactly
// one declared layer, and an import edge must go from a higher rank to a
// strictly lower one (Rule 1).
var layerRank = map[string]int{
	"internal/identity":         5,
	"internal/language/catalog": 10,
	"internal/source":           30,
	"internal/language/typeset": 32,
	"internal/scope":            35,
	"internal/language/analyze": 40,
	"internal/stagecheck":       60,
	"internal/compiler":         80,
	"cmd/gotots":                90,
	"internal/verify":           100,
}

// TestLayerRegistryIsTotal proves the registry and the module's production
// packages join exactly: an unranked package fails (the wall cannot be
// bypassed by omission), and a stale registry entry fails.
func TestLayerRegistryIsTotal(t *testing.T) {
	seen := map[string]bool{}
	for _, p := range productPackages(t) {
		seen[p.dir] = true
		if _, ranked := layerRank[p.dir]; !ranked {
			t.Errorf("Rule 1: production package %s has no declared layer; every package must be ranked", p.dir)
		}
	}
	for dir := range layerRank {
		if !seen[dir] {
			t.Errorf("layer registry names %s, which does not exist; delete the stale entry", dir)
		}
	}
}

// TestNoReverseLayerImports (Rule 1) proves lower layers never import higher
// ones, over every internal edge. An unranked importer or import target is
// itself a failure — the check never skips a package.
func TestNoReverseLayerImports(t *testing.T) {
	for _, p := range productPackages(t) {
		importerRank, ranked := layerRank[p.dir]
		if !ranked {
			t.Errorf("Rule 1: %s is not in the layer registry", p.dir)
			continue
		}
		for _, imp := range p.imports {
			if !strings.HasPrefix(imp, modulePath+"/") {
				continue
			}
			importedDir := strings.TrimPrefix(imp, modulePath+"/")
			importedRank, ranked := layerRank[importedDir]
			if !ranked {
				t.Errorf("Rule 1: %s imports unranked package %s", p.dir, importedDir)
				continue
			}
			if importerRank <= importedRank {
				t.Errorf("Rule 1: %s (rank %d) imports %s (rank %d); imports must go to a strictly lower layer",
					p.dir, importerRank, importedDir, importedRank)
			}
		}
	}
}
