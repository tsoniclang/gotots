// Package census produces the typed, reproducible source census that is the
// first gotots deliverable.
//
// The census runs in two explicit passes:
//
//  1. inventory (no syntax): a complete module/file/dependency enumeration
//     via the go list driver, so the partition provably covers every package
//     in the pinned module and every external dependency carries toolchain
//     evidence (Standard/Module), never path-shape guessing;
//  2. typed load (owned syntax only): owned and test-only roots are loaded
//     with full type information while dependencies are consumed as export
//     data — external implementation bodies are never parsed.
//
// All aggregate numbers are derived from identity-bearing records
// (declarations, files, directives, rare constructs). The census fails
// closed on load errors, unknown AST kinds, dirty checkouts, or a pass-1 /
// pass-2 owned-set mismatch, and must never reclassify a unit because
// analysis failed.
package census

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"

	"github.com/tsoniclang/gotots/internal/goenv"
	"github.com/tsoniclang/gotots/internal/inventory"
	"github.com/tsoniclang/gotots/internal/pinning"
	"github.com/tsoniclang/gotots/internal/profile"
)

// DeclarationRecord identifies one top-level declaration with source span
// and, for bodies, a content hash usable later for drift detection.
type DeclarationRecord struct {
	ID         string `json:"id"` // pkg::file::kind::name
	Package    string `json:"package"`
	File       string `json:"file"` // module-relative
	Kind       string `json:"kind"` // func|method|type|alias|const|var
	Name       string `json:"name"`
	Receiver   string `json:"receiver,omitempty"` // methods: base type name
	Exported   bool   `json:"exported"`
	Scope      string `json:"scope"` // production|test
	StartLine  int    `json:"startLine"`
	EndLine    int    `json:"endLine"`
	HasBody    bool   `json:"hasBody,omitempty"`
	Statements int    `json:"statements,omitempty"`
	BodySha256 string `json:"bodySha256,omitempty"`
}

// FileRecord identifies one analyzed source file.
type FileRecord struct {
	Path    string `json:"path"` // module-relative
	Package string `json:"package"`
	Scope   string `json:"scope"` // production|test
	Lines   int    `json:"lines"`
	Sha256  string `json:"sha256"`
}

// DirectiveRecord is one compiler directive occurrence. Unknown directives
// are recorded with Known=false and must receive a reviewed disposition
// before generation.
type DirectiveRecord struct {
	File      string `json:"file"`
	Line      int    `json:"line"`
	Directive string `json:"directive"`
	Known     bool   `json:"known"`
}

// RareConstructRecord pins the exact location of low-volume, high-impact
// constructs (concurrency, goto, unsafe, full slice expressions) that drive
// manual-ownership and lowering-design decisions.
type RareConstructRecord struct {
	Construct string `json:"construct"`
	File      string `json:"file"`
	Line      int    `json:"line"`
}

// Edge records one dependency from an owned package to a package it must
// not depend on under the profile.
type Edge struct {
	From     string `json:"from"`
	File     string `json:"file"`
	To       string `json:"to"`
	Class    string `json:"class"` // hard-excluded | unselected | test-only
	Category string `json:"category,omitempty"`
	Scope    string `json:"scope"` // production | test
}

// ExternalUse merges pass-1 evidence with pass-2 importer counts for
// packages directly imported by owned source.
type ExternalUse struct {
	Path                string `json:"path"`
	Standard            bool   `json:"standard"`
	ModulePath          string `json:"modulePath,omitempty"`
	ModuleVersion       string `json:"moduleVersion,omitempty"`
	ProductionImporters int    `json:"productionImporters"`
	TestImporters       int    `json:"testImporters"`
	BlankImports        int    `json:"blankImports"`
	DotImports          int    `json:"dotImports"`
}

// DeclCounts aggregates declarations by kind (derived from records).
type DeclCounts struct {
	Functions         int `json:"functions"`
	Methods           int `json:"methods"`
	BodylessFunctions int `json:"bodylessFunctions"`
	NamedTypes        int `json:"namedTypes"`
	Aliases           int `json:"aliases"`
	Constants         int `json:"constants"`
	Variables         int `json:"variables"`
}

// ScopeReport aggregates one non-overlapping source scope.
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

// PartitionCounts summarizes the complete pass-1 partition.
type PartitionCounts struct {
	Owned        int `json:"owned"`
	TestOnly     int `json:"ownedTestSupport"`
	HardExcluded int `json:"hardExcluded"`
	Unselected   int `json:"unselected"`
	ExternalStd  int `json:"externalStd"`
	ExternalMod  int `json:"externalModule"`
}

