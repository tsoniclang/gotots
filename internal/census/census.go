// Package census produces the typed, reproducible source census that is the
// first gotots deliverable: a complete partition of the pinned source into
// non-overlapping ownership classes plus typed inventories of declarations,
// bodies, language constructs, directives, and external obligations.
//
// The census is evidence, not translation. It must fail closed on load or
// typecheck errors and must never reclassify a unit because analysis failed.
package census

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"

	"github.com/tsoniclang/gotots/internal/pinning"
	"github.com/tsoniclang/gotots/internal/profile"
)

// Edge records one dependency from an owned package to a package it must not
// depend on under the profile. Every edge is a profile contradiction that
// must be resolved by review before generation can be attempted.
type Edge struct {
	From     string `json:"from"`
	To       string `json:"to"`
	Class    string `json:"class"`    // hard-excluded | unselected
	Category string `json:"category"` // exclusion category when hard-excluded
	Scope    string `json:"scope"`    // production | test
}

// DeclCounts inventories declarations by kind.
type DeclCounts struct {
	Functions int `json:"functions"`
	Methods   int `json:"methods"`
	// BodylessFunctions counts declarations with no Go body (assembly,
	// linkname, or intrinsic). Each needs an explicit disposition later.
	BodylessFunctions int `json:"bodylessFunctions"`
	NamedTypes        int `json:"namedTypes"`
	Aliases           int `json:"aliases"`
	Constants         int `json:"constants"`
	Variables         int `json:"variables"`
}

// ScopeReport aggregates one non-overlapping source scope
// (owned production or owned test).
type ScopeReport struct {
	Packages      int            `json:"packages"`
	Files         int            `json:"files"`
	Lines         int            `json:"lines"`
	Declarations  DeclCounts     `json:"declarations"`
	Bodies        int            `json:"bodies"`
	Statements    int            `json:"statements"`
	Constructs    map[string]int `json:"constructs"`
	Builtins      map[string]int `json:"builtins"`
	RangeOperands map[string]int `json:"rangeOperands"`
	IndexOperands map[string]int `json:"indexOperands"`
	Directives    map[string]int `json:"directives"`
	ASTKinds      map[string]int `json:"astKinds"`
}

func newScopeReport() *ScopeReport {
	return &ScopeReport{
		Constructs:    map[string]int{},
		Builtins:      map[string]int{},
		RangeOperands: map[string]int{},
		IndexOperands: map[string]int{},
		Directives:    map[string]int{},
		ASTKinds:      map[string]int{},
	}
}

// PackageReport is the per-package summary for owned packages.
type PackageReport struct {
	Path         string     `json:"path"`
	Files        int        `json:"files"`
	Lines        int        `json:"lines"`
	Declarations DeclCounts `json:"declarations"`
	Bodies       int        `json:"bodies"`
	Statements   int        `json:"statements"`
	TestFiles    int        `json:"testFiles"`
	TestLines    int        `json:"testLines"`
	TestBodies   int        `json:"testBodies"`
}

// ExternalPackage records one external import obligation.
type ExternalPackage struct {
	Path              string `json:"path"`
	Class             string `json:"class"` // external-std | external-module
	Productioncallers int    `json:"productionImporters"`
	TestImporters     int    `json:"testImporters"`
	BlankImports      int    `json:"blankImports"`
	DotImports        int    `json:"dotImports"`
}

// Partition lists every discovered package path in exactly one class.
type Partition struct {
	Owned        []string `json:"owned"`
	TestOnly     []string `json:"ownedTestSupport"`
	HardExcluded []string `json:"hardExcluded"`
	Unselected   []string `json:"unselected"`
	ExternalStd  []string `json:"externalStd"`
	ExternalMod  []string `json:"externalModule"`
}

