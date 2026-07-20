// Transactional body emission: each body prints into an overlay of its
// module — reads consult the overlay then its parent, writes land in the
// overlay only — and the overlay commits to the parent only after the
// body emitted completely. An abandoned overlay leaves the module exactly
// as it was, so a failed body can never leak imports, alias
// reservations, init edges, or equality helpers into the emitted file:
// the transaction is structural, not a snapshot/restore.
package emit

import "github.com/tsoniclang/gotots/internal/ir"

// Overlay returns a transactional child of m. The child shares m's
// immutable configuration (package identity, importable set, external
// method tables, boxed composites, withholding predicate) and records
// every mutation privately until Commit.
func (m *Module) Overlay() *Module {
	return &Module{
		Pkg:             m.Pkg,
		PkgName:         m.PkgName,
		ABI:             m.ABI,
		ExternMethods:   m.ExternMethods,
		imports:         m.imports,
		used:            map[string]bool{},
		typeUsed:        map[string]bool{},
		ifaceAliases:    map[string]string{},
		ifaceIdentity:   map[string]string{},
		ifaceAliasTypes: map[string]ir.Type{},
		ifaceEqFns:      map[string]string{},
		ifaceKeyFns:     map[string]bool{},
		initEdges:       map[string]bool{},
		BoxedComposites: m.BoxedComposites,
		Withheld:        m.Withheld,
		parent:          m,
	}
}

// Commit applies the overlay's recorded additions to its parent in
// recording order (alias order is first-registration order, so the
// emitted alias block stays deterministic).
func (m *Module) Commit() {
	p := m.parent
	p.emissions = append(p.emissions, m.emissions...)
	p.externSymbols = append(p.externSymbols, m.externSymbols...)
	for pkg := range m.used {
		p.used[pkg] = true
	}
	for pkg := range m.typeUsed {
		p.typeUsed[pkg] = true
	}
	for pkg := range m.initEdges {
		p.initEdges[pkg] = true
	}
	for _, name := range m.aliasOrder {
		p.aliasOrder = append(p.aliasOrder, name)
		p.ifaceAliases[name] = m.ifaceAliases[name]
		p.ifaceIdentity[name] = m.ifaceIdentity[name]
		p.ifaceAliasTypes[name] = m.ifaceAliasTypes[name]
	}
	for name, decl := range m.ifaceEqFns {
		p.RegisterIfaceEqFn(name, decl)
	}
	for name := range m.ifaceKeyFns {
		p.RequireIfaceKeyFn(name)
	}
}

// identityOf resolves an alias name to its recorded canonical interface
// identity through the overlay chain.
func (m *Module) identityOf(name string) (string, bool) {
	for cur := m; cur != nil; cur = cur.parent {
		if identity, ok := cur.ifaceIdentity[name]; ok {
			return identity, true
		}
	}
	return "", false
}

// aliasReserved reports whether an alias name is reserved anywhere in the
// overlay chain.
func (m *Module) aliasReserved(name string) bool {
	for cur := m; cur != nil; cur = cur.parent {
		if _, ok := cur.ifaceAliases[name]; ok {
			return true
		}
	}
	return false
}
