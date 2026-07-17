package translate

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"go/ast"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"

	"github.com/tsoniclang/gotots/internal/goid"
	"github.com/tsoniclang/gotots/internal/ir"
	"github.com/tsoniclang/gotots/internal/profile"
	"github.com/tsoniclang/gotots/internal/typedload"
)

// ProbeResult measures how much of the real corpus the current reviewed
// subset translates, and ranks exactly what blocks the rest. It is
// diagnostic evidence for roadmap ordering — never authoritative
// generation, which requires complete package coverage.
type ProbeResult struct {
	// Diagnostic marks this report as a development probe, never
	// acceptance evidence: it carries no body ledger or manifest and is
	// not gate output.
	Diagnostic bool `json:"diagnostic"`
	// SourceRevision and ProfileHash attest the probed inputs.
	SourceRevision string `json:"sourceRevision,omitempty"`
	ProfileHash    string `json:"profileHash,omitempty"`
	Packages       int    `json:"packages"`
	Bodies         int    `json:"bodies"`
	// IRAdmitted counts bodies with complete typed semantic IR — the
	// ir-admitted evidence stage. It is NOT module-retained coverage: a
	// body can be ir-admitted yet removed from every runnable module by
	// package withholding. See ModuleRetainedPackages / WithheldPackages
	// and spec 00 Translation Evidence Stages.
	IRAdmitted       int            `json:"irAdmitted"`
	Blocked          int            `json:"blocked"`
	BlockerHistogram map[string]int `json:"blockerHistogram"`
	// ConstructHistogram counts the raw (unnormalized) construct
	// spellings, ranking the exact shapes inside each blocker class.
	ConstructHistogram map[string]int `json:"constructHistogram"`
	// PackagesFullyTranslated lists packages whose COMPLETE declaration
	// set translates — verified by running the package translation, not
	// just body IR construction.
	PackagesFullyTranslated []string `json:"packagesFullyTranslated"`
	// PackagesBodyOnly lists packages where every body builds but a
	// declaration-level construct (initializer, type, var) still blocks
	// full translation, with the blocking diagnostic.
	PackagesBodyOnly []string `json:"packagesBodyOnly"`
	// PerPackage maps package path -> translated/total.
	PerPackage map[string]string `json:"perPackage"`
	// PackageBlockers lists, for packages close to full body coverage
	// (at most three blocked bodies), each blocked body's sites — the
	// highest-leverage targets for package-level completion.
	PackageBlockers map[string][]string `json:"packageBlockers"`
	// ExternalRefs counts blocked references per external function or
	// method (pkg.Name) — the evidence ranking emulation-layer priorities.
	ExternalRefs map[string]int `json:"externalRefs"`
	// PerBodyState maps each body's canonical identity to its probe
	// classification ("generated" or "unimplemented"), the join key for
	// reconciling probe results against corpus support ledgers.
	PerBodyState map[string]string `json:"perBodyState"`
	// ClassInventory summarizes every unsupported class blocking the
	// corpus: its disposition category (the kind of solution it needs),
	// the shared root abstraction that would clear the whole class, and
	// the exact unique-unit and raw-site counts — the reusable-solution
	// roadmap, so no repeated class is silently routed to manual bodies.
	ClassInventory []ClassRow `json:"classInventory"`
	// UnimplementedUnits is one machine-readable row per unimplemented
	// body, carrying EVERY unsupported site (not only the first) with its
	// classification — the exhaustive residual accounting.
	UnimplementedUnits []UnitInventory `json:"unimplementedUnits"`
}

// ClassRow is one unsupported class's residual-inventory summary.
type ClassRow struct {
	Class    string `json:"class"`
	Category string `json:"category"`
	Root     string `json:"rootAbstraction"`
	Units    int    `json:"units"`
	Sites    int    `json:"sites"`
}

// UnitInventory is one unimplemented body with every unsupported site.
type UnitInventory struct {
	ID    string          `json:"id"`
	Sites []InventorySite `json:"sites"`
}

