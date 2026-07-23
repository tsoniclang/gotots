package analyze

import (
	"go/ast"
	"go/token"

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
		if !packageHasFullUnit(pkg, depths, implicitDepths) {
			continue
		}
		view := pkg.CheckerView()
		pkgInv := &PackageInventory{id: pkg.ID()}
		hasRegion := false
		cgoBoundaries := pkg.CgoCounterparts()
		for _, file := range pkg.Files() {
			// Definitions for every unit of every file (full and non-full).
			for _, unit := range file.Units() {
				depth, ok := depths[unit.ID()]
				if !ok {
					return nil, source.RetentionProjection{}, newResolutionError(0, file.ID(), Span{}, "unit "+unit.ID().String()+" has no selected depth")
				}
				contract, err := ContractForKind(unit.Kind())
				if err != nil {
					return nil, source.RetentionProjection{}, newResolutionError(0, file.ID(), Span{}, err.Error())
				}
				pkgInv.definitions = append(pkgInv.definitions, ImplementationDefinition{
					unit: SourceUnitRef(unit.ID()), kind: unit.Kind(), contract: contract, depth: depth,
					full: depth == source.DepthFullSemantic,
				})
			}

			syntax := file.Syntax()
			switch {
			case syntax != nil:
				// Ordinary file: one declaration region plus one body region per
				// full unit, all in the file's own checked syntax.
				hasRegion = true
				boundaries := file.UnitBoundaries()
				fset := file.TraversalFset()
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
				for _, unit := range file.Units() {
					if depths[unit.ID()] != source.DepthFullSemantic {
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

			case file.CgoOriginal():
				// Cgo original: full units are inventoried through their exact
				// checked-view counterparts (which carry type evidence), in the
				// shared checked file set, rooted at the counterpart node.
				for _, unit := range file.Units() {
					if depths[unit.ID()] != source.DepthFullSemantic {
						continue
					}
					node, span, ok := pkg.CgoCounterpartNode(unit.ID())
					if !ok {
						return nil, source.RetentionProjection{}, newResolutionError(0, file.ID(), Span{}, "full cgo unit "+unit.ID().String()+" has no checked counterpart")
					}
					hasRegion = true
					region, refs, err := buildCgoBodyRegion(file, view, universe.Fset(), cgoBoundaries, unit, node, span)
					if err != nil {
						return nil, source.RetentionProjection{}, err
					}
					pkgInv.files = append(pkgInv.files, region)
					pkgInv.references = append(pkgInv.references, refs...)
					fullUnits = append(fullUnits, unit.ID())
				}
			}
		}
		for _, implicit := range pkg.ImplicitUnits() {
			depth, ok := implicitDepths[implicit]
			if !ok {
				return nil, source.RetentionProjection{}, newResolutionError(0, identity.FileID{}, Span{}, "implicit unit "+implicit.String()+" has no selected depth")
			}
			pkgInv.definitions = append(pkgInv.definitions, ImplementationDefinition{
				unit: ImplicitUnitRef(implicit), kind: identity.UnitImplicitExecutable,
				contract: ContractCatalogOwner, depth: depth,
				full: depth == source.DepthFullSemantic,
			})
			// Every implicit definition has its owning package-initialization
			// reference, so definitions = source units + implicit units all join
			// references (conservation totality).
			pkgInv.references = append(pkgInv.references, NewImplicitReference(pkg.ID(), implicit))
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

// FileGraph is one file's provider definition/reference graph: the topology
// authority the audit artifact carries so ordinary compilation exact-joins it
// without rescanning provider interiors. It is depth-independent structure
// (the extraction treats every unit as having a body region to recover its
// nested references); the flat unit list is a derived projection of Definitions.
type FileGraph struct {
	Definitions []ImplementationDefinition
	References  []ImplementationRef
}

// ExtractProviderGraph derives the definition/reference graph of every
// ordinary-source file with traversal syntax, treating every unit as having a
// body region so nested references are recovered. It retains no occurrences —
// only the topology — so it stays bounded per file. This is the audit
// producer's provider-graph extraction; graph ownership is here, not in source.
func ExtractProviderGraph(universe *source.Universe) (map[string]FileGraph, error) {
	out := map[string]FileGraph{}
	for _, pkg := range universe.Packages() {
		if pkg.Disposition() != source.DispositionOrdinarySource {
			continue
		}
		view := pkg.CheckerView()
		for _, file := range pkg.Files() {
			syntax := file.Syntax()
			if syntax == nil {
				continue
			}
			boundaries := file.UnitBoundaries()
			g := FileGraph{}
			decl := &builder{fset: file.TraversalFset(), file: file.ID(), owner: FileDeclarationOwner(file.ID()), info: view, boundaries: boundaries}
			if err := decl.visit(syntax, -1, 0, visitContext{}); err != nil {
				return nil, err
			}
			g.References = append(g.References, decl.references...)
			for _, unit := range file.Units() {
				contract, err := ContractForKind(unit.Kind())
				if err != nil {
					return nil, err
				}
				g.Definitions = append(g.Definitions, ImplementationDefinition{
					unit: SourceUnitRef(unit.ID()), kind: unit.Kind(), contract: contract,
					depth: source.DepthDeclarationContract, full: false,
				})
				root, ok := file.UnitRootNode(unit.ID())
				if !ok {
					continue // manifest-supplied interior with no local root node
				}
				b := &builder{fset: file.TraversalFset(), file: file.ID(), owner: UnitOwner(SourceUnitRef(unit.ID())), info: view, boundaries: boundaries}
				if err := b.visit(root, -1, 0, visitContext{signature: file.UnitSignature(unit.ID())}); err != nil {
					return nil, err
				}
				g.References = append(g.References, b.references...)
			}
			out[file.ID().String()] = g
		}
	}
	return out, nil
}

// packageHasFullUnit reports whether any of the package's units — source or
// implicit — is full-semantic under the selection.
func packageHasFullUnit(pkg *source.LoadedPackage, depths map[identity.SourceUnitID]source.EvidenceDepth, implicitDepths map[identity.ImplicitUnitID]source.EvidenceDepth) bool {
	for _, file := range pkg.Files() {
		for _, unit := range file.Units() {
			if depths[unit.ID()] == source.DepthFullSemantic {
				return true
			}
		}
	}
	for _, implicit := range pkg.ImplicitUnits() {
		if implicitDepths[implicit] == source.DepthFullSemantic {
			return true
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

// buildCgoBodyRegion builds one full-semantic cgo unit's body region from its
// checked-view counterpart, walked in the shared checked file set. Occurrence
// spans are checked-view coordinates (the counterpart carries the type
// evidence); the region roots at the original unit identity through the origin
// mapping's checked span.
func buildCgoBodyRegion(file *source.LoadedFile, view *source.TypeInfoView, checkedFset *token.FileSet, boundaries map[ast.Node]identity.SourceUnitID, unit source.SourceUnit, node ast.Node, checkedSpan source.Span) (*FileInventory, []ImplementationRef, error) {
	b := &builder{
		fset: checkedFset, file: file.ID(),
		owner: UnitOwner(SourceUnitRef(unit.ID())), info: view, boundaries: boundaries,
	}
	if err := b.visit(node, -1, 0, visitContext{}); err != nil {
		return nil, nil, err
	}
	if err := resolveVariants(b); err != nil {
		return nil, nil, err
	}
	detectImplicit(b)
	rootSpan := Span{
		Start: Position{Offset: checkedSpan.Start.Offset},
		End:   Position{Offset: checkedSpan.End.Offset},
	}
	region, err := newUnitInventory(file.Path(), file.ID(), file.EffectiveGoVersion(), unit.ID(), rootSpan, b.occurrences)
	if err != nil {
		return nil, nil, err
	}
	return region, b.references, nil
}
