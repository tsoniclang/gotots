package source

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/identity"
)

// resolveCgo derives one package's cgo evidence from authoritative checked and
// type information: the exact origin graph joining each checked-view element to
// its origin unit, the typed synthetic units, and per-unit C-dependence.
//
// C-dependence is NOT a spelling or a hand-maintained scope model. A unit is
// C-dependent iff its checked counterpart uses a package-synthetic (cgo-
// generated) object — a fact the toolchain's own type checker resolves, so a
// local declaration named C that resolves to an ordinary object never
// classifies, and every legal Go scope form falls out of resolved identity.
//
// The producer join (forward: origin unit -> counterpart) and the independent
// verifier join (reverse: counterpart -> origin unit, with an independently
// recomputed synthetic-object set) are exact-joined; any divergence fails
// closed.
func resolveCgo(u *Universe, pkg *LoadedPackage) error {
	if len(pkg.checkedDecls) == 0 {
		return nil // not a cgo package: no checked view, no C-dependence
	}
	originals := map[string]*LoadedFile{}
	for _, file := range pkg.files {
		originals[file.path] = file
	}
	producer, err := deriveOriginGraph(u, pkg, originals)
	if err != nil {
		return err
	}
	if err := verifyOriginGraph(u, pkg, originals, producer); err != nil {
		return err
	}
	pkg.checkedNodes = producer.counterparts
	pkg.synthetics = producer.synthetics
	for _, mapping := range producer.mappings {
		pkg.mappings = append(pkg.mappings, mapping)
	}
	return applyCDependence(u, pkg, producer)
}

// originGraph is one derivation of a cgo package's checked evidence.
type originGraph struct {
	counterparts map[identity.SourceUnitID]checkedCounterpart
	mappings     []CheckedUnitMapping
	synthetics   []SyntheticUnit
	syntheticObj map[types.Object]bool // cgo-generated objects (defs of synthetics)
}

// unitBearing enumerates the unit-bearing elements of one top-level
// declaration: (node whose position joins to an origin unit, the kind, and the
// node whose span is retained). For a func body the join node is the body and
// the retained node is the whole decl; for a literal both are the literal.
type unitBearing struct {
	joinNode ast.Node
	spanNode ast.Node
	kind     identity.UnitKind
}

// declElements returns the unit-bearing elements of one declaration, in source
// order (top-level unit first, then nested literals).
func declElements(decl ast.Node) []unitBearing {
	var out []unitBearing
	switch node := decl.(type) {
	case *ast.FuncDecl:
		if node.Body != nil {
			out = append(out, unitBearing{joinNode: node.Body, spanNode: node, kind: identity.UnitFuncBody})
			out = append(out, litElements(node.Body)...)
		} else {
			out = append(out, unitBearing{joinNode: node, spanNode: node, kind: identity.UnitBodylessDecl})
		}
	case *ast.GenDecl:
		if node.Tok != token.VAR {
			return nil
		}
		for _, spec := range node.Specs {
			value, ok := spec.(*ast.ValueSpec)
			if !ok || len(value.Values) == 0 {
				continue
			}
			out = append(out, unitBearing{joinNode: value, spanNode: value, kind: identity.UnitVarInitializer})
			out = append(out, litElements(value)...)
		}
	}
	return out
}

// litElements enumerates every function literal under root.
func litElements(root ast.Node) []unitBearing {
	var out []unitBearing
	ast.Inspect(root, func(n ast.Node) bool {
		if lit, ok := n.(*ast.FuncLit); ok {
			out = append(out, unitBearing{joinNode: lit, spanNode: lit, kind: identity.UnitFuncLitBody})
		}
		return true
	})
	return out
}

