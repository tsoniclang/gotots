package source

import (
	"go/ast"
	"go/token"
	"go/types"
	"sort"

	"github.com/tsoniclang/gotots/internal/identity"
)

var _ token.Pos // positions resolve through the shared file set

// FileEvidence is the sealed structural evidence state of one finalized file.
// The variant IS the state: there is no severed flag and no nil-plus-flag
// dual representation.
type FileEvidence interface{ fileEvidence() }

// FullSyntax retains the complete checked file tree: every unit in the file
// is full-semantic.
type FullSyntax struct{ Syntax *ast.File }

func (FullSyntax) fileEvidence() {}

// ContractOnly retains no syntax: every unit in the file is
// declaration-contract, external-boundary, or intrinsic; unit boundaries and
// the shared type graph carry the evidence.
type ContractOnly struct{}

func (ContractOnly) fileEvidence() {}

// MixedUnits retains exactly the full-semantic units' declaration subtrees
// (go/ast has no parent pointers, so a retained declaration cannot reach the
// enclosing file or sibling declarations).
type MixedUnits struct{ Retained []RetainedUnit }

func (MixedUnits) fileEvidence() {}

// RetainedUnit is one full-semantic unit's retained declaration subtree rooted
// at the unit's own exact node (the FuncDecl of a func/bodyless unit, the
// FuncLit of a literal unit, the ValueSpec of an initializer). Boundaries are
// the unit's DIRECT nested units' root nodes: they are structural holes in
// this region — excluded from both the retained type-information membership and
// the inventory walk, so a nested unit's body is unreachable through its
// parent and every implementation body is an independently traversable region.
// For cgo originals the subtree comes from the checked view; CheckedSpan then
// carries the declaration's span in checked-view coordinates (the origin
// mapping's evidence), and occurrence positions resolve through the shared
// file set whose //line handling maps display positions back to the original.
type RetainedUnit struct {
	Unit            identity.SourceUnitID
	Decl            ast.Node
	Boundaries      []ast.Node
	FromCheckedView bool
	CheckedSpan     Span
}

// Finalize consumes the transient universe plus the scope phase's immutable
// per-unit evidence-depth selection (explicit and implicit) and produces the
// finalized Workspace: structural per-file evidence, per-unit filtered type
// information, and depth-aware validated admission. The selection must be
// total over the census — every censused unit selected exactly once, no
// extras.
func Finalize(u *Universe, depths map[identity.SourceUnitID]EvidenceDepth, implicitDepths map[identity.ImplicitUnitID]EvidenceDepth) (*Workspace, error) {
	failScope := func(reason string) (*Workspace, error) {
		return nil, &LoadError{Dir: u.request.Dir, Reason: "scope selection invalid: " + reason}
	}
	remaining := make(map[identity.SourceUnitID]bool, len(depths))
	for id, depth := range depths {
		if !depth.Valid() {
			return failScope(id.String() + " has invalid depth")
		}
		remaining[id] = true
	}
	remainingImplicit := make(map[identity.ImplicitUnitID]bool, len(implicitDepths))
	for id, depth := range implicitDepths {
		if !depth.Valid() {
			return failScope(id.String() + " has invalid depth")
		}
		remainingImplicit[id] = true
	}
	ws := &Workspace{fset: u.fset, toolchain: u.toolchain}
	for _, loaded := range u.packages {
		record, err := finalizePackage(u, loaded, depths, remaining)
		if err != nil {
			return nil, err
		}
		for _, implicit := range loaded.implicitUnits {
			depth, selected := implicitDepths[implicit]
			if !selected {
				return failScope("selection missing implicit unit " + implicit.String())
			}
			delete(remainingImplicit, implicit)
			record.implicitUnits = append(record.implicitUnits, ImplicitUnit{id: implicit, depth: depth})
		}
		if err := ws.admit(record); err != nil {
			return nil, err
		}
	}
	for id := range remaining {
		return failScope("selection names unknown unit " + id.String())
	}
	for id := range remainingImplicit {
		return failScope("selection names unknown implicit unit " + id.String())
	}
	sort.Slice(ws.packages, func(i, j int) bool { return ws.packages[i].id.String() < ws.packages[j].id.String() })
	sort.Slice(ws.roots, func(i, j int) bool { return ws.roots[i].id.String() < ws.roots[j].id.String() })
	return ws, nil
}

