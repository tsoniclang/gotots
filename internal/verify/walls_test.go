package verify

import (
	"go/parser"
	"go/token"
	"io/fs"
	"os"
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
		"internal/identity",
		"internal/language/catalog",
		"internal/language/executable",
		"internal/language/frontend",
		"internal/language/semantic",
		"internal/language/selectionfacts",
		"internal/language/structure",
		"internal/language/typesemantics",
		"internal/scope",
		"internal/scope/contract",
		"internal/scope/sourceplan",
		"internal/source",
		"internal/stagecheck",
		"internal/compiler",
		"cmd/gotots",
	} {
		if !seen[required] {
			t.Errorf("wall gate did not observe package %s; the walk is incomplete", required)
		}
	}
}

// TestToolchainObjectImportsAreWalled proves transient syntax/checker imports
// occur only in their explicit Stage-1 owners and the independent verifier.
func TestToolchainObjectImportsAreWalled(t *testing.T) {
	allowed := map[string]map[string]bool{
		"go/ast": {
			"internal/source":                  true,
			"internal/language/structure":      true,
			"internal/language/selectionfacts": true,
			"internal/language/executable":     true,
			"internal/language/frontend":       true,
			"internal/stagecheck":              true,
		},
		"go/types": {
			"internal/source":                  true,
			"internal/language/structure":      true,
			"internal/language/selectionfacts": true,
			"internal/language/typesemantics":  true,
			"internal/language/frontend":       true,
			"internal/stagecheck":              true,
		},
		"go/token": {
			"internal/source":              true,
			"internal/language/structure":  true,
			"internal/language/executable": true,
			"internal/language/frontend":   true,
			"internal/stagecheck":          true,
		},
	}
	for _, p := range productPackages(t) {
		for _, imp := range p.imports {
			permitted, controlled := allowed[imp]
			if controlled && !permitted[p.dir] {
				t.Errorf(
					"toolchain-object wall: %s imports %s without owning that capability",
					p.dir,
					imp,
				)
			}
		}
	}
}