// deriveOriginGraph is the producer join (forward: each checked declaration's
// unit-bearing elements to their origin units, by exact //line-adjusted
// position with kind agreement and uniqueness). Declarations whose adjusted
// file is not a package original are typed synthetic units.
func deriveOriginGraph(u *Universe, pkg *LoadedPackage, originals map[string]*LoadedFile) (*originGraph, error) {
	graph := &originGraph{
		counterparts: map[identity.SourceUnitID]checkedCounterpart{},
		syntheticObj: map[types.Object]bool{},
	}
	syntheticSeen := map[string]bool{}
	for i := range pkg.checkedDecls {
		decl := &pkg.checkedDecls[i]
		display := u.fset.Position(decl.node.Pos())
		if _, isOriginal := originals[display.Filename]; !isOriginal {
			if err := recordSynthetic(u, pkg, decl.node, graph, syntheticSeen); err != nil {
				return nil, err
			}
			continue
		}
		for _, element := range declElements(decl.node) {
			origin, err := joinOrigin(u, originals, element)
			if err != nil {
				return nil, err
			}
			if _, dup := graph.counterparts[origin.id]; dup {
				return nil, &LoadError{Dir: u.request.Dir, Reason: "duplicate origin mapping for unit " + origin.id.String()}
			}
			span := checkedSpanOf(u, element.spanNode)
			graph.counterparts[origin.id] = checkedCounterpart{node: element.spanNode, span: span}
			graph.mappings = append(graph.mappings, CheckedUnitMapping{source: origin.id, checked: span})
		}
	}
	if err := requireTotalCoverage(u, pkg, graph.counterparts); err != nil {
		return nil, err
	}
	return graph, nil
}

// joinOrigin resolves one checked element to its unique origin unit: same
// kind, same //line-adjusted start line, column-disambiguated among same-line
// candidates, exactly one winner — never first-match.
func joinOrigin(u *Universe, originals map[string]*LoadedFile, element unitBearing) (*SourceUnit, error) {
	display := u.fset.Position(element.joinNode.Pos())
	file, isOriginal := originals[display.Filename]
	if !isOriginal {
		return nil, &LoadError{Dir: u.request.Dir, Reason: "checked element adjusts to " + display.Filename +
			", which is not a package original"}
	}
	var candidates []*SourceUnit
	for i := range file.units {
		unit := &file.units[i]
		if unit.id.Kind() == element.kind && unit.span.Start.Line == display.Line {
			candidates = append(candidates, unit)
		}
	}
	if len(candidates) > 1 && display.Column > 0 {
		var exact []*SourceUnit
		for _, unit := range candidates {
			if unit.span.Start.Column == display.Column {
				exact = append(exact, unit)
			}
		}
		candidates = exact
	}
	if len(candidates) != 1 {
		return nil, &LoadError{Dir: u.request.Dir, Reason: originAmbiguity(element.kind, display, len(candidates))}
	}
	return candidates[0], nil
}

func originAmbiguity(kind identity.UnitKind, pos token.Position, n int) string {
	return "origin join for a " + kind.String() + " element at " + pos.String() +
		" resolves " + itoa(n) + " candidate units (exactly one required)"
}

// recordSynthetic records the package-synthetic checked declarations one
// generated declaration introduces (one per declared package-scope name) and
// their defined objects. Each identity is the owning package plus the declared
// scope name plus the role derived from the declaration's own kind — canonical,
// collision-checked, and independent of any temporary or display path. Import
// and unnamed forms declare no synthetic unit.
func recordSynthetic(u *Universe, pkg *LoadedPackage, node ast.Node, graph *originGraph, seen map[string]bool) error {
	for _, decl := range syntheticDecls(node) {
		synthetic, err := NewSyntheticUnit(pkg.id, decl.name, decl.role)
		if err != nil {
			return err
		}
		key := synthetic.Pkg().String() + "|" + synthetic.Name()
		if seen[key] {
			return &LoadError{Dir: u.request.Dir, Reason: "synthetic unit identity collision: " +
				synthetic.Pkg().String() + " declares " + synthetic.Name() + " twice"}
		}
		seen[key] = true
		graph.synthetics = append(graph.synthetics, synthetic)
	}
	collectDefinedObjects(node, pkg.typesInfo, graph.syntheticObj)
	return nil
}

// syntheticDecl is one named package-scope declaration of a generated decl.
type syntheticDecl struct {
	name string
	role SyntheticRole
}

