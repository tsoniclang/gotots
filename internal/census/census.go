// Package census produces the typed, reproducible source census that is the
// first gotots deliverable.
//
// The census runs in two explicit passes:
//
//  1. inventory (no syntax): the tracked Git tree defines the source
//     universe; the go list driver enriches it with build/package semantics
//     and toolchain evidence for external dependencies;
//  2. typed load (owned syntax only): owned and test-only roots are loaded
//     with full type information while dependencies are consumed as export
//     data — external implementation bodies are never parsed.
//
// Every analyzed file is reconciled byte-for-byte against the pinned commit
// tree, the two passes must agree on exact per-package file sets, and every
// record carries a validated unique identity. Aggregates are derived from
// records. The census fails closed on load errors, unknown AST kinds, dirty
// or injected source, missing external evidence, and identity collisions,
// and must never reclassify a unit because analysis failed.
package census

import (
	"crypto/sha256"
	"encoding/hex"
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
// Package is the semantic Go package identity (a black-box test file keeps
// its p_test identity); Owner is the package whose scope owns it.
type DeclarationRecord struct {
	ID         string `json:"id"`
	Package    string `json:"package"`
	Owner      string `json:"owner,omitempty"` // set when different from Package
	File       string `json:"file"`            // module-relative
	Kind       string `json:"kind"`            // func|method|type|alias|const|var
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

// FileRecord identifies one analyzed source file, verified against the
// pinned commit tree.
type FileRecord struct {
	Path    string `json:"path"` // module-relative
	Package string `json:"package"`
	Owner   string `json:"owner,omitempty"`
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
	Col       int    `json:"col"`
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
	Col       int    `json:"col"`
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
// packages directly imported by owned source. Pass-1 evidence is
// mandatory: a direct import with no inventory evidence fails the census.
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
	// Product externals reachable from owned production/test scope; the
	// remainder is reachable only through excluded or unselected source.
	ExternalProductClosure int `json:"externalProductClosure"`
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

// Environment is the machine-specific evidence report.
type Environment struct {
	SourceDir string          `json:"sourceDir"`
	Toolchain *goenv.Resolved `json:"toolchain"`
}

// Result bundles the deterministic reports and the machine evidence.
type Result struct {
	Inventory   *inventory.Inventory
	Report      *Report
	Environment *Environment
	sourceDir   string
}

// Run executes both census passes.
func Run(prof *profile.Profile, sourceDir string, buildProfileName string) (*Result, error) {
	build, err := prof.BuildProfileByName(buildProfileName)
	if err != nil {
		return nil, err
	}

	absSource, err := filepath.Abs(sourceDir)
	if err != nil {
		return nil, err
	}
	sourceDir = absSource

	// Toolchain first: the go executable's digest is verified before it is
	// ever executed, and the in-process Go frontend must match the pin.
	resolved, err := pinning.VerifyToolchain(prof.Pin)
	if err != nil {
		return nil, err
	}
	env := resolved.Environ(goenv.EnvOptions{
		GOOS:       build.GOOS,
		GOARCH:     build.GOARCH,
		GOAMD64:    build.GOAMD64,
		CgoEnabled: build.CgoEnabled,
	})

	checkout, err := pinning.VerifySource(prof.Pin, sourceDir)
	if err != nil {
		return nil, err
	}
	verified := checkout.Source
	verified.ToolchainVersion = prof.Pin.Toolchain.Version
	verified.GoExecutableSha256 = prof.Pin.Toolchain.GoExecutableSha256
	verified.GorootSrcDigest = prof.Pin.Toolchain.GorootSrcDigest

	inv, err := inventory.Run(prof, resolved, env, sourceDir, checkout.Tree)
	if err != nil {
		return nil, err
	}

	report := &Report{
		SchemaVersion: 3,
		Product:       prof.Product,
		BuildProfile:  *build,
		Pin:           *prof.Pin,
		Source:        verified,
		Production:    newScopeReport(),
		Test:          newScopeReport(),
	}
	fillPartition(inv, report)

	loaded, err := load(prof, env, sourceDir)
	if err != nil {
		return nil, err
	}
	if err := analyze(prof, inv, checkout.Tree, loaded, sourceDir, report); err != nil {
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

	return &Result{
		Inventory:   inv,
		Report:      report,
		Environment: &Environment{SourceDir: sourceDir, Toolchain: resolved},
		sourceDir:   sourceDir,
	}, nil
}

func fillPartition(inv *inventory.Inventory, report *Report) {
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
		if pkg.ReachableFromProduction || pkg.ReachableFromTest {
			report.Partition.ExternalProductClosure++
		}
	}
}

func load(prof *profile.Profile, env []string, sourceDir string) ([]*packages.Package, error) {
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
// loader's ForTest field plus explicit file-location evidence for the
// synthesized test binary — never package IDs or path-suffix parsing.
type packageRole int

const (
	roleProduction packageRole = iota
	roleInPackageTest
	roleExternalTest
	roleSynthesizedTestMain
)

func roleOf(p *packages.Package, sourceDir string) packageRole {
	if p.ForTest != "" {
		if p.PkgPath == p.ForTest {
			return roleInPackageTest
		}
		return roleExternalTest
	}
	if p.Name == "main" && len(p.GoFiles) > 0 {
		// The toolchain-synthesized test binary consists entirely of
		// generated files (its _testmain.go) outside the pinned checkout.
		// A genuine owned main package has in-tree files and remains
		// production source.
		allOutside := true
		for _, file := range p.GoFiles {
			if relative, err := filepath.Rel(sourceDir, file); err == nil && !strings.HasPrefix(relative, "..") {
				allOutside = false
				break
			}
		}
		if allOutside {
			return roleSynthesizedTestMain
		}
	}
	return roleProduction
}

// ownerPath is the package whose profile scope owns this variant: the
// package under test for test variants, the package itself otherwise.
func ownerPath(p *packages.Package) string {
	if p.ForTest != "" {
		return p.ForTest
	}
	return p.PkgPath
}

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
				Path:    relative,
				Package: p.PkgPath,
				Scope:   scopeName,
				Lines:   lines,
				Sha256:  hex.EncodeToString(digest[:]),
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
					edges = append(edges, Edge{
						From: owner, File: relative, To: importPath,
						Class: "hard-excluded", Category: category, Scope: scopeName,
					})
				case profile.ClassUnselected:
					edges = append(edges, Edge{
						From: owner, File: relative, To: importPath,
						Class: "unselected", Scope: scopeName,
					})
				case profile.ClassTestOnly:
					if !isTestScope {
						edges = append(edges, Edge{
							From: owner, File: relative, To: importPath,
							Class: "test-only", Scope: scopeName,
						})
					}
				}
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

// reconcileFileSets proves exact selected file-set equivalence between the
// pass-1 inventory and the pass-2 typed load for every owned and test-only
// package: production files, in-package tests, and black-box tests each
// match exactly.
func reconcileFileSets(inv *inventory.Inventory, goFiles, testFiles, xtestFiles map[string][]string) error {
	var problems []string
	compare := func(pkg, kind string, want, got []string) {
		sort.Strings(got)
		if len(want) != len(got) {
			problems = append(problems, fmt.Sprintf("%s: %s: inventory has %d files, typed pass analyzed %d", pkg, kind, len(want), len(got)))
			return
		}
		for i := range want {
			if want[i] != got[i] {
				problems = append(problems, fmt.Sprintf("%s: %s: inventory %s vs typed pass %s", pkg, kind, want[i], got[i]))
				return
			}
		}
	}
	for _, pkg := range inv.Module {
		class := profile.PackageClass(pkg.Class)
		if class != profile.ClassOwned && class != profile.ClassTestOnly {
			continue
		}
		if len(pkg.CgoFiles) > 0 {
			problems = append(problems, fmt.Sprintf("%s: cgo files selected but cgo analysis is unsupported", pkg.ImportPath))
		}
		compare(pkg.ImportPath, "go files", pkg.GoFiles, goFiles[pkg.ImportPath])
		compare(pkg.ImportPath, "in-package test files", pkg.TestGoFiles, testFiles[pkg.ImportPath])
		compare(pkg.ImportPath, "black-box test files", pkg.XTestGoFiles, xtestFiles[pkg.ImportPath])
	}
	if len(problems) > 0 {
		sort.Strings(problems)
		return fmt.Errorf("pass-1/pass-2 file reconciliation fails closed on %d mismatches:\n%s",
			len(problems), strings.Join(problems, "\n"))
	}
	return nil
}

// validateIdentities proves that every purported record identity is unique.
func validateIdentities(report *Report) error {
	var duplicates []string
	check := func(kind string, keys []string) {
		seen := map[string]bool{}
		for _, key := range keys {
			if seen[key] {
				duplicates = append(duplicates, kind+": "+key)
			}
			seen[key] = true
		}
	}

	declarationIDs := make([]string, len(report.Declarations))
	for i, d := range report.Declarations {
		declarationIDs[i] = d.ID
	}
	check("declaration", declarationIDs)

	filePaths := make([]string, len(report.Files))
	for i, f := range report.Files {
		filePaths[i] = f.Path
	}
	check("file", filePaths)

	directiveKeys := make([]string, len(report.Directives))
	for i, d := range report.Directives {
		directiveKeys[i] = fmt.Sprintf("%s:%d:%d %s", d.File, d.Line, d.Col, d.Directive)
	}
	check("directive", directiveKeys)

	rareKeys := make([]string, len(report.RareConstructs))
	for i, r := range report.RareConstructs {
		rareKeys[i] = fmt.Sprintf("%s:%d:%d %s", r.File, r.Line, r.Col, r.Construct)
	}
	check("rare-construct", rareKeys)

	if len(duplicates) > 0 {
		sort.Strings(duplicates)
		return fmt.Errorf("identity validation fails closed on %d duplicates:\n%s",
			len(duplicates), strings.Join(duplicates, "\n"))
	}
	return nil
}

func sortRecords(report *Report, edges []Edge, externalUse map[string]*ExternalUse) {
	edgeSeen := map[string]bool{}
	for _, e := range edges {
		key := e.From + "\x00" + e.File + "\x00" + e.To + "\x00" + e.Scope
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
		if a.File != b.File {
			return a.File < b.File
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
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		return a.Col < b.Col
	})
	sort.Slice(report.RareConstructs, func(i, j int) bool {
		a, b := report.RareConstructs[i], report.RareConstructs[j]
		if a.File != b.File {
			return a.File < b.File
		}
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		return a.Col < b.Col
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