// InventorySite is one classified unsupported site.
type InventorySite struct {
	Class     string `json:"class"`
	Category  string `json:"category"`
	Construct string `json:"construct"`
	File      string `json:"file"`
	Line      int    `json:"line"`
}

// disposition names the KIND of solution a class of unsupported site
// requires and the shared root abstraction that would clear it.
type disposition struct{ category, root string }

// Disposition categories — the closed set the inventory reports.
const (
	catLanguageLowering  = "language-lowering"
	catRepresentation    = "representation-planning"
	catExternalContract  = "external-typed-contract"
	catProductPolicy     = "product-policy"
	catUnclassified      = "unclassified"
	rootUnclassifiedNote = "UNCLASSIFIED: this construct has no reviewed disposition; classify it explicitly rather than defaulting it"
)

// Shared root abstractions each category's clearing work threads through.
const (
	rootLangType     = "type universe: reviewed lowering for the remaining named-type forms"
	rootLangGenerics = "generic value semantics: per-instantiation copy operation threading"
	rootLangConvert  = "type conversion: same-underlying and generic-parameter conversion lowering"
	rootLangPointer  = "pointer cells: stable storage for the remaining address-taken locations"
	rootLangDispatch = "interface dispatch: closed-union lowering for the remaining boxing/assertion forms"
	rootLangDecl     = "local declarations: on-the-fly type-universe registration"
	rootLangControl  = "control flow: reviewed lowering for the remaining statement forms"
	rootLangBuiltin  = "builtin lowering: reviewed carrier operations for the remaining builtins"
	rootRepresent    = "keyed-map: per-instantiation static key encoding (uniform keyed carrier)"
	rootExternal     = "external contract surface: typed obligation/stub, never a manual caller body"
	rootConcurrency  = "concurrency: one explicit TS-Go concurrency product policy"
)

