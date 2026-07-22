package analyze

import (
	"go/ast"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/source"
)

// Analyze is the single parent-directed catalog traversal over the transient
// checker graph. It runs after scope selection and before source finalization,
// and is the sole producer of catalog classification, edges, roles, tokens,
// occurrences, variants, implicit operations, implementation definitions, and
// references. Source never classifies constructs; the traversal queries the one
// checker graph through source's narrow transient capability.
//
// For every ordinary-source file it builds one declaration region (rooted at
// the file, stopping at each body edge with a typed reference) and, for every
// full-semantic unit, one body region (rooted at the unit, stopping at nested
// units). A non-full unit is a definition with no body region: its parent keeps
// the reference; its interior contributes zero occurrences and no reachable
// body node.
func Analyze(universe *source.Universe, depths map[identity.SourceUnitID]source.EvidenceDepth, implicitDepths map[identity.ImplicitUnitID]source.EvidenceDepth) (*WorkspaceInventory, source.RetentionProjection, error) {
	out := &WorkspaceInventory{version: InventoryArtifactVersion}
	var fullUnits []identity.SourceUnitID
	var fullImplicit []identity.ImplicitUnitID
	for _, pkg := range universe.Packages() {
		if pkg.Disposition() != source.DispositionOrdinarySource {
			continue
		}
		// Only application packages (with at least one full-semantic unit) enter
		// the region model; contract-depth provider packages are audited
		// separately and contribute no application occurrences, even when the
		// audit's recursive census gave their files traversable syntax.
		if !packageHasFullUnit(pkg, depths) {
			continue
		}
		view := pkg.CheckerView()
		pkgInv := &PackageInventory{id: pkg.ID()}
		hasRegion := false
		for _, file := range pkg.Files() {
			syntax := file.Syntax()
			if syntax == nil {
				continue // provider-owned / manifest file: no body regions
			}
			hasRegion = true
			boundaries := file.UnitBoundaries()
			fset := file.TraversalFset()

			// Declaration region: file-rooted, stops at every body edge.
			decl := &builder{fset: fset, file: file.ID(), owner: FileDeclarationOwner(file.ID()), info: view, boundaries: boundaries}
			if err := decl.visit(syntax, -1, 0, visitContext{}); err != nil {
				return nil, source.RetentionProjection{}, err
			}
			if err := resolveVariants(decl); err != nil {
				return nil, source.RetentionProjection{}, err
			}
			detectImplicit(decl)
			directives, err := scanDirectives(decl, syntax)
			if err != nil {
				return nil, source.RetentionProjection{}, err
			}
			declRegion, err := newFileInventory(file.Path(), file.ID(), file.EffectiveGoVersion(), decl.occurrences, directives)
			if err != nil {
				return nil, source.RetentionProjection{}, err
			}
			pkgInv.files = append(pkgInv.files, declRegion)
			pkgInv.references = append(pkgInv.references, decl.references...)

			// Body regions: one per full-semantic unit in the file.
			for _, unit := range file.Units() {
				depth, ok := depths[unit.ID()]
				if !ok {
					return nil, source.RetentionProjection{}, newResolutionError(0, file.ID(), Span{}, "unit "+unit.ID().String()+" has no selected depth")
				}
				pkgInv.definitions = append(pkgInv.definitions, ImplementationDefinition{
					unit: SourceUnitRef(unit.ID()), kind: unit.Kind(), depth: depth,
					full: depth == source.DepthFullSemantic,
				})
				if depth != source.DepthFullSemantic {
					continue
				}
				root, ok := file.UnitRootNode(unit.ID())
				if !ok {
					return nil, source.RetentionProjection{}, newResolutionError(0, file.ID(), Span{}, "full unit "+unit.ID().String()+" has no root node")
				}
				region, refs, err := buildBodyRegion(file, view, boundaries, unit, root)
				if err != nil {
					return nil, source.RetentionProjection{}, err
				}
				pkgInv.files = append(pkgInv.files, region)
				pkgInv.references = append(pkgInv.references, refs...)
				fullUnits = append(fullUnits, unit.ID())
			}
		}
		for _, implicit := range pkg.ImplicitUnits() {
			depth, ok := implicitDepths[implicit]
			if !ok {
				return nil, source.RetentionProjection{}, newResolutionError(0, identity.FileID{}, Span{}, "implicit unit "+implicit.String()+" has no selected depth")
			}
			pkgInv.definitions = append(pkgInv.definitions, ImplementationDefinition{
				unit: ImplicitUnitRef(implicit), kind: identity.UnitImplicitExecutable, depth: depth,
				full: depth == source.DepthFullSemantic,
			})
			if depth == source.DepthFullSemantic {
				fullImplicit = append(fullImplicit, implicit)
			}
		}
		if hasRegion || len(pkgInv.definitions) > 0 {
			out.packages = append(out.packages, pkgInv)
		}
	}
	projection, err := source.NewRetentionProjection(fullUnits, fullImplicit)
	if err != nil {
		return nil, source.RetentionProjection{}, err
	}
	return out, projection, nil
}

// packageHasFullUnit reports whether any of the package's units is
// full-semantic under the selection.
func packageHasFullUnit(pkg *source.LoadedPackage, depths map[identity.SourceUnitID]source.EvidenceDepth) bool {
	for _, file := range pkg.Files() {
		for _, unit := range file.Units() {
			if depths[unit.ID()] == source.DepthFullSemantic {
				return true
			}
		}
	}
	return false
}

// buildBodyRegion builds one full-semantic unit's body region rooted at its
// own node, with nested units as reference boundaries and the parent-supplied
// signature for return resolution.
func buildBodyRegion(file *source.LoadedFile, view *source.TypeInfoView, boundaries map[ast.Node]identity.SourceUnitID, unit source.SourceUnit, root ast.Node) (*FileInventory, []ImplementationRef, error) {
	b := &builder{
		fset: file.TraversalFset(), file: file.ID(),
		owner: UnitOwner(SourceUnitRef(unit.ID())), info: view, boundaries: boundaries,
	}
	if err := b.visit(root, -1, 0, visitContext{signature: file.UnitSignature(unit.ID())}); err != nil {
		return nil, nil, err
	}
	if err := resolveVariants(b); err != nil {
		return nil, nil, err
	}
	detectImplicit(b)
	rootSpan := Span{
		Start: Position{Offset: unit.Span().Start.Offset},
		End:   Position{Offset: unit.Span().End.Offset},
	}
	region, err := newUnitInventory(file.Path(), file.ID(), file.EffectiveGoVersion(), unit.ID(), rootSpan, b.occurrences)
	if err != nil {
		return nil, nil, err
	}
	return region, b.references, nil
}
