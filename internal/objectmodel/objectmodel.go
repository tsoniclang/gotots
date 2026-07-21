// Package objectmodel recovers the inheritance hierarchy that a family
// of Go structs encodes with embedded base structs and a virtual-method
// interface (ADR-0012). It proves, per concrete struct, a single
// PRIMARY inheritance spine (emitted as native `extends`) and the
// SECONDARY embedded components, so the emitter produces native
// TypeScript classes and O(1) virtual dispatch instead of nested
// forwarding chains, boxed vtables, and exhaustive per-call switches.
//
// The analysis is a sealed whole-program product: it reads the typed
// universe once and yields an immutable plan. A struct that does not
// belong to a provable object-model family is absent from the plan and
// keeps its current representation.
package objectmodel

import (
	"go/types"
	"sort"
)

// Plan is the immutable object-model result for one translation unit.
type Plan struct {
	// classes maps a named struct's canonical identity to its class
	// shape (primary base, secondary components, virtual role).
	classes map[string]Class
	// families maps a virtual-contract interface identity to its
	// family (root type + members).
	families map[string]Family
}

// Class is one struct's recovered class shape.
type Class struct {
	// Name is the struct's own type name.
	Name string
	// PrimaryBase is the embedded field name whose type is this class's
	// superclass (`extends PrimaryBase`); empty for a family root.
	PrimaryBase string
	// PrimaryBaseType is the superclass's type name (empty for a root).
	PrimaryBaseType string
	// Secondary lists the other embedded named-struct fields, kept as
	// owned components with forwarding accessors.
	Secondary []string
	// Root marks the family root class (owns the virtual contract; has
	// no primary base within the family).
	Root bool
}

// Family is one virtual-contract interface's recovered hierarchy.
type Family struct {
	// Interface is the virtual-contract interface's type name.
	Interface string
	// RootType is the common type every member embeds transitively —
	// the base class of the family hierarchy.
	RootType string
	// Members are the concrete implementer type names, sorted.
	Members []string
	// Unplaced are members with more than one embedded base reaching
	// the root (genuine multiple inheritance): they keep their current
	// representation, and dispatch to them falls back to the union path.
	Unplaced []string
}

// Class returns a struct's recovered shape, if it belongs to a family.
func (p *Plan) Class(identity string) (Class, bool) {
	c, ok := p.classes[identity]
	return c, ok
}