// Report is the complete deterministic census output.
type Report struct {
	SchemaVersion  int                     `json:"schemaVersion"`
	Product        string                  `json:"product"`
	BuildProfile   profile.BuildProfile    `json:"buildProfile"`
	Pin            pinning.Pin             `json:"pin"`
	Source         *pinning.VerifiedSource `json:"source"`
	Partition      Partition               `json:"partition"`
	Contradictions []Edge                  `json:"contradictions"`
	Production     *ScopeReport            `json:"ownedProduction"`
	Test           *ScopeReport            `json:"ownedTest"`
	Packages       []PackageReport         `json:"packages"`
	External       []ExternalPackage       `json:"external"`
}

// Run verifies the pin, loads the owned roots under the selected build
// profile with full type information, and produces the census report.
func Run(prof *profile.Profile, sourceDir string, buildProfileName string) (*Report, error) {
	var selected *profile.BuildProfile
	for i := range prof.BuildProfiles {
		if prof.BuildProfiles[i].Name == buildProfileName {
			selected = &prof.BuildProfiles[i]
		}
	}
	if selected == nil {
		return nil, fmt.Errorf("build profile %q is not defined in the project profile", buildProfileName)
	}

	verified, err := pinning.Verify(prof.Pin, sourceDir)
	if err != nil {
		return nil, err
	}

	loaded, err := load(prof, sourceDir, selected)
	if err != nil {
		return nil, err
	}

	report := &Report{
		SchemaVersion: 1,
		Product:       prof.Product,
		BuildProfile:  *selected,
		Pin:           *prof.Pin,
		Source:        verified,
		Production:    newScopeReport(),
		Test:          newScopeReport(),
	}

	if err := analyze(prof, loaded, report); err != nil {
		return nil, err
	}
	return report, nil
}

func load(prof *profile.Profile, sourceDir string, build *profile.BuildProfile) ([]*packages.Package, error) {
	patternRoots := append(append([]string{}, prof.OwnedRoots...), prof.TestOnlyRoots...)
	sort.Strings(patternRoots)
	patterns := make([]string, 0, len(patternRoots))
	for _, root := range patternRoots {
		patterns = append(patterns, "./"+root+"/...")
	}

	env := append(os.Environ(),
		"GOOS="+build.GOOS,
		"GOARCH="+build.GOARCH,
		"CGO_ENABLED=0",
		"GOPROXY=off",
		"GOFLAGS=-mod=mod",
	)
	config := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles |
			packages.NeedImports | packages.NeedDeps | packages.NeedModule |
			packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo,
		Dir:   sourceDir,
		Env:   env,
		Tests: true,
	}
	if len(build.Tags) > 0 {
		config.BuildFlags = []string{"-tags=" + strings.Join(build.Tags, ",")}
	}

	loaded, err := packages.Load(config, patterns...)
	if err != nil {
		return nil, fmt.Errorf("load packages: %w", err)
	}

	var loadErrors []string
	packages.Visit(loaded, nil, func(p *packages.Package) {
		for _, e := range p.Errors {
			loadErrors = append(loadErrors, fmt.Sprintf("%s: %s", p.ID, e))
		}
	})
	if len(loadErrors) > 0 {
		sort.Strings(loadErrors)
		return nil, fmt.Errorf("census fails closed on %d load/type errors:\n%s",
			len(loadErrors), strings.Join(loadErrors, "\n"))
	}
	return loaded, nil
}

// packageRole distinguishes the go/packages test variants.
type packageRole int

const (
	roleProduction packageRole = iota
	roleInPackageTest
	roleExternalTest
	roleSynthesizedTestMain
)

func roleOf(p *packages.Package) packageRole {
	if strings.HasSuffix(p.PkgPath, ".test") {
		return roleSynthesizedTestMain
	}
	if !strings.Contains(p.ID, " [") {
		return roleProduction
	}
	if strings.HasSuffix(p.PkgPath, "_test") {
		return roleExternalTest
	}
	return roleInPackageTest
}

// basePath maps an external test package path (p_test) to its package under
// classification (p). Other paths are returned unchanged.
func basePath(pkgPath string) string {
	return strings.TrimSuffix(pkgPath, "_test")
}