// dispositionByConstruct is the TOTAL classification of the closed
// construct-key set (ir.ClassConstructKeys). Every registered construct
// maps to an explicitly reviewed category — there is no substring matching
// and no catch-all default that would make an unreviewed class look like a
// simple language lowering. A construct outside this map (a diagnostic
// carrying free text with no registered prefix) classifies as
// "unclassified", surfaced honestly rather than silently absorbed.
var dispositionByConstruct = map[string]disposition{
	// Named-type and value forms not yet lowered.
	"non-basic type":          {catLanguageLowering, rootLangType},
	"basic type":              {catLanguageLowering, rootLangType},
	"interface type":          {catLanguageLowering, rootLangDispatch},
	"array type":              {catLanguageLowering, rootLangType},
	"struct type":             {catLanguageLowering, rootLangType},
	"type":                    {catLanguageLowering, rootLangType},
	"type parameter":          {catLanguageLowering, rootLangGenerics},
	"constant of type":        {catLanguageLowering, rootLangType},
	"zero value of":           {catLanguageLowering, rootLangType},
	"nil of type":             {catLanguageLowering, rootLangType},
	"interface value of type": {catLanguageLowering, rootLangDispatch},
	"string constant with":    {catLanguageLowering, rootLangType},
	"generic function instantiated with an unreviewed type argument": {catLanguageLowering, rootLangGenerics},
	// Pointers.
	"pointer to non-named type":  {catLanguageLowering, rootLangPointer},
	"pointer to non-struct type": {catLanguageLowering, rootLangPointer},
	"dereference of":             {catLanguageLowering, rootLangPointer},
	"address of":                 {catLanguageLowering, rootLangPointer},
	// Representation planning.
	"map key type": {catRepresentation, rootRepresent},
	// Concurrency — one product policy.
	"channel type":           {catProductPolicy, rootConcurrency},
	"goroutine statement":    {catProductPolicy, rootConcurrency},
	"select statement":       {catProductPolicy, rootConcurrency},
	"channel send statement": {catProductPolicy, rootConcurrency},
	// External contract surfaces (references reaching outside the unit).
	"pointer to type outside the translated unit:":    {catExternalContract, rootExternal},
	"call outside the translated unit":                {catExternalContract, rootExternal},
	"method call outside the translated unit":         {catExternalContract, rootExternal},
	"method value outside the translated unit (":      {catExternalContract, rootExternal},
	"method expression outside the translated unit (": {catExternalContract, rootExternal},
	"field access on":                                 {catExternalContract, rootExternal},
	"composite literal of":                            {catExternalContract, rootExternal},
	"pointer-receiver method value on":                {catExternalContract, rootExternal},
	// Selectors, calls, operators.
	"identifier":         {catLanguageLowering, rootLangType},
	"call of":            {catLanguageLowering, rootLangDispatch},
	"index on":           {catLanguageLowering, rootLangType},
	"nil comparison on":  {catLanguageLowering, rootLangDispatch},
	"operator":           {catLanguageLowering, rootLangType},
	"conversion from":    {catLanguageLowering, rootLangConvert},
	"non-field selector": {catLanguageLowering, rootLangType},
	"method call on":     {catLanguageLowering, rootLangDispatch},
	"equality on":        {catLanguageLowering, rootLangDispatch},
	"ordering on":        {catLanguageLowering, rootLangType},
	"inc/dec of":         {catLanguageLowering, rootLangType},
	// Builtins.
	"len of":       {catLanguageLowering, rootLangBuiltin},
	"cap of":       {catLanguageLowering, rootLangBuiltin},
	"make of":      {catLanguageLowering, rootLangBuiltin},
	"builtin":      {catLanguageLowering, rootLangBuiltin},
	"clear of":     {catLanguageLowering, rootLangBuiltin},
	"append to":    {catLanguageLowering, rootLangBuiltin},
	"copy between": {catLanguageLowering, rootLangBuiltin},
	"panic with":   {catLanguageLowering, rootLangControl},
	// Slices and ranges.
	"reslice of": {catLanguageLowering, rootLangType},
	"range over": {catLanguageLowering, rootLangControl},
	// Switches and assertions.
	"switch tag of":     {catLanguageLowering, rootLangControl},
	"switch case of":    {catLanguageLowering, rootLangControl},
	"type assertion on": {catLanguageLowering, rootLangDispatch},
	"type switch on":    {catLanguageLowering, rootLangDispatch},
	// Statements and assignments.
	"expression statement":           {catLanguageLowering, rootLangControl},
	"assignment to":                  {catLanguageLowering, rootLangControl},
	"compound assignment to":         {catLanguageLowering, rootLangControl},
	"indexed compound assignment on": {catLanguageLowering, rootLangControl},
	"field assignment on":            {catLanguageLowering, rootLangControl},
	"branch":                         {catLanguageLowering, rootLangControl},
	"package-level":                  {catLanguageLowering, rootLangDecl},
}

// classifySite maps one normalized unsupported-site class (ir.ClassOf's
// "CODE: <construct>") to its reviewed disposition. It matches the
// construct portion EXACTLY against the closed classification map — never
// a substring — and returns "unclassified" for a construct outside the
// map, so an unreviewed class is surfaced honestly instead of being
// silently deferred to a language lowering.
func classifySite(class string) (category, root string) {
	construct := class
	if _, rest, found := strings.Cut(class, ": "); found {
		construct = rest
	}
	if d, ok := dispositionByConstruct[construct]; ok {
		return d.category, d.root
	}
	return catUnclassified, rootUnclassifiedNote
}

