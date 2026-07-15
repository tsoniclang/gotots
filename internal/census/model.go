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
//
// This file owns the report data model; the pipeline stages live in
// run.go, load.go, analyze.go, reconcile.go, identity.go, aggregate.go,
// inspect.go, and publish.go.
package census

import (
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
	// Generated marks files following the toolchain's documented
	// generated-code convention (marker before the first non-comment,
	// non-blank text); GeneratedBy is the exact marker line, which names
	// the generator.
	Generated   bool   `json:"generated,omitempty"`
	GeneratedBy string `json:"generatedBy,omitempty"`
}

// DirectiveRecord is one compiler directive occurrence. Unknown directives
// are recorded with Known=false and block an authoritative result until
// they receive a reviewed disposition.
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

// ProfileAttestation records the exact corpus boundary that produced a
// census: the profile file's content hash plus the resolved scope roots,
// so two bundles from different boundaries are always distinguishable.
type ProfileAttestation struct {
	Hash              string              `json:"hash"`
	OwnedRoots        []string            `json:"ownedRoots"`
	TestOnlyRoots     []string            `json:"testOnlyRoots,omitempty"`
	HardExcludedRoots map[string][]string `json:"hardExcludedRoots,omitempty"`
	ToolingRoots      []string            `json:"toolingRoots,omitempty"`
}

// Report is the deterministic census output. It contains no machine paths;
// machine evidence lives in the environment report.
type Report struct {
	SchemaVersion int                     `json:"schemaVersion"`
	Product       string                  `json:"product"`
	Profile       ProfileAttestation      `json:"profile"`
	BuildProfile  profile.BuildProfile    `json:"buildProfile"`
	Pin           pinning.Pin             `json:"pin"`
	Source        *pinning.VerifiedSource `json:"source"`
	Partition     PartitionCounts         `json:"partition"`
	Production    *ScopeReport            `json:"ownedProduction"`
	Test          *ScopeReport            `json:"ownedTest"`
	// Blockers lists why this census cannot be an authoritative-success
	// result (unresolved contradictions, unknown directives). A census with
	// blockers is valid evidence but must not gate downstream generation.
	Blockers       []string              `json:"blockers,omitempty"`
	Contradictions []Edge                `json:"contradictions"`
	External       []ExternalUse         `json:"external"`
	Files          []FileRecord          `json:"files"`
	Declarations   []DeclarationRecord   `json:"declarations"`
	Directives     []DirectiveRecord     `json:"directives"`
	RareConstructs []RareConstructRecord `json:"rareConstructs"`
	// PredeclaredUniverse is the pinned toolchain's complete predeclared
	// object list, verified against the reviewed contract.
	PredeclaredUniverse []string `json:"predeclaredUniverse"`
	// ExternalObligations are the object-level declaration/stub obligations
	// owned source places on external packages.
	ExternalObligations []ExternalObligation `json:"externalObligations"`
	// TestFunctions is the test-discovery ledger for owned test scope.
	TestFunctions []TestFunctionRecord `json:"testFunctions"`
}

// Environment is the machine-specific evidence report: the resolved
// toolchain locations, the source checkout path, and the exact child
// environment used for every toolchain invocation.
type Environment struct {
	SourceDir string          `json:"sourceDir"`
	Toolchain *goenv.Resolved `json:"toolchain"`
	ChildEnv  []string        `json:"childEnv"`
}

// Result bundles the deterministic reports and the machine evidence.
type Result struct {
	Inventory   *inventory.Inventory
	Report      *Report
	Shapes      *DeclarationShapes
	Externals   *ExternalContract
	Environment *Environment
	sourceDir   string
}