func analyze(prof *profile.Profile, loaded []*packages.Package, report *Report) error {
	classes := map[string]profile.PackageClass{}
	categories := map[string]string{}
	external := map[string]*ExternalPackage{}
	perPackage := map[string]*PackageReport{}
	var contradictions []Edge

	classify := func(pkgPath string) profile.PackageClass {
		if c, ok := classes[pkgPath]; ok {
			return c
		}
		class, category := prof.Classify(basePath(pkgPath))
		classes[pkgPath] = class
		categories[pkgPath] = category
		return class
	}

	recordExternal := func(pkgPath string, class profile.PackageClass, test bool) {
		entry := external[pkgPath]
		if entry == nil {
			entry = &ExternalPackage{Path: pkgPath, Class: string(class)}
			external[pkgPath] = entry
		}
		if test {
			entry.TestImporters++
		} else {
			entry.Productioncallers++
		}
	}

	// Deterministic iteration: sort by package ID.
	sorted := make([]*packages.Package, 0)
	packages.Visit(loaded, nil, func(p *packages.Package) { sorted = append(sorted, p) })
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].ID < sorted[j].ID })

	for _, p := range sorted {
		role := roleOf(p)
		if role == roleSynthesizedTestMain {
			continue
		}
		class := classify(p.PkgPath)
		if class != profile.ClassOwned && class != profile.ClassTestOnly {
			continue
		}
		// Test-only packages are owned test-support source: every file they
		// contain belongs to the owned-test scope, whichever variant carries
		// it. Owned packages split by variant.
		isTestScope := class == profile.ClassTestOnly || role != roleProduction
		scope := report.Production
		scopeName := "production"
		if isTestScope {
			scope = report.Test
			scopeName = "test"
		}

		pkgReport := perPackage[basePath(p.PkgPath)]
		if pkgReport == nil {
			pkgReport = &PackageReport{Path: basePath(p.PkgPath)}
			perPackage[basePath(p.PkgPath)] = pkgReport
		}

		// Select the files this variant contributes: production packages
		// contribute every compiled file; the in-package test variant
		// contributes only its _test.go files (its other files belong to the
		// production variant); external test packages contribute everything.
		countedFiles := 0
		for _, file := range p.Syntax {
			filename := p.Fset.Position(file.Pos()).Filename
			if role == roleInPackageTest && !strings.HasSuffix(filename, "_test.go") {
				continue
			}
			countedFiles++
			lines, err := countLines(filename)
			if err != nil {
				return err
			}
			scope.Lines += lines
			if isTestScope {
				pkgReport.TestFiles++
				pkgReport.TestLines += lines
			} else {
				pkgReport.Files++
				pkgReport.Lines += lines
			}
			stats := inspectFile(p, file)
			mergeFileStats(scope, pkgReport, stats, isTestScope)

			for _, imp := range file.Imports {
				importPath := strings.Trim(imp.Path.Value, `"`)
				importClass := classify(importPath)
				switch importClass {
				case profile.ClassExternalStd, profile.ClassExternalMod:
					recordExternal(importPath, importClass, isTestScope)
					if imp.Name != nil && imp.Name.Name == "_" {
						external[importPath].BlankImports++
					}
					if imp.Name != nil && imp.Name.Name == "." {
						external[importPath].DotImports++
					}
				case profile.ClassHardExcluded:
					contradictions = append(contradictions, Edge{
						From: p.PkgPath, To: importPath,
						Class: "hard-excluded", Category: categories[importPath],
						Scope: scopeName,
					})
				case profile.ClassUnselected:
					contradictions = append(contradictions, Edge{
						From: p.PkgPath, To: importPath,
						Class: "unselected", Scope: scopeName,
					})
				case profile.ClassTestOnly:
					// Test scope may use owned test-support freely; a
					// production import of test-support is a contradiction.
					if !isTestScope {
						contradictions = append(contradictions, Edge{
							From: p.PkgPath, To: importPath,
							Class: "test-only", Scope: scopeName,
						})
					}
				}
			}
		}
		if countedFiles > 0 {
			scope.Files += countedFiles
			if role == roleProduction || role == roleExternalTest {
				scope.Packages++
			}
		}
	}

	// The partition must list every package the loader observed, each in
	// exactly one class.
	partitionSets := map[profile.PackageClass][]string{}
	seen := map[string]bool{}
	packages.Visit(loaded, nil, func(p *packages.Package) {
		if roleOf(p) == roleSynthesizedTestMain {
			return
		}
		path := basePath(p.PkgPath)
		if seen[path] {
			return
		}
		seen[path] = true
		class := classify(p.PkgPath)
		partitionSets[class] = append(partitionSets[class], path)
	})
	for _, list := range partitionSets {
		sort.Strings(list)
	}
	report.Partition = Partition{
		Owned:        partitionSets[profile.ClassOwned],
		TestOnly:     partitionSets[profile.ClassTestOnly],
		HardExcluded: partitionSets[profile.ClassHardExcluded],
		Unselected:   partitionSets[profile.ClassUnselected],
		ExternalStd:  partitionSets[profile.ClassExternalStd],
		ExternalMod:  partitionSets[profile.ClassExternalMod],
	}

	// Deduplicate contradiction edges (one per from/to/scope).
	edgeSeen := map[string]bool{}
	var uniqueEdges []Edge
	for _, e := range contradictions {
		key := e.From + "\x00" + e.To + "\x00" + e.Scope
		if !edgeSeen[key] {
			edgeSeen[key] = true
			uniqueEdges = append(uniqueEdges, e)
		}
	}
	sort.Slice(uniqueEdges, func(i, j int) bool {
		if uniqueEdges[i].From != uniqueEdges[j].From {
			return uniqueEdges[i].From < uniqueEdges[j].From
		}
		if uniqueEdges[i].To != uniqueEdges[j].To {
			return uniqueEdges[i].To < uniqueEdges[j].To
		}
		return uniqueEdges[i].Scope < uniqueEdges[j].Scope
	})
	report.Contradictions = uniqueEdges

	for _, entry := range external {
		report.External = append(report.External, *entry)
	}
	sort.Slice(report.External, func(i, j int) bool { return report.External[i].Path < report.External[j].Path })

	for _, entry := range perPackage {
		report.Packages = append(report.Packages, *entry)
	}
	sort.Slice(report.Packages, func(i, j int) bool { return report.Packages[i].Path < report.Packages[j].Path })

	return nil
}