// finalizePackage builds one finalized package record with structural
// evidence per depth.
func finalizePackage(u *Universe, loaded *LoadedPackage, depths map[identity.SourceUnitID]EvidenceDepth, remaining map[identity.SourceUnitID]bool) (*Package, error) {
	out := &Package{
		id: loaded.id, provenance: loaded.provenance, acquisition: loaded.acquisition,
		disposition: loaded.disposition, moduleGoVersion: loaded.moduleGoVersion,
		requestedRoot: loaded.requestedRoot,
		imports:       append([]string(nil), loaded.imports...),
		otherFiles:    append([]string(nil), loaded.otherFiles...),
		embedFiles:    append([]string(nil), loaded.embedFiles...),
		embedPatterns: append([]string(nil), loaded.embedPatterns...),
		types:         loaded.types,
		mappings:      append([]CheckedUnitMapping(nil), loaded.mappings...),
		synthetics:    append([]SyntheticUnit(nil), loaded.synthetics...),
	}
	allFull, anyFull := true, false
	for _, loadedFile := range loaded.files {
		fset := loadedFile.fset
		if fset == nil {
			// Cgo originals resolve retained checked-view positions through
			// the shared file set (//line evidence maps display back here).
			fset = u.fset
		}
		file := &File{
			path: loadedFile.path, id: loadedFile.id, fset: fset,
			effectiveVersion: loadedFile.effectiveVersion,
			overlaid:         loadedFile.overlaid, cgoOriginal: loadedFile.cgoOriginal,
			byteDigest: loadedFile.byteDigest,
		}
		fileAllFull, fileAnyFull := true, false
		for _, unit := range loadedFile.units {
			depth, selected := depths[unit.id]
			if !selected {
				return nil, &LoadError{Dir: u.request.Dir, Reason: "scope selection missing unit " + unit.id.String()}
			}
			delete(remaining, unit.id)
			unit.depth = depth
			file.units = append(file.units, unit)
			if depth == DepthFullSemantic {
				fileAnyFull, anyFull = true, true
			} else {
				fileAllFull, allFull = false, false
			}
		}
		if len(loadedFile.units) == 0 {
			// Declaration-only files (types, constants) carry no executable
			// units; their evidence class follows the package's other files
			// and the shared type graph. They are full-syntax retained only
			// when the file itself is checked and the package retains
			// full-semantic evidence.
			fileAllFull = loadedFile.syntax != nil
		}
		switch {
		case loadedFile.syntax != nil && fileAllFull:
			file.evidence = FullSyntax{Syntax: loadedFile.syntax}
			if len(loadedFile.units) == 0 && loaded.disposition == DispositionOrdinarySource {
				// keep declaration-only files with their package's retention
				// class: contract packages retain no syntax
				if !packageRetainsFull(loaded, depths) {
					file.evidence = ContractOnly{}
				}
			}
		case fileAnyFull:
			retained, err := retainUnits(u, loaded, loadedFile, depths)
			if err != nil {
				return nil, err
			}
			file.evidence = MixedUnits{Retained: retained}
		default:
			file.evidence = ContractOnly{}
		}
		out.files = append(out.files, file)
	}
	sort.Slice(out.files, func(i, j int) bool { return out.files[i].id.Rel() < out.files[j].id.Rel() })
	// Type information retention follows the depth partition structurally:
	// uniform-full packages keep the checker's Info (it indexes only
	// full-semantic bodies); mixed packages keep a filtered per-unit view;
	// packages with no full-semantic unit keep none. The shared
	// types.Package declaration graph is always retained.
	switch {
	case anyFull && allFull && loaded.typesInfo != nil:
		out.typesInfo = loaded.typesInfo
	case anyFull && loaded.typesInfo != nil:
		out.typesInfo = filterInfoByMembership(loaded.typesInfo, retainedNodes(out.files))
	}
	return out, nil
}

