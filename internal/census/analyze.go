package census

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"go/ast"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"

	"github.com/tsoniclang/gotots/internal/inventory"
	"github.com/tsoniclang/gotots/internal/pinning"
	"github.com/tsoniclang/gotots/internal/profile"
)

// generatedFileMarker is the toolchain's documented convention for
// source-generated files (go.dev/s/generatedcode).
var generatedFileMarker = regexp.MustCompile(`(?m)^// Code generated .* DO NOT EDIT\.$`)

// analyze walks every loaded owned/test-only variant: it verifies each file
// against the pinned tree, produces the identity-bearing records, records
// dependency edges and external use, and enforces the pass-1/pass-2 file
// reconciliation and identity uniqueness gates.
func analyze(prof *profile.Profile, inv *inventory.Inventory, tree *pinning.Tree, loaded []*packages.Package, sourceDir string, report *Report) error {
	externalIndex := inv.ExternalIndex()
	externalUse := map[string]*ExternalUse{}
	var edges []Edge

	// Deterministic iteration order.
	roots := append([]*packages.Package{}, loaded...)
	sort.Slice(roots, func(i, j int) bool { return roots[i].ID < roots[j].ID })

	// Per-package analyzed file sets for exact pass-1/pass-2 reconciliation.
	analyzedGoFiles := map[string][]string{}    // production variant files
	analyzedTestFiles := map[string][]string{}  // in-package _test files
	analyzedXTestFiles := map[string][]string{} // black-box files, keyed by owner

	for _, p := range roots {
		role := roleOf(p, sourceDir)
		if role == roleSynthesizedTestMain {
			continue
		}
		owner := ownerPath(p)
		class, _ := prof.Classify(owner)
		if class != profile.ClassOwned && class != profile.ClassTestOnly {
			continue
		}

		// Test-only packages are owned test-support source: every file they
		// contain belongs to the owned-test scope, whichever variant carries
		// it. Owned packages split by variant, with the in-package test
		// variant contributing only its _test.go files (the rest belong to
		// the production variant).
		isTestScope := class == profile.ClassTestOnly || role != roleProduction
		scope := report.Production
		scopeName := "production"
		if isTestScope {
			scope = report.Test
			scopeName = "test"
		}

		countedFiles := 0
		for _, file := range p.Syntax {
			filename := p.Fset.Position(file.Pos()).Filename
			if role == roleInPackageTest && !strings.HasSuffix(filename, "_test.go") {
				continue // production files re-listed in the in-package test variant
			}
			relative, err := filepath.Rel(sourceDir, filename)
			if err != nil || strings.HasPrefix(relative, "..") {
				return fmt.Errorf("file %s is outside the source checkout", filename)
			}
			relative = filepath.ToSlash(relative)
			countedFiles++

			switch role {
			case roleProduction:
				analyzedGoFiles[p.PkgPath] = append(analyzedGoFiles[p.PkgPath], relative)
			case roleInPackageTest:
				analyzedTestFiles[owner] = append(analyzedTestFiles[owner], relative)
			case roleExternalTest:
				analyzedXTestFiles[owner] = append(analyzedXTestFiles[owner], relative)
			}

			data, err := os.ReadFile(filename)
			if err != nil {
				return err
			}
			// Reconcile against the pinned commit: cleanliness alone does
			// not catch ignored files injected into package directories.
			if err := tree.VerifyFile(relative, data); err != nil {
				return err
			}

			lines := countLines(data)
			digest := sha256.Sum256(data)
			fileRecord := FileRecord{
				Path:      relative,
				Package:   p.PkgPath,
				Scope:     scopeName,
				Lines:     lines,
				Sha256:    hex.EncodeToString(digest[:]),
				Generated: generatedFileMarker.Match(data),
			}
			if owner != p.PkgPath {
				fileRecord.Owner = owner
			}
			report.Files = append(report.Files, fileRecord)
			scope.Files++
			scope.Lines += lines

			stats, err := inspectFile(p, file, relative, scopeName, owner, data)
			if err != nil {
				return err
			}
			report.Declarations = append(report.Declarations, stats.declarations...)
			report.Directives = append(report.Directives, stats.directives...)
			report.RareConstructs = append(report.RareConstructs, stats.rare...)
			mergeAggregates(scope, stats)

			if err := recordImports(prof, externalIndex, externalUse, &edges, file, owner, relative, scopeName, isTestScope); err != nil {
				return err
			}
		}
		if countedFiles > 0 && role != roleInPackageTest {
			scope.Packages++
		}
	}

	if err := reconcileFileSets(inv, analyzedGoFiles, analyzedTestFiles, analyzedXTestFiles); err != nil {
		return err
	}

	sortRecords(report, edges, externalUse)
	if err := validateIdentities(report); err != nil {
		return err
	}
	deriveDeclarationAggregates(report)
	return nil
}

// recordImports classifies one file's imports: external use is recorded
// against mandatory pass-1 evidence; imports that violate the profile
// become contradiction edges.
func recordImports(prof *profile.Profile, externalIndex map[string]*inventory.ExternalPackage, externalUse map[string]*ExternalUse, edges *[]Edge, file *ast.File, owner, relative, scopeName string, isTestScope bool) error {
	for _, importSpec := range file.Imports {
		importPath := strings.Trim(importSpec.Path.Value, `"`)
		importClass, category := prof.Classify(importPath)
		switch importClass {
		case profile.ClassExternal:
			use := externalUse[importPath]
			if use == nil {
				evidence := externalIndex[importPath]
				if evidence == nil {
					return fmt.Errorf("typed pass imports external package %s with no pass-1 evidence", importPath)
				}
				use = &ExternalUse{
					Path:          importPath,
					Standard:      evidence.Standard,
					ModulePath:    evidence.ModulePath,
					ModuleVersion: evidence.ModuleVersion,
				}
				externalUse[importPath] = use
			}
			if isTestScope {
				use.TestImporters++
			} else {
				use.ProductionImporters++
			}
			if importSpec.Name != nil && importSpec.Name.Name == "_" {
				use.BlankImports++
			}
			if importSpec.Name != nil && importSpec.Name.Name == "." {
				use.DotImports++
			}
		case profile.ClassHardExcluded:
			*edges = append(*edges, Edge{
				From: owner, File: relative, To: importPath,
				Class: "hard-excluded", Category: category, Scope: scopeName,
			})
		case profile.ClassUnselected:
			*edges = append(*edges, Edge{
				From: owner, File: relative, To: importPath,
				Class: "unselected", Scope: scopeName,
			})
		case profile.ClassTestOnly:
			if !isTestScope {
				*edges = append(*edges, Edge{
					From: owner, File: relative, To: importPath,
					Class: "test-only", Scope: scopeName,
				})
			}
		}
	}
	return nil
}

func mergeAggregates(scope *ScopeReport, stats *fileStats) {
	for k, v := range stats.constructs {
		scope.Constructs[k] += v
	}
	for k, v := range stats.builtins {
		scope.Builtins[k] += v
	}
	for k, v := range stats.rangeOperands {
		scope.RangeOperands[k] += v
	}
	for k, v := range stats.indexOperands {
		scope.IndexOperands[k] += v
	}
	for k, v := range stats.astKinds {
		scope.ASTKinds[k] += v
	}
}

func countLines(data []byte) int {
	lines := 0
	for _, b := range data {
		if b == '\n' {
			lines++
		}
	}
	if len(data) > 0 && data[len(data)-1] != '\n' {
		lines++
	}
	return lines
}