// Report is the deterministic census output. It contains no machine paths;
// machine evidence lives in the environment report.
type Report struct {
	SchemaVersion  int                     `json:"schemaVersion"`
	Product        string                  `json:"product"`
	BuildProfile   profile.BuildProfile    `json:"buildProfile"`
	Pin            pinning.Pin             `json:"pin"`
	Source         *pinning.VerifiedSource `json:"source"`
	Partition      PartitionCounts         `json:"partition"`
	Production     *ScopeReport            `json:"ownedProduction"`
	Test           *ScopeReport            `json:"ownedTest"`
	Contradictions []Edge                  `json:"contradictions"`
	External       []ExternalUse           `json:"external"`
	Files          []FileRecord            `json:"files"`
	Declarations   []DeclarationRecord     `json:"declarations"`
	Directives     []DirectiveRecord       `json:"directives"`
	RareConstructs []RareConstructRecord   `json:"rareConstructs"`
}

// Result bundles the two deterministic reports and the machine evidence.
type Result struct {
	Inventory   *inventory.Inventory
	Report      *Report
	Environment *goenv.Resolved
}

// Run executes both census passes.
func Run(prof *profile.Profile, sourceDir string, buildProfileName string) (*Result, error) {
	build, err := prof.BuildProfileByName(buildProfileName)
	if err != nil {
		return nil, err
	}
	if len(build.Tags) > 0 {
		// Tag plumbing (BuildFlags for both passes) is untested; refuse
		// rather than silently produce a wrong file selection.
		return nil, fmt.Errorf("build profile %q: build tags are not supported yet", build.Name)
	}

	resolved, err := goenv.Resolve()
	if err != nil {
		return nil, err
	}
	env := resolved.Environ(build.GOOS, build.GOARCH, false)

	verified, err := pinning.Verify(prof.Pin, sourceDir, resolved)
	if err != nil {
		return nil, err
	}

	inv, err := inventory.Run(prof, resolved, env, sourceDir)
	if err != nil {
		return nil, err
	}

	report := &Report{
		SchemaVersion: 2,
		Product:       prof.Product,
		BuildProfile:  *build,
		Pin:           *prof.Pin,
		Source:        verified,
		Production:    newScopeReport(),
		Test:          newScopeReport(),
	}
	fillPartition(prof, inv, report)

	loaded, err := load(prof, resolved, env, sourceDir)
	if err != nil {
		return nil, err
	}
	if err := analyze(prof, inv, loaded, sourceDir, report); err != nil {
		return nil, err
	}

	// Extraction must not have mutated the pinned source.
	clean, err := pinning.CheckClean(sourceDir)
	if err != nil {
		return nil, err
	}
	if !clean {
		return nil, fmt.Errorf("source checkout %s is dirty after loading; extraction mutated pinned input", sourceDir)
	}
	report.Source.CleanAfterLoad = true

	return &Result{Inventory: inv, Report: report, Environment: resolved}, nil
}

func fillPartition(prof *profile.Profile, inv *inventory.Inventory, report *Report) {
	for _, pkg := range inv.Module {
		switch profile.PackageClass(pkg.Class) {
		case profile.ClassOwned:
			report.Partition.Owned++
		case profile.ClassTestOnly:
			report.Partition.TestOnly++
		case profile.ClassHardExcluded:
			report.Partition.HardExcluded++
		case profile.ClassUnselected:
			report.Partition.Unselected++
		}
	}
	for _, pkg := range inv.External {
		if pkg.Standard {
			report.Partition.ExternalStd++
		} else {
			report.Partition.ExternalMod++
		}
	}
}