// retainedNodes collects the exact AST node set of every retained
// full-semantic subtree in the package: complete trees of uniform-full files
// plus the retained unit subtrees of mixed files, each excluding its nested
// units' subtrees at their boundaries. Nothing outside this set — no non-full
// body, and no full nested unit's interior (which is a member under its own
// region instead) — can key a retained Info entry.
func retainedNodes(files []*File) map[ast.Node]bool {
	members := map[ast.Node]bool{}
	for _, file := range files {
		switch evidence := file.evidence.(type) {
		case FullSyntax:
			collectExcluding(evidence.Syntax, nil, members)
		case MixedUnits:
			for _, retained := range evidence.Retained {
				collectExcluding(retained.Decl, boundarySet(retained.Boundaries), members)
			}
		}
	}
	return members
}

// boundarySet indexes a retained unit's boundary nodes for membership tests.
func boundarySet(boundaries []ast.Node) map[ast.Node]bool {
	if len(boundaries) == 0 {
		return nil
	}
	set := make(map[ast.Node]bool, len(boundaries))
	for _, n := range boundaries {
		set[n] = true
	}
	return set
}

// collectExcluding adds every node of root's subtree to members, halting at
// (and excluding) each boundary node so a nested unit's interior never enters
// its parent's membership.
func collectExcluding(root ast.Node, boundaries map[ast.Node]bool, members map[ast.Node]bool) {
	ast.Inspect(root, func(n ast.Node) bool {
		if n == nil {
			return false
		}
		if boundaries[n] {
			return false
		}
		members[n] = true
		return true
	})
}

// packageRetainsFull reports whether any unit of the package is full-semantic.
func packageRetainsFull(loaded *LoadedPackage, depths map[identity.SourceUnitID]EvidenceDepth) bool {
	for _, file := range loaded.files {
		for _, unit := range file.units {
			if depths[unit.id] == DepthFullSemantic {
				return true
			}
		}
	}
	return false
}

// retainUnits collects the full-semantic units' retained regions of one mixed
// file: each rooted at the unit's exact node with its direct nested units as
// structural boundaries. For cgo originals the retained subtree comes from the
// checked view through the origin mapping (nested-unit exactness there rides on
// the origin graph and carries no boundaries yet).
func retainUnits(u *Universe, loaded *LoadedPackage, file *LoadedFile, depths map[identity.SourceUnitID]EvidenceDepth) ([]RetainedUnit, error) {
	var nodeOf map[identity.SourceUnitID]ast.Node
	var boundaryOf map[identity.SourceUnitID][]ast.Node
	if file.syntax != nil {
		var err error
		nodeOf, boundaryOf, err = unitTopology(u, file)
		if err != nil {
			return nil, err
		}
	}
	var out []RetainedUnit
	for _, unit := range file.units {
		if depths[unit.id] != DepthFullSemantic {
			continue
		}
		retained := RetainedUnit{Unit: unit.id}
		if file.syntax != nil {
			retained.Decl = nodeOf[unit.id]
			retained.Boundaries = boundaryOf[unit.id]
		} else if counterpart, ok := loaded.checkedNodes[unit.id]; ok {
			retained.Decl = counterpart.node
			retained.FromCheckedView = true
			retained.CheckedSpan = counterpart.span
		}
		if retained.Decl == nil {
			return nil, &LoadError{Dir: u.request.Dir, Reason: "no retainable declaration for full-semantic unit " + unit.id.String()}
		}
		out = append(out, retained)
	}
	return out, nil
}