// Probe loads the owned corpus under the profile and attempts IR building
// for every production function body.
func Probe(prof *profile.Profile, env []string, sourceDir string) (*ProbeResult, error) {
	loaded, err := typedload.Load(prof, env, sourceDir)
	if err != nil {
		return nil, err
	}

	result := &ProbeResult{
		Diagnostic:         true,
		PerBodyState:       map[string]string{},
		SourceRevision:     prof.Pin.Revision,
		ProfileHash:        prof.Hash,
		BlockerHistogram:   map[string]int{},
		ConstructHistogram: map[string]int{},
		PerPackage:         map[string]string{},
		PackageBlockers:    map[string][]string{},
		ExternalRefs:       map[string]int{},
	}
	sort.Slice(loaded, func(i, j int) bool { return loaded[i].ID < loaded[j].ID })

	// The unit is every owned production package: cross-package references
	// among them resolve to co-generated modules.
	owned := func(p *packages.Package) bool {
		if typedload.RoleOf(p, sourceDir) != typedload.RoleProduction {
			return false
		}
		class, _ := prof.Classify(p.PkgPath)
		return class == profile.ClassOwned
	}
	var ownedPaths []string
	var ownedPackages []*packages.Package
	for _, p := range loaded {
		if owned(p) {
			ownedPaths = append(ownedPaths, p.PkgPath)
			ownedPackages = append(ownedPackages, p)
		}
	}
	unit := ir.NewScope(ownedPaths...)
	collectGenericInstances(unit, ownedPackages)

	for _, p := range loaded {
		if !owned(p) {
			continue
		}
		result.Packages++
		packageBodies, packageTranslated := 0, 0
		var packageSites []string

		for _, file := range p.Syntax {
			filename := p.Fset.Position(file.Pos()).Filename
			relative, relErr := filepath.Rel(sourceDir, filename)
			if relErr != nil || strings.HasPrefix(relative, "..") {
				continue
			}
			source, readErr := os.ReadFile(filename)
			if readErr != nil {
				return nil, readErr
			}
			for _, decl := range file.Decls {
				funcDecl, isFunc := decl.(*ast.FuncDecl)
				if !isFunc || funcDecl.Body == nil {
					continue
				}
				result.Bodies++
				packageBodies++
				function, err := probeFunc(p, sourceDir, unit, source, funcDecl)
				if err != nil {
					return nil, err
				}
				result.PerBodyState[function.ID] = string(function.Support)
				if function.Support == ir.SupportUnimplemented {
					result.Blocked++
					unitInv := UnitInventory{ID: function.ID}
					for _, site := range function.Sites {
						result.BlockerHistogram[site.Class]++
						result.ConstructHistogram[site.Construct]++
						category, _ := classifySite(site.Class)
						unitInv.Sites = append(unitInv.Sites, InventorySite{
							Class: site.Class, Category: category, Construct: site.Construct,
							File: site.Span.File, Line: site.Span.Line})
						packageSites = append(packageSites,
							fmt.Sprintf("%s: %s (%s:%d)", function.ID, site.Construct, site.Span.File, site.Span.Line))
						if ref, isExternal := externalRefOfSite(site); isExternal {
							result.ExternalRefs[ref]++
						}
					}
					result.UnimplementedUnits = append(result.UnimplementedUnits, unitInv)
				} else {
					result.IRAdmitted++
					packageTranslated++
				}
			}
		}
		if packageBodies > 0 {
			result.PerPackage[p.PkgPath] = fmt.Sprintf("%d/%d", packageTranslated, packageBodies)
			if packageTranslated == packageBodies {
				result.PackagesFullyTranslated = append(result.PackagesFullyTranslated, p.PkgPath)
			}
			if blocked := packageBodies - packageTranslated; blocked > 0 && blocked <= 3 {
				result.PackageBlockers[p.PkgPath] = packageSites
			}
		}
	}
	sort.Strings(result.PackagesFullyTranslated)

	// Body IR alone is not package translatability: initializers, type
	// declarations, and package variables all count. Every candidate is
	// verified by actually translating the package.
	byPath := map[string]*packages.Package{}
	for _, p := range loaded {
		if owned(p) {
			byPath[p.PkgPath] = p
		}
	}
	verified := make([]string, 0, len(result.PackagesFullyTranslated))
	for _, candidate := range result.PackagesFullyTranslated {
		throwaway := &Generated{Files: map[string]string{}, Ownership: map[string]string{}, Withheld: map[string]string{}}
		var emitters []func() error
		if err := translatePackage(throwaway, byPath[candidate], sourceDir, unit, Options{}, &emitters); err != nil {
			result.PackagesBodyOnly = append(result.PackagesBodyOnly, candidate+": "+firstLine(err.Error()))
			continue
		}
		if reason, withheld := throwaway.Withheld[candidate]; withheld {
			blocking := reason
			for _, support := range throwaway.Support {
				if support.State == ir.SupportUnimplemented && len(support.Sites) > 0 {
					blocking = support.Sites[0].Class
					break
				}
			}
			result.PackagesBodyOnly = append(result.PackagesBodyOnly, candidate+": "+blocking)
			continue
		}
		verified = append(verified, candidate)
	}
	result.PackagesFullyTranslated = verified
	result.ClassInventory = buildClassInventory(result)
	return result, nil
}