func load(prof *profile.Profile, resolved *goenv.Resolved, env []string, sourceDir string) ([]*packages.Package, error) {
	patternRoots := append(append([]string{}, prof.OwnedRoots...), prof.TestOnlyRoots...)
	sort.Strings(patternRoots)
	patterns := make([]string, 0, len(patternRoots))
	for _, root := range patternRoots {
		patterns = append(patterns, "./"+root+"/...")
	}

	// NeedDeps is deliberately absent: with it, x/tools parses and
	// type-checks every transitive dependency from source, violating the
	// external-body boundary. Without it, syntax and type information are
	// produced for the root packages only; dependencies are consumed as
	// compiled export data.
	config := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles |
			packages.NeedImports | packages.NeedModule | packages.NeedForTest |
			packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo,
		Dir:   sourceDir,
		Env:   env,
		Tests: true,
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

// packageRole distinguishes the go/packages test variants using the
// loader's ForTest field, never package ID or path-suffix parsing.
type packageRole int

const (
	roleProduction packageRole = iota
	roleTestVariant
	roleSynthesizedTestMain
)

func roleOf(p *packages.Package) packageRole {
	if p.ForTest == "" {
		if strings.HasSuffix(p.PkgPath, ".test") {
			return roleSynthesizedTestMain // defensive: drivers vary here
		}
		return roleProduction
	}
	if p.Name == "main" {
		return roleSynthesizedTestMain
	}
	return roleTestVariant
}

// classificationPath is the package path used for profile classification:
// test variants classify as the package under test.
func classificationPath(p *packages.Package) string {
	if p.ForTest != "" {
		return p.ForTest
	}
	return p.PkgPath
}

func analyze(prof *profile.Profile, inv *inventory.Inventory, loaded []*packages.Package, sourceDir string, report *Report) error {
	externalIndex := inv.ExternalIndex()
	externalUse := map[string]*ExternalUse{}
	var edges []Edge

	// Deterministic iteration order.
	roots := append([]*packages.Package{}, loaded...)
	sort.Slice(roots, func(i, j int) bool { return roots[i].ID < roots[j].ID })

	loadedOwned := map[string]bool{}

	for _, p := range roots {
		role := roleOf(p)
		if role == roleSynthesizedTestMain {
			continue
		}
		class, _ := prof.Classify(classificationPath(p))
		if class != profile.ClassOwned && class != profile.ClassTestOnly {
			continue
		}
		if role == roleProduction {
			loadedOwned[p.PkgPath] = true
		}

		// Test-only packages are owned test-support source: every file they
		// contain belongs to the owned-test scope. Owned packages split by
		// variant, with the in-package test variant contributing only its
		// _test.go files (the rest belong to the production variant).
		isTestScope := class == profile.ClassTestOnly || role == roleTestVariant
		scope := report.Production
		scopeName := "production"
		if isTestScope {
			scope = report.Test
			scopeName = "test"
		}

		countedFiles := 0
		for _, file := range p.Syntax {
			filename := p.Fset.Position(file.Pos()).Filename
			if role == roleTestVariant && p.PkgPath == p.ForTest && !strings.HasSuffix(filename, "_test.go") {
				continue // production files re-listed in the in-package test variant
			}
			relative, err := filepath.Rel(sourceDir, filename)
			if err != nil || strings.HasPrefix(relative, "..") {
				return fmt.Errorf("file %s is outside the source checkout", filename)
			}
			relative = filepath.ToSlash(relative)
			countedFiles++

			data, err := os.ReadFile(filename)
			if err != nil {
				return err
			}
			lines := countLines(data)
			digest := sha256.Sum256(data)
			report.Files = append(report.Files, FileRecord{
				Path:    relative,
				Package: classificationPath(p),
				Scope:   scopeName,
				Lines:   lines,
				Sha256:  hex.EncodeToString(digest[:]),
			})
			scope.Files++
			scope.Lines += lines

			stats, err := inspectFile(p, file, relative, scopeName, data)
			if err != nil {
				return err
			}
			report.Declarations = append(report.Declarations, stats.declarations...)
			report.Directives = append(report.Directives, stats.directives...)
			report.RareConstructs = append(report.RareConstructs, stats.rare...)
			mergeAggregates(scope, stats)

			for _, importSpec := range file.Imports {
				importPath := strings.Trim(importSpec.Path.Value, `"`)
				importClass, category := prof.Classify(importPath)
				switch importClass {
				case profile.ClassExternal:
					use := externalUse[importPath]
					if use == nil {
						use = &ExternalUse{Path: importPath}
						if evidence := externalIndex[importPath]; evidence != nil {
							use.Standard = evidence.Standard
							use.ModulePath = evidence.ModulePath
							use.ModuleVersion = evidence.ModuleVersion
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
					edges = append(edges, Edge{
						From: classificationPath(p), File: relative, To: importPath,
						Class: "hard-excluded", Category: category, Scope: scopeName,
					})
				case profile.ClassUnselected:
					edges = append(edges, Edge{
						From: classificationPath(p), File: relative, To: importPath,
						Class: "unselected", Scope: scopeName,
					})
				case profile.ClassTestOnly:
					if !isTestScope {
						edges = append(edges, Edge{
							From: classificationPath(p), File: relative, To: importPath,
							Class: "test-only", Scope: scopeName,
						})
					}
				}
			}
		}
		if countedFiles > 0 && (role == roleProduction || p.PkgPath != p.ForTest) {
			scope.Packages++
		}
	}

	// Completeness cross-check: every owned/test-only package pass 1 found
	// must have been loaded and analyzed by pass 2.
	var missing []string
	for _, pkg := range inv.Module {
		class := profile.PackageClass(pkg.Class)
		if (class == profile.ClassOwned || class == profile.ClassTestOnly) && len(pkg.GoFiles) > 0 {
			if !loadedOwned[pkg.ImportPath] {
				missing = append(missing, pkg.ImportPath)
			}
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return fmt.Errorf("census fails closed: %d owned packages in the inventory were not loaded by the typed pass:\n%s",
			len(missing), strings.Join(missing, "\n"))
	}

	sortRecords(report, edges, externalUse)
	deriveDeclarationAggregates(report)
	return nil
}

func sortRecords(report *Report, edges []Edge, externalUse map[string]*ExternalUse) {
	edgeSeen := map[string]bool{}
	for _, e := range edges {
		key := e.From + "\x00" + e.To + "\x00" + e.Scope
		if !edgeSeen[key] {
			edgeSeen[key] = true
			report.Contradictions = append(report.Contradictions, e)
		}
	}
	sort.Slice(report.Contradictions, func(i, j int) bool {
		a, b := report.Contradictions[i], report.Contradictions[j]
		if a.From != b.From {
			return a.From < b.From
		}
		if a.To != b.To {
			return a.To < b.To
		}
		return a.Scope < b.Scope
	})

	for _, use := range externalUse {
		report.External = append(report.External, *use)
	}
	sort.Slice(report.External, func(i, j int) bool { return report.External[i].Path < report.External[j].Path })

	sort.Slice(report.Files, func(i, j int) bool { return report.Files[i].Path < report.Files[j].Path })
	sort.Slice(report.Declarations, func(i, j int) bool {
		a, b := report.Declarations[i], report.Declarations[j]
		if a.ID != b.ID {
			return a.ID < b.ID
		}
		return a.StartLine < b.StartLine
	})
	sort.Slice(report.Directives, func(i, j int) bool {
		a, b := report.Directives[i], report.Directives[j]
		if a.File != b.File {
			return a.File < b.File
		}
		return a.Line < b.Line
	})
	sort.Slice(report.RareConstructs, func(i, j int) bool {
		a, b := report.RareConstructs[i], report.RareConstructs[j]
		if a.File != b.File {
			return a.File < b.File
		}
		return a.Line < b.Line
	})
}

// deriveDeclarationAggregates computes every DeclCounts/Bodies/Statements
// total from the declaration records so aggregates cannot drift from the
// identity-bearing data.
func deriveDeclarationAggregates(report *Report) {
	scopeOf := func(name string) *ScopeReport {
		if name == "test" {
			return report.Test
		}
		return report.Production
	}
	for i := range report.Declarations {
		d := &report.Declarations[i]
		scope := scopeOf(d.Scope)
		switch d.Kind {
		case "func":
			scope.Declarations.Functions++
		case "method":
			scope.Declarations.Methods++
		case "type":
			scope.Declarations.NamedTypes++
		case "alias":
			scope.Declarations.Aliases++
		case "const":
			scope.Declarations.Constants++
		case "var":
			scope.Declarations.Variables++
		}
		if d.Kind == "func" || d.Kind == "method" {
			if d.HasBody {
				scope.Bodies++
				scope.Statements += d.Statements
			} else {
				scope.Declarations.BodylessFunctions++
			}
		}
	}
	fileScopes := make(map[string]string, len(report.Files))
	for _, f := range report.Files {
		fileScopes[f.Path] = f.Scope
	}
	for _, directive := range report.Directives {
		scope := fileScopes[directive.File]
		if scope == "" {
			scope = "production"
		}
		scopeOf(scope).Directives[directive.Directive]++
	}
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

// WriteReports serializes the deterministic reports and the machine
// evidence into outDir. inventory.json and census.json are byte-stable
// across runs; environment.json holds machine-specific paths.
func WriteReports(result *Result, outDir string) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	write := func(name string, value any) error {
		data, err := json.MarshalIndent(value, "", "  ")
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(outDir, name), append(data, '\n'), 0o644)
	}
	if err := write("inventory.json", result.Inventory); err != nil {
		return err
	}
	if err := write("census.json", result.Report); err != nil {
		return err
	}
	return write("environment.json", result.Environment)
}