// unitTopology maps every censused unit of one file to its exact root node and
// its direct nested units' root nodes (its structural boundaries). Nested
// units are function literals; a unit's direct children are the units it
// contains with no intervening unit. Nesting is decided by physical span
// containment — never by walking order.
func unitTopology(u *Universe, file *LoadedFile) (map[identity.SourceUnitID]ast.Node, map[identity.SourceUnitID][]ast.Node, error) {
	type spanKey struct{ start, end int }
	funcBody := map[spanKey]ast.Node{} // body span -> *ast.FuncDecl
	bodyless := map[spanKey]ast.Node{} // decl span -> *ast.FuncDecl
	litBody := map[spanKey]ast.Node{}  // funclit span -> *ast.FuncLit
	valueSpec := map[spanKey]ast.Node{}
	keyOf := func(n ast.Node) spanKey {
		return spanKey{file.fset.PositionFor(n.Pos(), false).Offset, file.fset.PositionFor(n.End(), false).Offset}
	}
	ast.Inspect(file.syntax, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.FuncDecl:
			if n.Body != nil {
				funcBody[keyOf(n.Body)] = n
			} else {
				bodyless[keyOf(n)] = n
			}
		case *ast.FuncLit:
			litBody[keyOf(n)] = n
		case *ast.ValueSpec:
			valueSpec[keyOf(n)] = n
		}
		return true
	})
	nodeOf := make(map[identity.SourceUnitID]ast.Node, len(file.units))
	for _, unit := range file.units {
		k := spanKey{unit.span.Start.Offset, unit.span.End.Offset}
		var node ast.Node
		switch unit.id.Kind() {
		case identity.UnitFuncBody:
			node = funcBody[k]
		case identity.UnitBodylessDecl:
			node = bodyless[k]
		case identity.UnitFuncLitBody:
			node = litBody[k]
		case identity.UnitVarInitializer:
			node = valueSpec[k]
		}
		if node == nil {
			return nil, nil, &LoadError{Dir: u.request.Dir, Reason: "no exact syntax node for unit " + unit.id.String()}
		}
		nodeOf[unit.id] = node
	}
	contains := func(a, b SourceUnit) bool {
		if a.span.Start.Offset == b.span.Start.Offset && a.span.End.Offset == b.span.End.Offset {
			return false
		}
		return a.span.Start.Offset <= b.span.Start.Offset && b.span.End.Offset <= a.span.End.Offset
	}
	boundaryOf := map[identity.SourceUnitID][]ast.Node{}
	for _, parent := range file.units {
		for _, child := range file.units {
			if !contains(parent, child) {
				continue
			}
			direct := true
			for _, mid := range file.units {
				if mid.id == parent.id || mid.id == child.id {
					continue
				}
				if contains(parent, mid) && contains(mid, child) {
					direct = false
					break
				}
			}
			if direct {
				boundaryOf[parent.id] = append(boundaryOf[parent.id], nodeOf[child.id])
			}
		}
	}
	return nodeOf, boundaryOf, nil
}

// filterInfoByMembership copies exactly the type-information entries whose
// keys are members of retained full-semantic subtrees — exact AST identity,
// never byte offsets that different files could share. No retained Info key
// can reference a non-full body.
func filterInfoByMembership(info *types.Info, members map[ast.Node]bool) *types.Info {
	out := &types.Info{
		Types: map[ast.Expr]types.TypeAndValue{}, Defs: map[*ast.Ident]types.Object{},
		Uses: map[*ast.Ident]types.Object{}, Selections: map[*ast.SelectorExpr]*types.Selection{},
		Implicits: map[ast.Node]types.Object{}, Instances: map[*ast.Ident]types.Instance{},
		FileVersions: map[*ast.File]string{},
	}
	for key, value := range info.Types {
		if members[key] {
			out.Types[key] = value
		}
	}
	for key, value := range info.Defs {
		if members[key] {
			out.Defs[key] = value
		}
	}
	for key, value := range info.Uses {
		if members[key] {
			out.Uses[key] = value
		}
	}
	for key, value := range info.Selections {
		if members[key] {
			out.Selections[key] = value
		}
	}
	for key, value := range info.Implicits {
		if members[key] {
			out.Implicits[key] = value
		}
	}
	for key, value := range info.Instances {
		if members[key] {
			out.Instances[key] = value
		}
	}
	return out
}
