package census

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"go/ast"
	"go/types"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"

	"github.com/tsoniclang/gotots/internal/inventory"
	"github.com/tsoniclang/gotots/internal/pinning"
	"github.com/tsoniclang/gotots/internal/profile"
	"github.com/tsoniclang/gotots/internal/typedload"
)

// generatedFileMarker is the toolchain's documented convention for
// source-generated files (go.dev/s/generatedcode). The convention requires
// the marker line to appear before the first non-comment, non-blank text.
var generatedFileMarker = regexp.MustCompile(`^// Code generated .* DO NOT EDIT\.$`)

// generatedBy returns the marker line when the file follows the
// generated-code convention, or "" otherwise.
func generatedBy(data []byte) string {
	for line := range strings.Lines(string(data)) {
		trimmed := strings.TrimRight(line, "\r\n")
		if generatedFileMarker.MatchString(trimmed) {
			return trimmed
		}
		blank := strings.TrimSpace(trimmed) == ""
		comment := strings.HasPrefix(trimmed, "//")
		if !blank && !comment {
			// Block comments before the marker are unusual; the documented
			// convention is satisfied by line comments and blank lines only.
			return ""
		}
	}
	return ""
}

// analyze walks every loaded owned/test-only variant: it verifies each file
// against the pinned tree, produces the identity-bearing records and typed
// declaration shapes, records dependency edges and external obligations,
// and enforces the pass-1/pass-2 file reconciliation and identity
// uniqueness gates.
func analyze(prof *profile.Profile, inv *inventory.Inventory, tree *pinning.Tree, loaded []*packages.Package, sourceDir string, report *Report, shapes *DeclarationShapes) (*ExternalContract, error) {
	externalIndex := inv.ExternalIndex()
	externalUse := map[string]*ExternalUse{}
	obligations := map[string]*ExternalObligation{}
	externalObjects := map[types.Object]*contractRef{}
	var edges []Edge
	isExternal := func(pkgPath string) bool {
		class, _ := prof.Classify(pkgPath)
		return class == profile.ClassExternal
	}

	// Deterministic iteration order.
	roots := append([]*packages.Package{}, loaded...)
	sort.Slice(roots, func(i, j int) bool { return roots[i].ID < roots[j].ID })

	// Per-package analyzed file sets for exact pass-1/pass-2 reconciliation.
	analyzedGoFiles := map[string][]string{}    // production variant files
	analyzedTestFiles := map[string][]string{}  // in-package _test files
	analyzedXTestFiles := map[string][]string{} // black-box files, keyed by owner

	for _, p := range roots {
		role := typedload.RoleOf(p, sourceDir)
		if role == typedload.RoleSynthesizedTestMain {
			continue
		}
		owner := typedload.OwnerPath(p)
		class, _ := prof.Classify(owner)
		if class != profile.ClassOwned && class != profile.ClassTestOnly {
			continue
		}

		// Test-only packages are owned test-support source: every file they
		// contain belongs to the owned-test scope, whichever variant carries
		// it. Owned packages split by variant, with the in-package test
		// variant contributing only its _test.go files (the rest belong to
		// the production variant).
		isTestScope := class == profile.ClassTestOnly || role != typedload.RoleProduction
		scope := report.Production
		scopeName := "production"
		if isTestScope {
			scope = report.Test
			scopeName = "test"
		}

		countedFiles := 0
		for _, file := range p.Syntax {
			filename := p.Fset.Position(file.Pos()).Filename
			if role == typedload.RoleInPackageTest && !strings.HasSuffix(filename, "_test.go") {
				continue // production files re-listed in the in-package test variant
			}
			relative, err := filepath.Rel(sourceDir, filename)
			if err != nil || strings.HasPrefix(relative, "..") {
				return nil, fmt.Errorf("file %s is outside the source checkout", filename)
			}
			relative = filepath.ToSlash(relative)
			countedFiles++

			switch role {
			case typedload.RoleProduction:
				analyzedGoFiles[p.PkgPath] = append(analyzedGoFiles[p.PkgPath], relative)
			case typedload.RoleInPackageTest:
				analyzedTestFiles[owner] = append(analyzedTestFiles[owner], relative)
			case typedload.RoleExternalTest:
				analyzedXTestFiles[owner] = append(analyzedXTestFiles[owner], relative)
			}

			data, err := os.ReadFile(filename)
			if err != nil {
				return nil, err
			}
			// Reconcile against the pinned commit: cleanliness alone does
			// not catch ignored files injected into package directories.
			if err := tree.VerifyFile(relative, data); err != nil {
				return nil, err
			}

			lines := countLines(data)
			digest := sha256.Sum256(data)
			marker := generatedBy(data)
			fileRecord := FileRecord{
				Path:        relative,
				Package:     p.PkgPath,
				Scope:       scopeName,
				Lines:       lines,
				Sha256:      hex.EncodeToString(digest[:]),
				Generated:   marker != "",
				GeneratedBy: marker,
			}
			if owner != p.PkgPath {
				fileRecord.Owner = owner
			}
			report.Files = append(report.Files, fileRecord)
			scope.Files++
			scope.Lines += lines

			stats, err := inspectFile(p, file, relative, scopeName, owner, data, isExternal)
			if err != nil {
				return nil, err
			}
			report.Declarations = append(report.Declarations, stats.declarations...)
			report.Directives = append(report.Directives, stats.directives...)
			report.RareConstructs = append(report.RareConstructs, stats.rare...)
			report.TestFunctions = append(report.TestFunctions, stats.testFunctions...)
			shapes.Functions = append(shapes.Functions, stats.functionShapes...)
			shapes.Types = append(shapes.Types, stats.typeShapes...)
			shapes.Constants = append(shapes.Constants, stats.constShapes...)
			shapes.Variables = append(shapes.Variables, stats.varShapes...)
			shapes.Aliases = append(shapes.Aliases, stats.aliasShapes...)
			shapes.FunctionLiterals = append(shapes.FunctionLiterals, stats.funcLitShapes...)
			for object, owner := range stats.externalObjects {
				reference := externalObjects[object]
				if reference == nil {
					reference = &contractRef{}
					externalObjects[object] = reference
				}
				if isTestScope {
					reference.Test = true
				} else {
					reference.Production = true
				}
				if reference.Owner == "" {
					reference.Owner = owner
				}
			}
			for key, use := range stats.externalUses {
				existing := obligations[key]
				if existing == nil {
					existing = &ExternalObligation{Package: use.Package, Name: use.Name, Kind: use.Kind}
					obligations[key] = existing
				}
				// Per-file stats count into ProductionUses; the file's scope
				// decides the real bucket here.
				if isTestScope {
					existing.TestUses += use.ProductionUses
				} else {
					existing.ProductionUses += use.ProductionUses
				}
			}
			mergeAggregates(scope, stats)

			if err := recordImports(prof, externalIndex, externalUse, &edges, file, owner, relative, scopeName, isTestScope); err != nil {
				return nil, err
			}
		}
		if countedFiles > 0 && role != typedload.RoleInPackageTest {
			scope.Packages++
		}
	}

	if err := reconcileFileSets(inv, analyzedGoFiles, analyzedTestFiles, analyzedXTestFiles); err != nil {
		return nil, err
	}

	for _, obligation := range obligations {
		report.ExternalObligations = append(report.ExternalObligations, *obligation)
	}
	sortRecords(report, edges, externalUse)
	sortShapes(shapes)
	if err := validateIdentities(report); err != nil {
		return nil, err
	}
	deriveDeclarationAggregates(report)

	contract, err := buildExternalContract(externalObjects, isExternal)
	if err != nil {
		return nil, err
	}
	return contract, nil
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
		case profile.ClassOutsideUniverse:
			// The inventory scope-closure check fails before analysis can
			// see such an edge; reaching here is a filter defect.
			return fmt.Errorf("GOTOTS_SCOPE_DEPENDENCY_OUTSIDE:\n%s (%s) imports outside-universe package %s (%s)",
				owner, scopeName, importPath, category)
		case profile.ClassPolicyExcluded, profile.ClassTooling:
			// Contract rule 10: no selected source imports policy-excluded
			// or tooling packages. Recorded as a contradiction edge — the
			// census completes as non-authoritative evidence and gate 02
			// fails on the blocker.
			*edges = append(*edges, Edge{
				From: owner, File: relative, To: importPath,
				Class: string(importClass), Scope: scopeName,
			})
		case profile.ClassUnclassified:
			// Total-classification defect: the package matched no rule.
			*edges = append(*edges, Edge{
				From: owner, File: relative, To: importPath,
				Class: "unclassified", Scope: scopeName,
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
