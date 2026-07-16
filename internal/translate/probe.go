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
					for _, site := range function.Sites {
						result.BlockerHistogram[site.Class]++
						result.ConstructHistogram[site.Construct]++
						packageSites = append(packageSites,
							fmt.Sprintf("%s: %s (%s:%d)", function.ID, site.Construct, site.Span.File, site.Span.Line))
						if ref, isExternal := externalRefOfSite(site); isExternal {
							result.ExternalRefs[ref]++
						}
					}
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
	return result, nil
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
