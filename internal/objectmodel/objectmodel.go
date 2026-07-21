// Package objectmodel recovers the inheritance hierarchy that a family
// of Go structs encodes with embedded base structs and a virtual-method
// interface (ADR-0012). It proves, per concrete struct, a single
// PRIMARY inheritance spine (emitted as native `extends`) and the
// SECONDARY embedded components, so the emitter produces native
// TypeScript classes and O(1) virtual dispatch instead of nested
// forwarding chains, boxed vtables, and exhaustive per-call switches.
//
// Recognition is by typed facts — a virtual-contract interface I is an
// object-model contract only when a ROOT struct declares a
// self-reference field of type I and the concrete implementers of I
// embed that root — never by a member count. Primary-spine selection
// reproduces Go's own field/method selection (minimum promotion depth;
// equal depth is ambiguous and fails closed). A family that cannot
// place every member atomically is blocked whole, never mixed.
package objectmodel

import (
	"go/types"
	"sort"
)

// Plan is the immutable object-model result for one translation unit.
type Plan struct {
	classes  map[string]Class
	families map[string]Family
	blocked  map[string]string
}

// EmbeddedEdge models one embedded field exactly (ADR-0012 4.2).
type EmbeddedEdge struct {
	Field       string
	BaseType    string
	BaseCanon   string
	Pointer     bool
	Index       int
	DepthToRoot int
}

// MethodDisposition is one method's translation approach on one class
// (ADR-0012 4.4 / receiver decision 5).
type MethodDisposition string

const (
	// MethodDeclared: a body declared on this class (no ancestor has it).
	MethodDeclared MethodDisposition = "declared"
	// MethodOverride: a body redefining an ancestor's method.
	MethodOverride MethodDisposition = "override"
	// MethodInherited: promoted through the primary base — emit nothing,
	// native inheritance carries it.
	MethodInherited MethodDisposition = "inherited"
	// MethodTrampolineRemoved: a root contract method that only
	// dispatches to the self-reference (Node.Name(){return n.data.Name()})
	// — it becomes the abstract virtual method, no body.
	MethodTrampolineRemoved MethodDisposition = "trampoline-removed"
	// MethodComponentDelegated: promoted through a secondary component —
	// emit a one-line forwarding accessor to the component.
	MethodComponentDelegated MethodDisposition = "component-delegated"
)

// Class is one struct's recovered class shape.
type Class struct {
	Name       string
	Canon      string
	Primary    EmbeddedEdge
	HasPrimary bool
	Secondary  []EmbeddedEdge
	Root       bool
	Family     string
	// Methods is every method reachable on this class keyed by name,
	// with its translation disposition. Complete by construction: a
	// method with no disposition is a planner defect.
	Methods map[string]MethodDisposition
}

// Family is one virtual-contract interface's recovered hierarchy.
type Family struct {
	Interface      string
	InterfaceCanon string
	RootType       string
	RootCanon      string
	SelfField      string
	Members        []string
}

func (p *Plan) Class(canon string) (Class, bool) { c, ok := p.classes[canon]; return c, ok }