// buildClassInventory summarizes every unsupported class: its disposition
// category and root abstraction, its raw site count, and the number of
// DISTINCT units it blocks — sorted most-blocking first.
func buildClassInventory(result *ProbeResult) []ClassRow {
	unitsByClass := map[string]map[string]bool{}
	for _, unitInv := range result.UnimplementedUnits {
		for _, site := range unitInv.Sites {
			if unitsByClass[site.Class] == nil {
				unitsByClass[site.Class] = map[string]bool{}
			}
			unitsByClass[site.Class][unitInv.ID] = true
		}
	}
	rows := make([]ClassRow, 0, len(result.BlockerHistogram))
	for class, sites := range result.BlockerHistogram {
		category, root := classifySite(class)
		rows = append(rows, ClassRow{
			Class: class, Category: category, Root: root,
			Units: len(unitsByClass[class]), Sites: sites})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Sites != rows[j].Sites {
			return rows[i].Sites > rows[j].Sites
		}
		return rows[i].Class < rows[j].Class
	})
	return rows
}

func probeFunc(p *packages.Package, sourceDir string, unit ir.Scope, source []byte, decl *ast.FuncDecl) (*ir.Func, error) {
	start := p.Fset.Position(decl.Body.Pos()).Offset
	end := p.Fset.Position(decl.Body.End()).Offset
	if start < 0 || end > len(source) || start >= end {
		return nil, fmt.Errorf("invalid body span")
	}
	digest := sha256.Sum256(source[start:end])
	id := goid.Func(p.PkgPath, decl.Name.Name)
	if decl.Recv != nil {
		id = goid.Method(p.PkgPath, receiverBase(decl.Recv), decl.Name.Name)
	} else if goid.IsRepeatable("func", decl.Name.Name) {
		// init and blank functions repeat legally: their identities are
		// position-qualified, exactly like the census and the corpus, so
		// probe and corpus classifications join one-to-one.
		filename := p.Fset.Position(decl.Pos()).Filename
		relative, err := filepath.Rel(sourceDir, filename)
		if err != nil {
			return nil, err
		}
		position := p.Fset.Position(decl.Name.Pos())
		id = goid.Repeatable(p.PkgPath, "func", decl.Name.Name, filepath.ToSlash(relative), position.Line, position.Column)
	}
	return ir.BuildFunc(p, sourceDir, unit, decl, id, hex.EncodeToString(digest[:]))
}

// externalRefOfSite extracts the qualified callee of an external-
// reference site, when the blocker is one.
func externalRefOfSite(site ir.UnsupportedSite) (string, bool) {
	for _, prefix := range []string{"call outside the translated unit (", "method call outside the translated unit ("} {
		if strings.HasPrefix(site.Construct, prefix) && strings.HasSuffix(site.Construct, ")") {
			return strings.TrimSuffix(strings.TrimPrefix(site.Construct, prefix), ")"), true
		}
	}
	return "", false
}

func firstLine(s string) string {
	if index := strings.IndexByte(s, '\n'); index >= 0 {
		return s[:index]
	}
	return s
}