func mergeFileStats(scope *ScopeReport, pkg *PackageReport, stats *fileStats, testScope bool) {
	scope.Declarations.Functions += stats.decls.Functions
	scope.Declarations.Methods += stats.decls.Methods
	scope.Declarations.BodylessFunctions += stats.decls.BodylessFunctions
	scope.Declarations.NamedTypes += stats.decls.NamedTypes
	scope.Declarations.Aliases += stats.decls.Aliases
	scope.Declarations.Constants += stats.decls.Constants
	scope.Declarations.Variables += stats.decls.Variables
	scope.Bodies += stats.bodies
	scope.Statements += stats.statements
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
	for k, v := range stats.directives {
		scope.Directives[k] += v
	}
	for k, v := range stats.astKinds {
		scope.ASTKinds[k] += v
	}
	if testScope {
		pkg.TestBodies += stats.bodies
	} else {
		pkg.Declarations.Functions += stats.decls.Functions
		pkg.Declarations.Methods += stats.decls.Methods
		pkg.Declarations.BodylessFunctions += stats.decls.BodylessFunctions
		pkg.Declarations.NamedTypes += stats.decls.NamedTypes
		pkg.Declarations.Aliases += stats.decls.Aliases
		pkg.Declarations.Constants += stats.decls.Constants
		pkg.Declarations.Variables += stats.decls.Variables
		pkg.Bodies += stats.bodies
		pkg.Statements += stats.statements
	}
}

func countLines(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	lines := bytes.Count(data, []byte{'\n'})
	if len(data) > 0 && data[len(data)-1] != '\n' {
		lines++
	}
	return lines, nil
}

// Write serializes the report deterministically.
func Write(report *Report, outPath string) error {
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(outPath, append(data, '\n'), 0o644)
}