// TestSourceDoesNotOwnCatalogClassification proves source owns acquisition and
// transient lifetime, not language classification.
func TestSourceDoesNotOwnCatalogClassification(t *testing.T) {
	catalogPkg := modulePath + "/internal/language/catalog"
	for _, p := range productPackages(t) {
		if p.dir != "internal/source" {
			continue
		}
		for _, imp := range p.imports {
			if imp == catalogPkg {
				t.Errorf(
					"Stage-1 seam: internal/source imports %s; classification belongs to language/structure",
					imp,
				)
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
	"internal/identity":                5,
	"internal/language/catalog":        10,
	"internal/source":                  20,
	"internal/scope/contract":          25,
	"internal/language/typesemantics":  25,
	"internal/scope/sourceplan":        30,
	"internal/language/structure":      40,
	"internal/language/selectionfacts": 45,
	"internal/scope":                   50,
	"internal/language/executable":     55,
	"internal/language/semantic":       58,
	"internal/language/frontend":       60,
	"internal/stagecheck":              70,
	"internal/compiler":                80,
	"cmd/gotots":                       90,
	"internal/verify":                  100,
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

// TestSupersededStage1ArchitectureIsAbsent rejects the prior mixed-unit,
// source-owned semantic inventory and recursive dependency hydration. These
// names identify deleted abstractions, not compatibility vocabulary.
func TestSupersededStage1ArchitectureIsAbsent(t *testing.T) {
	banned := []string{
		"MixedUnits",
		"RetainedUnit",
		"FullSyntax",
		"unitTopology",
		"filterInfoByMembership",
	}
	for _, file := range productionGoFiles(t) {
		raw, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		for _, spelling := range banned {
			if strings.Contains(string(raw), spelling) {
				t.Errorf(
					"superseded Stage-1 spelling %q remains in %s",
					spelling,
					file,
				)
			}
		}
	}
	if _, err := os.Stat(
		filepath.Join(repoRoot(t), "internal", "language", "analyze"),
	); !os.IsNotExist(err) {
		t.Error("superseded internal/language/analyze package remains")
	}
	hydration, err := os.ReadFile(
		filepath.Join(repoRoot(t), "internal", "source", "hydrate.go"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(hydration), "packages.NeedDeps") {
		t.Error("semantic hydration recursively requests dependency interiors")
	}
}

func TestSupersededStage2ProjectionArchitectureIsAbsent(t *testing.T) {
	semanticBanned := []string{
		"type providerShard struct",
		"decodeProviderShardWithWire",
		"map[identity.PackageID]Package",
		"Local *Package",
		"overlayLocalPackage",
		"providerManifestPackage",
		"writeProviderShard",
		"decodeProviderShard",
		"measureProviderManifest",
	}
	globalBanned := []string{
		"IdentifierOccurrence",
		"TransientContext",
		"ownerCache",
		"transientStructure",
		"typeParameterLocation",
		"typeParameterByLocation",
		"parameterLocations",
		"object.Pos()",
		"Obj().Pos()",
	}
	for _, file := range productionGoFiles(t) {
		raw, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.ToSlash(file)
		if strings.Contains(path, "/internal/language/frontend/") {
			for _, spelling := range []string{
				"AdditionalOccurrenceRefs()",
				"map[identity.PackageID]*packageInput",
				"[]*packageInput",
			} {
				if strings.Contains(string(raw), spelling) {
					t.Errorf(
						"eager Stage-2 package-input spelling %q remains in %s",
						spelling,
						file,
					)
				}
			}
		}
		if strings.Contains(path, "/internal/language/semantic/") {
			for _, spelling := range semanticBanned {
				if strings.Contains(string(raw), spelling) {
					t.Errorf(
						"superseded Stage-2 projection spelling %q remains in %s",
						spelling,
						file,
					)
				}
			}
		}
		for _, spelling := range globalBanned {
			if strings.Contains(string(raw), spelling) {
				t.Errorf(
					"superseded Stage-2 identity/projection spelling %q remains in %s",
					spelling,
					file,
				)
			}
		}
	}
	superseded := filepath.Join(
		repoRoot(t),
		"internal",
		"language",
		"semantic",
		"projection_overlay.go",
	)
	if _, err := os.Stat(superseded); !os.IsNotExist(err) {
		t.Errorf(
			"superseded complete-package overlay remains at %s: %v",
			superseded,
			err,
		)
	}
}

// TestNoGenericProductionBuckets prevents responsibilities from escaping the
// declared package graph through an unowned utility layer.
func TestNoGenericProductionBuckets(t *testing.T) {
	for _, p := range productPackages(t) {
		base := filepath.Base(p.dir)
		switch base {
		case "util", "utils", "helper", "helpers", "misc", "common", "v2":
			t.Errorf("production package %s is an unowned generic bucket", p.dir)
		}
	}
}

// TestProductionFilesStayResponsibilitySized enforces the repository's
// non-generated 600-line ownership limit.
func TestProductionFilesStayResponsibilitySized(t *testing.T) {
	for _, path := range productionGoFiles(t) {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		lines := 0
		if len(raw) != 0 {
			lines = 1 + strings.Count(string(raw), "\n")
			if raw[len(raw)-1] == '\n' {
				lines--
			}
		}
		if lines > 600 {
			t.Errorf(
				"%s has %d lines; split responsibility before 600",
				path,
				lines,
			)
		}
	}
}

func productionGoFiles(t *testing.T) []string {
	t.Helper()
	root := repoRoot(t)
	var files []string
	err := filepath.WalkDir(root, func(
		path string,
		entry fs.DirEntry,
		walkErr error,
	) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			name := entry.Name()
			if path != root &&
				(strings.HasPrefix(name, ".") ||
					name == "testdata" ||
					name == "vendor" ||
					name == "node_modules") {
				return fs.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(path, ".go") &&
			!strings.HasSuffix(path, "_test.go") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk production Go files: %v", err)
	}
	sort.Strings(files)
	return files
}