// syntheticDecls enumerates the named package-scope declarations one generated
// checked declaration introduces. Import and unnamed forms yield nothing.
func syntheticDecls(node ast.Node) []syntheticDecl {
	var out []syntheticDecl
	switch n := node.(type) {
	case *ast.FuncDecl:
		if n.Name != nil && n.Name.Name != "" {
			out = append(out, syntheticDecl{funcDisplayName(n), SyntheticAdapter})
		}
	case *ast.GenDecl:
		switch n.Tok {
		case token.TYPE:
			for _, spec := range n.Specs {
				if ts, ok := spec.(*ast.TypeSpec); ok && ts.Name != nil {
					out = append(out, syntheticDecl{ts.Name.Name, SyntheticTypeDecl})
				}
			}
		case token.VAR, token.CONST:
			for _, spec := range n.Specs {
				if vs, ok := spec.(*ast.ValueSpec); ok {
					for _, name := range vs.Names {
						if name.Name != "" && name.Name != "_" {
							out = append(out, syntheticDecl{name.Name, SyntheticData})
						}
					}
				}
			}
		}
	}
	return out
}

// collectDefinedObjects records the objects a declaration defines.
func collectDefinedObjects(node ast.Node, info *types.Info, into map[types.Object]bool) {
	if info == nil {
		return
	}
	ast.Inspect(node, func(n ast.Node) bool {
		if ident, ok := n.(*ast.Ident); ok {
			if obj := info.Defs[ident]; obj != nil {
				into[obj] = true
			}
		}
		return true
	})
}

// requireTotalCoverage fails closed unless every unit of every cgo-original
// file has an exact checked counterpart — the origin graph is total.
func requireTotalCoverage(u *Universe, pkg *LoadedPackage, counterparts map[identity.SourceUnitID]checkedCounterpart) error {
	for _, file := range pkg.files {
		if !file.cgoOriginal {
			continue
		}
		for _, unit := range file.units {
			if _, ok := counterparts[unit.id]; !ok {
				return &LoadError{Dir: u.request.Dir, Reason: "unit " + unit.id.String() +
					" has no checked-view counterpart in the origin graph"}
			}
		}
	}
	return nil
}

// applyCDependence sets each cgo-original unit's C-dependence from typed
// evidence: the unit's checked counterpart uses a cgo-generated object.
func applyCDependence(u *Universe, pkg *LoadedPackage, graph *originGraph) error {
	for _, file := range pkg.files {
		if !file.cgoOriginal {
			continue
		}
		for i := range file.units {
			counterpart := graph.counterparts[file.units[i].id]
			file.units[i].cDependent = usesSyntheticObject(counterpart.node, pkg.typesInfo, graph.syntheticObj)
		}
	}
	return nil
}

// usesSyntheticObject reports whether a checked counterpart's OWN region uses
// any cgo-generated object. Nested function literals are separate units with
// their own C-dependence, so the walk halts at them — C-dependence is exact
// per unit, never inherited from a nested literal that touches C.
func usesSyntheticObject(node ast.Node, info *types.Info, synthetic map[types.Object]bool) bool {
	if node == nil || info == nil || len(synthetic) == 0 {
		return false
	}
	found := false
	ast.Inspect(node, func(n ast.Node) bool {
		if found || n == nil {
			return false
		}
		if lit, ok := n.(*ast.FuncLit); ok && ast.Node(lit) != node {
			return false // nested unit boundary
		}
		if ident, ok := n.(*ast.Ident); ok {
			if obj := info.Uses[ident]; obj != nil && synthetic[obj] {
				found = true
			}
		}
		return !found
	})
	return found
}

// checkedSpanOf measures a checked-view node's physical span.
func checkedSpanOf(u *Universe, node ast.Node) Span {
	start := u.fset.PositionFor(node.Pos(), false)
	end := u.fset.PositionFor(node.End(), false)
	return Span{
		Start: Position{Line: start.Line, Column: start.Column, Offset: start.Offset},
		End:   Position{Line: end.Line, Column: end.Column, Offset: end.Offset},
	}
}

// itoa renders a small non-negative int without importing strconv here.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