// Families returns every recovered family's interface identity, sorted.
func (p *Plan) Families() []string {
	out := make([]string, 0, len(p.families))
	for id := range p.families {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

// Family returns one family by interface identity.
func (p *Plan) Family(interfaceIdentity string) (Family, bool) {
	f, ok := p.families[interfaceIdentity]
	return f, ok
}

// embeddedBases returns the named-struct types a struct embeds directly,
// in Go field order (an embedded field is anonymous and named by its
// type). Pointer-embedded bases are included (Go promotes through them).
func embeddedBases(named *types.Named) []*types.Named {
	strukt, ok := named.Underlying().(*types.Struct)
	if !ok {
		return nil
	}
	var bases []*types.Named
	for i := 0; i < strukt.NumFields(); i++ {
		field := strukt.Field(i)
		if !field.Embedded() {
			continue
		}
		t := field.Type()
		if pointer, isPointer := t.(*types.Pointer); isPointer {
			t = pointer.Elem()
		}
		base, isNamed := types.Unalias(t).(*types.Named)
		if !isNamed {
			continue
		}
		if _, isStruct := base.Underlying().(*types.Struct); !isStruct {
			continue
		}
		bases = append(bases, base)
	}
	return bases
}

// embedsTransitively reports whether `from` reaches `target` by any
// chain of embedded bases (target included as itself).
func embedsTransitively(from, target *types.Named, cache map[*types.Named]map[*types.Named]bool) bool {
	if from == target {
		return true
	}
	if seen, ok := cache[from]; ok {
		if reaches, done := seen[target]; done {
			return reaches
		}
	}
	if cache[from] == nil {
		cache[from] = map[*types.Named]bool{}
	}
	// Provisionally false to break embedding cycles (Go forbids them,
	// but the walk stays total either way).
	cache[from][target] = false
	for _, base := range embeddedBases(from) {
		if embedsTransitively(base, target, cache) {
			cache[from][target] = true
			return true
		}
	}
	return false
}

// Analyze recovers the object model from the unit's named types. A
// virtual-contract family is an interface with a broad closed
// implementer set whose members all embed a single common root struct
// through a unique primary spine; only such families enter the plan.
func Analyze(named []*types.Named) *Plan {
	plan := &Plan{classes: map[string]Class{}, families: map[string]Family{}}
	cache := map[*types.Named]map[*types.Named]bool{}

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
		// Members: structs whose pointer implements the interface.
		var members []*types.Named
		for _, s := range structs {
			if types.Implements(types.NewPointer(s), ifaceType) {
				members = append(members, s)
			}
		}
		if len(members) < 3 {
			// Too small to be an object-model family; ordinary dispatch.
			continue
		}
		root := commonRoot(members, cache)
		if root == nil {
			continue
		}
		family := buildFamily(iface, root, members, cache)
		if family == nil {
			continue
		}
		plan.families[canonical(iface)] = family.Family
		// Record every class on the spine (members and intermediates).
		for _, class := range family.classes {
			plan.classes[canonical(class.self)] = class.Class
		}
	}
	return plan
}

// commonRoot returns the unique struct every member embeds transitively
// that itself embeds no other common-embedded struct (the spine sink),
// or nil if there is no single such root.
func commonRoot(members []*types.Named, cache map[*types.Named]map[*types.Named]bool) *types.Named {
	// Candidate set: types embedded transitively by the FIRST member.
	var candidates []*types.Named
	collectEmbedded(members[0], &candidates, map[*types.Named]bool{})
	// Keep only those embedded by EVERY member.
	var common []*types.Named
	for _, cand := range candidates {
		all := true
		for _, m := range members[1:] {
			if !embedsTransitively(m, cand, cache) {
				all = false
				break
			}
		}
		if all {
			common = append(common, cand)
		}
	}
	// Root = the common candidate that embeds no OTHER common candidate.
	var roots []*types.Named
	for _, cand := range common {
		sink := true
		for _, other := range common {
			if other != cand && embedsTransitively(cand, other, cache) {
				sink = false
				break
			}
		}
		if sink {
			roots = append(roots, cand)
		}
	}
	if len(roots) != 1 {
		return nil
	}
	return roots[0]
}

// collectEmbedded gathers all structs a type embeds transitively.
func collectEmbedded(from *types.Named, out *[]*types.Named, seen map[*types.Named]bool) {
	for _, base := range embeddedBases(from) {
		if seen[base] {
			continue
		}
		seen[base] = true
		*out = append(*out, base)
		collectEmbedded(base, out, seen)
	}
}

type spineClass struct {
	self *types.Named
	Class
}

// buildFamily proves the primary spine for every struct that embeds the
// root, failing (nil) if any struct has more than one embedded base
// reaching the root (genuine multiple inheritance the plan cannot
// express).
func buildFamily(iface, root *types.Named, members []*types.Named, cache map[*types.Named]map[*types.Named]bool) *family {
	// The class set: root plus every struct transitively embedding it
	// that also appears on some member's spine.
	classSet := map[*types.Named]bool{root: true}
	for _, m := range members {
		var chain []*types.Named
		collectEmbedded(m, &chain, map[*types.Named]bool{})
		classSet[m] = true
		for _, c := range chain {
			if embedsTransitively(c, root, cache) {
				classSet[c] = true
			}
		}
	}
	out := &family{Family: Family{Interface: iface.Obj().Name(), RootType: root.Obj().Name()}}
	for s := range classSet {
		var primary []*types.Named
		var secondary []string
		if s != root {
			for _, base := range embeddedBases(s) {
				if embedsTransitively(base, root, cache) {
					primary = append(primary, base)
				} else {
					secondary = append(secondary, base.Obj().Name())
				}
			}
			if len(primary) != 1 {
				// More than one embedded base reaches the root (genuine
				// multiple inheritance) or none does: no single spine.
				// The struct keeps its current representation; it is
				// unplaced, not fatal to the family.
				out.Unplaced = append(out.Unplaced, s.Obj().Name())
				continue
			}
		}
		class := Class{Name: s.Obj().Name(), Secondary: secondary, Root: s == root}
		if len(primary) == 1 {
			class.PrimaryBase = primary[0].Obj().Name()
			class.PrimaryBaseType = primary[0].Obj().Name()
		}
		sort.Strings(class.Secondary)
		out.classes = append(out.classes, spineClass{self: s, Class: class})
	}
	for _, m := range members {
		out.Members = append(out.Members, m.Obj().Name())
	}
	sort.Strings(out.Members)
	return out
}

type family struct {
	Family
	classes []spineClass
}

func canonical(n *types.Named) string {
	if n.Obj().Pkg() == nil {
		return n.Obj().Name()
	}
	return n.Obj().Pkg().Path() + "." + n.Obj().Name()
}
