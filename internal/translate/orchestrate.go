// Whole-unit translation orchestration: the prepasses, the
// materialization and publication fixed points, emission, the ABI
// modules, and the single final evidence pass.
// Package translate orchestrates the translation proof chain for one
// unit of packages: census-aligned declaration identity -> typed body IR
// -> conservative representation plan -> lowering -> deterministic
// TypeScript with provenance -> per-declaration proof records. Named
// types, calls, and pointer-receiver methods resolve across the unit
// through generated module imports.
//
// Every construct outside the reviewed subset fails closed with a stable
// GOTOTS_UNSUPPORTED_* diagnostic; nothing is approximated.
package translate

import (
	"fmt"
	"go/ast"
	"path"
	"sort"

	"golang.org/x/tools/go/packages"

	"github.com/tsoniclang/gotots/internal/abi"
	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/facts"
	"github.com/tsoniclang/gotots/internal/implid"
	"github.com/tsoniclang/gotots/internal/ir"
	"github.com/tsoniclang/gotots/internal/plan"
	"github.com/tsoniclang/gotots/internal/tsident"
	"strings"
)

// Packages translates a set of loaded production packages as one unit:
// named types, direct calls, and pointer-receiver method calls resolve
// across every package in the set, and each package emits one module at
// core/<pkgPath>/package.ts importing its co-generated dependencies.
func Packages(pkgs []*packages.Package, sourceDir string, options Options) (*Generated, error) {
	sorted := append([]*packages.Package{}, pkgs...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].PkgPath < sorted[j].PkgPath })
	seen := map[string]bool{}
	paths := make([]string, 0, len(sorted))
	for _, p := range sorted {
		if seen[p.PkgPath] {
			return nil, fmt.Errorf("package %s appears twice in the translation unit", p.PkgPath)
		}
		seen[p.PkgPath] = true
		paths = append(paths, p.PkgPath)
	}
	unit := ir.NewScope(paths...)
	if err := collectGenericInstances(unit, sorted); err != nil {
		return nil, err
	}

	out := &Generated{
		Files:           map[string]string{},
		Ownership:       map[string]string{},
		Withheld:        map[string]string{},
		NotMaterialized: map[string]string{},
		IfaceArtifacts:  emit.NewIfaceArtifacts(),
	}
	var emitters []func() error
	for _, p := range sorted {
		if err := translatePackage(out, p, sourceDir, unit, options, &emitters); err != nil {
			return nil, err
		}
	}
	// The pipeline boundary: translation is complete, so the whole-
	// program facts seal and the plans freeze BEFORE any emitter runs.
	factStore := facts.New()
	for _, fact := range out.NilabilityFacts {
		f := facts.ReceiverNilability{EquivalentAtEntry: fact.EquivalentAtEntry, ToleratesNil: fact.ToleratesNil}
		if fact.GenericReceiver || fact.CarrierReceiver {
			// Generic-container methods ride the generic ownership wave
			// (step 19) and carrier-receiver methods the carrier wave:
			// withhold the entry-equivalence proof so the planner keeps
			// them on the exception lowering.
			f = facts.ReceiverNilability{ToleratesNil: fact.ToleratesNil}
		}
		factStore.PutReceiverNilability(fact.ID, f)
	}
	factStore.Seal()
	planBuilder := plan.NewImplBuilder()
	methods := make([]implid.ID, 0, len(out.NilabilityFacts))
	delegateEligible := make(map[string]bool, len(out.NilabilityFacts))
	for _, fact := range out.NilabilityFacts {
		methods = append(methods, implid.MustNew(fact.ID, "default"))
		delegateEligible[fact.ID] = !fact.GenericReceiver && !fact.CarrierReceiver
	}
	if err := plan.BuildMethodPlans(planBuilder, factStore, methods, delegateEligible); err != nil {
		return nil, err
	}
	out.MethodPlans = planBuilder.Build()
	// The plan owns each method's emitted symbol: proofs of
	// ordinary-planned methods record the class-member spelling instead
	// of the legacy free-function symbol.
	planKeyByCensus := make(map[string]string, len(out.NilabilityFacts))
	for _, fact := range out.NilabilityFacts {
		planKeyByCensus[fact.CensusID] = fact.ID
	}
	for i := range out.Proofs {
		planKey, isMethod := planKeyByCensus[out.Proofs[i].ID]
		if !isMethod {
			continue
		}
		id := planKey
		p, ok := out.MethodPlans.Get(implid.MustNew(id, "default"))
		if !ok || p.MethodEmission != plan.MethodOrdinaryNilChecked {
			continue
		}
		symbol := id
		if last := strings.LastIndex(symbol, "::"); last >= 0 {
			symbol = symbol[last+2:]
		}
		// Class-qualified member evidence: Type.member, each segment in
		// its emitted spelling.
		if dot := strings.LastIndex(symbol, "."); dot >= 0 {
			symbol = tsident.EscapeDeclared(symbol[:dot]) + "." + tsident.EscapeDeclared(symbol[dot+1:])
		} else {
			symbol = tsident.EscapeDeclared(symbol)
		}
		out.Proofs[i].GeneratedSymbol = symbol
	}

	// Materialization cascade FIRST, over source imports (declaration
	// blockers are known before emission): a package importing a
	// non-materializable one cannot materialize either. The fixed point
	// determines exactly which packages emit, so the emitters run once.
	for growNotMaterializedByImports(out, sorted) {
	}
	// Materialize every eligible package (typed throwing placeholders for its
	// unimplemented bodies). Emission is order-independent — every referenced
	// class exists because materialization is closed under imports.
	for _, emitPackage := range emitters {
		if err := emitPackage(); err != nil {
			return nil, err
		}
	}
	if err := emitExternalStubs(out, unit, sorted[0], sourceDir, options); err != nil {
		return nil, err
	}
	// The canonical interface-artifacts module (ADR-0008): every union
	// registered by a consumer is defined here exactly once. Built after
	// all consumers so the registry is complete.
	if err := emitInterfacesModule(out, unit, sorted[0], sourceDir, options); err != nil {
		return nil, err
	}
	// Publication withholding is a SEPARATE fixed point over the real emitted
	// import edges: a package with any unimplemented unit — or one importing
	// a publication-withheld package — does not enter the runnable product.
	// Its materialized file is RETAINED for analysis (never deleted).
	for growWithheldByImports(out, sorted) {
	}
	// Every PUBLISHED (retained) module must import only published modules: a
	// runnable product cannot depend on a withheld package. Materialized-but-
	// withheld modules are excluded from this closure (they are analysis
	// artifacts, not product).
	if err := assertRetainedImportsResolve(out); err != nil {
		return nil, err
	}

	abiFiles := abi.Files()
	abiNames := make([]string, 0, len(abiFiles))
	for name := range abiFiles {
		abiNames = append(abiNames, name)
	}
	sort.Strings(abiNames)
	for _, name := range abiNames {
		abiPath := path.Join(abiDir, name)
		content, err := emit.FileWithProvenance(emit.Provenance{
			SchemaVersion:  1,
			SourceRevision: options.SourceRevision,
			ProfileHash:    options.ProfileHash,
			AbiVersion:     abi.Version,
			Path:           abiPath,
		}, abiFiles[name])
		if err != nil {
			return nil, err
		}
		out.Files[abiPath] = content
		out.Ownership[abiPath] = "generated-language-abi"
	}

	// Verify every union-$key unreachability claim against the now
	// corpus-complete value-box log: a member claimed unreachable that IS
	// value-boxed somewhere would make the invariant stop reachable — an
	// emitter defect, never a shippable state.
	for memberK := range out.KeyUnreachableClaims {
		if unit.ValueBoxedNamed(memberK) {
			out.EmitterDefects = append(out.EmitterDefects, EmitterDefect{
				Package: "corpus", ID: memberK,
				Err: "union $key claims value-box unreachability for " + memberK + " but its value form IS boxed (claim violated)",
			})
		}
	}
	sort.Slice(out.Proofs, func(i, j int) bool { return out.Proofs[i].ID < out.Proofs[j].ID })
	sort.Slice(out.Support, func(i, j int) bool { return out.Support[i].ID < out.Support[j].ID })
	// The single final evidence pass: every file and proof now exists
	// (core, ABI, external stubs), so stages finalize exactly once and
	// invalid evidence is a build failure, not a normalization.
	if err := finalizeEvidenceStages(out); err != nil {
		return nil, err
	}
	return out, nil
}

// fileSource pairs one production file's AST with its exact source.
type fileSource struct {
	file     *ast.File
	relative string
	source   []byte
}