func (p *Plan) Families() []string {
	out := make([]string, 0, len(p.families))
	for id := range p.families {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

func (p *Plan) Family(interfaceCanon string) (Family, bool) {
	f, ok := p.families[interfaceCanon]
	return f, ok
}

// Blocked returns recognized-but-unplaceable families and their reasons.
func (p *Plan) Blocked() map[string]string { return p.blocked }

type edge struct {
	base    *types.Named
	pointer bool
	index   int
}

// embeddedEdges returns a struct's embedded named-struct edges in field
// order, distinguishing pointer from value embedding.
func embeddedEdges(named *types.Named) []edge {
	strukt, ok := named.Underlying().(*types.Struct)
	if !ok {
		return nil
	}
	var out []edge
	for i := 0; i < strukt.NumFields(); i++ {
		field := strukt.Field(i)
		if !field.Embedded() {
			continue
		}
		t := field.Type()
		pointer := false
		if p, isPointer := t.(*types.Pointer); isPointer {
			t = p.Elem()
			pointer = true
		}
		base, isNamed := types.Unalias(t).(*types.Named)
		if !isNamed {
			continue
		}
		if _, isStruct := base.Underlying().(*types.Struct); !isStruct {
			continue
		}
		out = append(out, edge{base, pointer, i})
	}
	return out
}

// depthToRoot returns the minimum number of VALUE-embedding steps from
// `from` to `root` (0 when equal), or -1 when unreachable by value
// embedding. A pointer-embedded step carries no object identity and is
// not part of the inheritance spine.
func depthToRoot(from, root *types.Named, memo map[*types.Named]int) int {
	if from == root {
		return 0
	}
	if d, ok := memo[from]; ok {
		return d
	}
	memo[from] = -1
	best := -1
	for _, e := range embeddedEdges(from) {
		if e.pointer {
			continue
		}
		if d := depthToRoot(e.base, root, memo); d >= 0 {
			if best < 0 || d+1 < best {
				best = d + 1
			}
		}
	}
	memo[from] = best
	return best
}

func canonical(n *types.Named) string {
	if n.Obj().Pkg() == nil {
		return n.Obj().Name()
	}
	return n.Obj().Pkg().Path() + "." + n.Obj().Name()
}

// Analyze recovers the object model from the unit's named types.
func Analyze(named []*types.Named) *Plan {
	plan := &Plan{classes: map[string]Class{}, families: map[string]Family{}, blocked: map[string]string{}}

	var structs, interfaces []*types.Named
	for _, n := range named {
		switch n.Underlying().(type) {
		case *types.Struct:
			structs = append(structs, n)
		case *types.Interface:
			interfaces = append(interfaces, n)
		}
	}

	for _, iface := range interfaces {
		ifaceType := iface.Underlying().(*types.Interface)
		if ifaceType.NumMethods() == 0 {
			continue
		}
		// Recognition (4.1): the ROOT is the struct that declares a
		// self-reference field of this interface AND is value-embedded
		// by the interface's implementers. Cast-view aliases (a defined
		// type over the root that shares the field) are not embedded by
		// implementers and so are not selected.
		root, selfField := selfReferenceRoot(iface, ifaceType, structs)
		if root == nil {
			continue
		}
		var members []*types.Named
		for _, s := range structs {
			if s == root {
				continue
			}
			if types.Implements(types.NewPointer(s), ifaceType) && depthToRoot(s, root, map[*types.Named]int{}) >= 0 {
				members = append(members, s)
			}
		}
		if len(members) == 0 {
			continue
		}
		fam := Family{
			Interface: iface.Obj().Name(), InterfaceCanon: canonical(iface),
			RootType: root.Obj().Name(), RootCanon: canonical(root), SelfField: selfField,
		}
		classes, reason := placeClasses(iface, ifaceType, root, members)
		if reason != "" {
			// 4.5: no mixed family — block the whole family.
			plan.blocked[canonical(iface)] = reason
			continue
		}
		for _, m := range members {
			fam.Members = append(fam.Members, m.Obj().Name())
		}
		sort.Strings(fam.Members)
		plan.families[canonical(iface)] = fam
		for _, c := range classes {
			c.Family = canonical(iface)
			if prior, exists := plan.classes[c.Canon]; exists && prior.Family != c.Family {
				// 4.4: reject overlapping families with incompatible
				// plans rather than silently overwriting.
				plan.blocked[canonical(iface)] = "class " + c.Canon + " belongs to two families"
				delete(plan.families, canonical(iface))
				break
			}
			plan.classes[c.Canon] = c
		}
	}
	return plan
}

// selfReferenceRoot finds the struct that declares a field of exactly
// the interface type (the object-model self-reference) AND is embedded
// by the interface's implementers — never a cast-view alias that shares
// the field but is embedded by no implementer.
func selfReferenceRoot(iface *types.Named, ifaceType *types.Interface, structs []*types.Named) (*types.Named, string) {
	type candidate struct {
		root  *types.Named
		field string
	}
	var candidates []candidate
	for _, s := range structs {
		strukt := s.Underlying().(*types.Struct)
		for i := 0; i < strukt.NumFields(); i++ {
			field := strukt.Field(i)
			if field.Embedded() {
				continue
			}
			named, isNamed := types.Unalias(field.Type()).(*types.Named)
			if isNamed && named == iface {
				candidates = append(candidates, candidate{s, field.Name()})
				break
			}
		}
	}
	// Select the candidate value-embedded by the most implementers; a
	// cast-view alias is embedded by none.
	var best *types.Named
	var bestField string
	bestCount := 0
	for _, c := range candidates {
		count := 0
		for _, s := range structs {
			if s != c.root && types.Implements(types.NewPointer(s), ifaceType) &&
				depthToRoot(s, c.root, map[*types.Named]int{}) >= 0 {
				count++
			}
		}
		if count > bestCount {
			best, bestField, bestCount = c.root, c.field, count
		}
	}
	return best, bestField
}

// placeClasses builds the class plan for the root and every struct that
// value-embeds it, selecting each class's primary base by Go's minimum
// promotion depth. An equal-depth tie (genuine ambiguity) returns a
// non-empty reason so the family blocks (4.5).
func placeClasses(iface *types.Named, ifaceType *types.Interface, root *types.Named, members []*types.Named) ([]Class, string) {
	// Class set: root plus every struct on some member's value-embedding
	// spine to root.
	classSet := map[*types.Named]bool{root: true}
	for _, m := range members {
		classSet[m] = true
		collectSpine(m, root, classSet)
	}
	contract := map[string]bool{}
	for i := 0; i < ifaceType.NumMethods(); i++ {
		contract[ifaceType.Method(i).Name()] = true
	}
	var out []Class
	for s := range classSet {
		class := Class{Name: s.Obj().Name(), Canon: canonical(s), Root: s == root}
		if s != root {
			primary, secondary, reason := selectPrimary(s, root)
			if reason != "" {
				return nil, s.Obj().Name() + ": " + reason
			}
			class.Primary = primary
			class.HasPrimary = true
			class.Secondary = secondary
		} else {
			// The root's own embedded bases (if any) are components.
			for _, e := range embeddedEdges(s) {
				class.Secondary = append(class.Secondary, edgeOf(e, root))
			}
		}
		class.Methods = methodDispositions(s, root, contract, &class)
		out = append(out, class)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Canon < out[j].Canon })
	return out, ""
}

// collectSpine adds every struct on a value-embedding path from `from`
// to `root` into the set.
func collectSpine(from, root *types.Named, set map[*types.Named]bool) {
	for _, e := range embeddedEdges(from) {
		if e.pointer {
			continue
		}
		if depthToRoot(e.base, root, map[*types.Named]int{}) >= 0 {
			if !set[e.base] {
				set[e.base] = true
				collectSpine(e.base, root, set)
			}
		}
	}
}

// selectPrimary picks the value-embedded base with the unique minimum
// promotion depth to the root as the superclass; all other embedded
// bases become secondary components. An equal-depth tie among
// root-reaching value bases is genuine ambiguity.
func selectPrimary(s, root *types.Named) (EmbeddedEdge, []EmbeddedEdge, string) {
	var candidates []edge
	var others []edge
	for _, e := range embeddedEdges(s) {
		reaches := !e.pointer && depthToRoot(e.base, root, map[*types.Named]int{}) >= 0
		if reaches {
			candidates = append(candidates, e)
		} else {
			others = append(others, e)
		}
	}
	if len(candidates) == 0 {
		return EmbeddedEdge{}, nil, "no value-embedded base reaches the root"
	}
	// Minimum depth wins; equal minimum is ambiguous.
	best := candidates[0]
	bestDepth := depthToRoot(best.base, root, map[*types.Named]int{})
	tie := false
	for _, e := range candidates[1:] {
		d := depthToRoot(e.base, root, map[*types.Named]int{})
		switch {
		case d < bestDepth:
			best, bestDepth, tie = e, d, false
		case d == bestDepth:
			tie = true
		}
	}
	if tie {
		return EmbeddedEdge{}, nil, "ambiguous equal-depth primary bases (genuine diamond)"
	}
	secondary := make([]EmbeddedEdge, 0, len(candidates)+len(others)-1)
	for _, e := range candidates {
		if e.base != best.base {
			secondary = append(secondary, edgeOf(e, root))
		}
	}
	for _, e := range others {
		secondary = append(secondary, edgeOf(e, root))
	}
	sort.Slice(secondary, func(i, j int) bool { return secondary[i].BaseCanon < secondary[j].BaseCanon })
	return edgeOf(best, root), secondary, ""
}

func edgeOf(e edge, root *types.Named) EmbeddedEdge {
	return EmbeddedEdge{
		Field: e.base.Obj().Name(), BaseType: e.base.Obj().Name(), BaseCanon: canonical(e.base),
		Pointer: e.pointer, Index: e.index, DepthToRoot: depthToRoot(e.base, root, map[*types.Named]int{}),
	}
}

// declaredMethods returns the method names a named type declares
// directly (value or pointer receiver), not promoted.
func declaredMethods(named *types.Named) map[string]bool {
	out := map[string]bool{}
	for i := 0; i < named.NumMethods(); i++ {
		out[named.Method(i).Name()] = true
	}
	return out
}

// methodDispositions assigns every reachable method its translation
// approach: root contract methods are trampolines removed to abstract;
// a directly-declared method is declared or (if an ancestor has it)
// override; a method promoted through the primary base is inherited;
// through a secondary component it is component-delegated.
func methodDispositions(s, root *types.Named, contract map[string]bool, class *Class) map[string]MethodDisposition {
	out := map[string]MethodDisposition{}
	own := declaredMethods(s)

	if class.Root {
		for name := range own {
			if contract[name] {
				out[name] = MethodTrampolineRemoved
			} else {
				out[name] = MethodDeclared
			}
		}
		return out
	}

	// Ancestor declared-method names along the primary spine.
	ancestor := map[string]bool{}
	for base := primaryBaseNamed(s, root); base != nil; base = primaryBaseNamed(base, root) {
		for n := range declaredMethods(base) {
			ancestor[n] = true
		}
		if base == root {
			break
		}
	}
	for name := range own {
		if ancestor[name] {
			out[name] = MethodOverride
		} else {
			out[name] = MethodDeclared
		}
	}
	// Promoted methods: inherited through the primary spine, delegated
	// through a secondary component.
	mset := types.NewMethodSet(types.NewPointer(s))
	for i := 0; i < mset.Len(); i++ {
		sel := mset.At(i)
		name := sel.Obj().Name()
		if _, done := out[name]; done {
			continue
		}
		if promotedThroughPrimary(s, root, sel.Index()) {
			out[name] = MethodInherited
		} else {
			out[name] = MethodComponentDelegated
		}
	}
	return out
}

// primaryBaseNamed returns the value-embedded base of s selected as its
// primary superclass (minimum depth to root), or nil.
func primaryBaseNamed(s, root *types.Named) *types.Named {
	if s == root {
		return nil
	}
	var best *types.Named
	bestDepth := -1
	for _, e := range embeddedEdges(s) {
		if e.pointer {
			continue
		}
		d := depthToRoot(e.base, root, map[*types.Named]int{})
		if d < 0 {
			continue
		}
		if bestDepth < 0 || d < bestDepth {
			best, bestDepth = e.base, d
		}
	}
	return best
}

// promotedThroughPrimary reports whether a selection index path enters
// through the struct's primary base field.
func promotedThroughPrimary(s, root *types.Named, index []int) bool {
	if len(index) == 0 {
		return true
	}
	primary := primaryBaseNamed(s, root)
	if primary == nil {
		return false
	}
	edges := embeddedEdges(s)
	first := index[0]
	if first < 0 || first >= s.Underlying().(*types.Struct).NumFields() {
		return false
	}
	// index[0] is a field index of s; match it to the primary base's edge.
	for _, e := range edges {
		if e.index == first {
			return e.base == primary
		}
	}
	return false
}
