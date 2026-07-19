// Interface-union alias definitions: a deferred, deduplicated named
// alias per distinct interface identity, its member-payload spelling,
// and the withheld-reference filtering that keeps a union closed over
// only its materialized implementers.
package emit

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/tsoniclang/gotots/internal/ir"
)

func ifaceAliasName(identity string) string {
	digest := sha256.Sum256([]byte(identity))
	return "Iface$" + hex.EncodeToString(digest[:])
}

// ifaceUnionAlias spells one interface type as its closed discriminated
// union alias (ADR-0004): one GoBox member per implementer with exact
// literal, payload, and vtable types, undefined for nil, and — for the
// empty interface — the predeclared and composite members. Payload and
// vtable references use type-only symbols, so alias spelling adds no
// runtime import edge.
func (p *printer) ifaceUnionAlias(t ir.Type) (string, error) {
	identity := t.IfaceID
	if identity == "" {
		// The canonical interface identity is the ONLY key: falling back
		// to the Go spelling would let two structurally distinct
		// interfaces with equal spellings share one alias. An empty
		// identity is a construction defect, not a spelling to guess.
		return "", fmt.Errorf("interface %q has no canonical identity", t.Go)
	}
	name := ifaceAliasName(identity)
	if p.module == nil {
		return name, nil
	}
	if prior, exists := p.module.identityOf(name); exists {
		if prior != identity {
			return "", fmt.Errorf("interface alias digest collision: %q names both %q and %q", name, prior, identity)
		}
		return name, nil
	}
	p.module.ifaceIdentity[name] = identity
	// Reserve the name and DEFER the definition. An external union member's
	// vtable type is its adapter arrow type, which is only fully populated
	// after newModule builds every external adapter. Spelling the definition
	// eagerly here — e.g. while building an adapter type whose result IS this
	// interface — would cache an incomplete Record<never,never> vtable that
	// the idempotent registration then locks in. finalizeUnionAliases builds
	// every reserved alias's definition once, after all adapter types exist.
	p.module.RegisterIfaceAlias(name, "")
	p.module.ifaceAliasTypes[name] = t
	return name, nil
}

// buildUnionAliasDefinition spells one reserved union alias's declaration
// from its interface type. It is called only after every external adapter
// type is populated, so external members carry their exact vtable type.
func (p *printer) buildUnionAliasDefinition(name string, t ir.Type) (string, error) {
	members := []string{"undefined"}
	for _, member := range p.retainedMembers(t) {
		payload, err := p.memberPayload(member)
		if err != nil {
			return "", err
		}
		var vtable string
		if member.Composite != "" {
			vtable, err = p.instVtableType(member.InstType, member.InstSlots, member.Pointer)
			if err != nil {
				return "", err
			}
			members = append(members, fmt.Sprintf("goif$.GoBox<%q, %s, %s>", member.K, payload, vtable))
			continue
		}
		if member.Extern {
			// External implementers carry inline stub-adapter vtables;
			// the member type is the exact structural adapter surface.
			entries := []string{}
			for _, method := range p.module.ExternMethods[member.Pkg+"."+member.Type] {
				adapterType := method.AdapterType
				if member.Pointer {
					adapterType = method.AdapterPtrType
				}
				if adapterType != "" {
					slot := requireIdentity(method.Slot, "external vtable-type slot for "+method.Name)
					entries = append(entries, slot+": "+adapterType)
				}
			}
			vtable = "{ " + joinComma(entries) + " }"
			if len(entries) == 0 {
				vtable = "Record<never, never>"
			}
		} else {
			suffix := "$vtable"
			if member.Pointer {
				suffix = "$vtablePtr"
			}
			ref, err := p.module.typeSymbol(member.Pkg, member.Type+suffix)
			if err != nil {
				return "", err
			}
			vtable = "typeof " + ref
		}
		members = append(members, fmt.Sprintf("goif$.GoBox<%q, %s, %s>", member.K, payload, vtable))
	}
	if t.IfaceEmpty {
		// The empty interface accepts every predeclared type and every
		// composite type ACTUALLY BOXED in the unit — each an exact
		// member, so composite assertions narrow without any cast
		// (ADR-0004; the enumeration is closed by construction).
		for _, member := range predeclaredMembers {
			members = append(members, fmt.Sprintf("goif$.GoBox<%q, %s, Record<never, never>>", "p:"+member.name, member.payload))
		}
		for _, composite := range p.module.BoxedComposites {
			if p.referencesWithheldType(composite.T) {
				// A composite whose spelling would reference a withheld
				// package cannot appear as an exact member here: no runnable
				// module defines its type, so nothing of it can box at
				// runtime and its file is absent. It is not a member of this
				// bundle's empty-interface union.
				continue
			}
			payload, err := p.tsType(composite.T)
			if err != nil {
				return "", err
			}
			members = append(members, fmt.Sprintf("goif$.GoBox<%q, %s, Record<never, never>>", "c:"+composite.Canon, payload))
		}
	}
	declaration := "type " + name + " = " + strings.Join(members, " | ") + ";"
	p.module.RegisterIfaceEqFn(name, p.ifaceEqFn(t, name))
	if p.module.ifaceKeyRequired(name) {
		keyFn, err := p.ifaceKeyFn(t, name)
		if err != nil {
			return "", err
		}
		p.module.RegisterIfaceEqFn(name+"$key", keyFn)
	}
	return declaration, nil
}

// finalizeUnionAliases builds every reserved union alias's deferred
// definition, after newModule has populated all external adapter types.
// The index loop covers aliases newly reserved while building an earlier
// one (an empty-interface member's boxed composite may denote a further
// interface). A reserved name without a stored type is a construction defect.
func finalizeUnionAliases(module *Module) error {
	p := &printer{module: module}
	for i := 0; i < len(module.aliasOrder); i++ {
		name := module.aliasOrder[i]
		t, ok := module.ifaceAliasTypes[name]
		if !ok {
			return fmt.Errorf("emit: union alias %s reserved without a stored interface type", name)
		}
		declaration, err := p.buildUnionAliasDefinition(name, t)
		if err != nil {
			return err
		}
		module.ifaceAliases[name] = declaration
	}
	return nil
}

// memberPayload spells one union member's payload by name: class
// instances are identity carriers (the pointer IS the instance, nilable
// when boxed through a pointer); named value carriers box the value or
// its cell; external handles are branded and nilable through pointers.
func (p *printer) memberPayload(member ir.IfaceMember) (string, error) {
	if member.Composite != "" {
		// An instantiated-generic member: the payload IS the exact
		// instance type (identity for pointer members over struct
		// underlyings, the instance value otherwise — the spelling
		// carries both).
		base, err := p.tsType(member.InstType)
		if err != nil {
			return "", err
		}
		if member.Pointer && !member.Struct {
			return "(gort$.GoCell<" + base + "> | undefined)", nil
		}
		if member.Pointer {
			return "(" + base + " | undefined)", nil
		}
		return base, nil
	}
	if member.Extern {
		if member.ExternCarrier != "" {
			// Basic-underlying external named type: the payload is its
			// exact value carrier.
			if member.Pointer {
				return "(gort$.GoCell<" + member.ExternCarrier + "> | undefined)", nil
			}
			return member.ExternCarrier, nil
		}
		handle := fmt.Sprintf("goext$.GoExtern<%q>", member.Pkg+"."+member.Type)
		if member.Pointer {
			return "(" + handle + " | undefined)", nil
		}
		return handle, nil
	}
	base, err := p.module.typeSymbol(member.Pkg, member.Type)
	if err != nil {
		return "", err
	}
	if !member.Pointer {
		return base, nil
	}
	if member.Struct {
		return "(" + base + " | undefined)", nil
	}
	return "(gort$.GoCell<" + base + "> | undefined)", nil
}

// referencesWithheldType reports whether an ir.Type's spelling would
// reference a package whose module is withheld from the bundle: such a
// type has no runnable module and its file is absent, so it can neither
// box at runtime nor be imported.
func (p *printer) referencesWithheldType(t ir.Type) bool {
	if p.module == nil || p.module.Withheld == nil {
		return false
	}
	var walk func(ir.Type) bool
	walk = func(t ir.Type) bool {
		if t.Pkg != "" && t.Kind != ir.KindExternal && p.module.Withheld(t.Pkg) {
			return true
		}
		if t.Elem != nil && walk(*t.Elem) {
			return true
		}
		if t.Key != nil && walk(*t.Key) {
			return true
		}
		for _, arg := range t.TypeArgs {
			if walk(arg) {
				return true
			}
		}
		if t.Sig != nil {
			for _, param := range t.Sig.Params {
				if walk(param) {
					return true
				}
			}
			for _, result := range t.Sig.Results {
				if walk(result) {
					return true
				}
			}
		}
		// An interface-typed component is SPELLED as the filtered local
		// union alias, which already excludes withheld members
		// (retainedMembers): the component's own pre-filter member list
		// must not disqualify the enclosing type — asking it here dropped
		// self-referential empty-interface composites ([]any) from the
		// union whenever ANY corpus package was withheld, turning their
		// asserts into statically-false panics.
		return false
	}
	return walk(t)
}

// retainedMembers filters an interface's implementer union to the
// packages retained in this bundle: a withheld package's class exists
// in no runnable module, so nothing of that type can box at runtime and
// no alias or dispatch branch may reference it.
func (p *printer) retainedMembers(t ir.Type) []ir.IfaceMember {
	if p.module == nil || p.module.Withheld == nil {
		return t.IfaceMembers
	}
	out := make([]ir.IfaceMember, 0, len(t.IfaceMembers))
	for _, member := range t.IfaceMembers {
		if !member.Extern && p.module.Withheld(member.Pkg) {
			continue
		}
		if member.Composite != "" && p.referencesWithheldType(member.InstType) {
			continue
		}
		out = append(out, member)
	}
	return out
}

// ifaceKeyFn generates one union's map-key encoder for the admitted
// pointer-member subset: discriminant + pointer identity, exactly Go's
// (dynamic type, pointer) key equality. The nil interface is its own key.
func (p *printer) ifaceKeyFn(t ir.Type, name string) (string, error) {
	var b strings.Builder
	fmt.Fprintf(&b, "export function %s$key($v: %s): string {\n", name, name)
	b.WriteString("  if ($v === undefined) return \"n\";\n")
	members := p.retainedMembers(t)
	if len(members) == 0 {
		b.WriteString("  return gort$.goPanicUnreachableType(\"key\");\n}\n")
		return b.String(), nil
	}
	b.WriteString("  switch ($v.k) {\n")
	for _, member := range members {
		component, err := p.memberKeyComponent(member)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&b, "    case %q: return %q + %s;\n", member.K, member.K+"|", component)
	}
	if t.IfaceEmpty {
		for _, member := range predeclaredMembers {
			component := scalarKeyComponentFor(member.payload, "$v.v")
			fmt.Fprintf(&b, "    case %q: return %q + %s;\n", "p:"+member.name, "p:"+member.name+"|", component)
		}
		for _, composite := range p.module.BoxedComposites {
			if p.referencesWithheldType(composite.T) {
				continue
			}
			component := compositeKeyComponent(composite)
			fmt.Fprintf(&b, "    case %q: return %s;\n", "c:"+composite.Canon, component)
		}
	}
	b.WriteString("    default: return gort$.goPanicUnreachableType(\"key\");\n  }\n}\n")
	return b.String(), nil
}

// memberKeyComponent spells one named member's key encoding: pointer
// identity, the exact scalar carrier, or a key-encodable struct's goKey$.
func (p *printer) memberKeyComponent(member ir.IfaceMember) (string, error) {
	switch {
	case member.Pointer:
		return "gort$.goKeyId($v.v)", nil
	case member.Extern && member.ExternCarrier == "number":
		return "gort$.goKeyFloat($v.v)", nil
	case member.Extern && member.ExternCarrier != "":
		return scalarKeyComponentFor(member.ExternCarrier, "$v.v"), nil
	case member.Extern:
		if member.Eq != nil && member.Eq.Kind == ir.EqUncomparable {
			// Go's exact runtime behavior for hashing this dynamic type.
			return fmt.Sprintf("gort$.goKeyUnhashable(%q)", member.Eq.Display), nil
		}
		// A comparable opaque handle: the claim (verified at finalize
		// against the corpus-complete value-box log) is that no VALUE of
		// this type is ever boxed, so the branch is machine-unreachable.
		p.module.ClaimKeyUnreachable(member.K)
		return fmt.Sprintf("gort$.goKeyUnreachable(%q)", member.K), nil
	case member.Struct:
		if !member.KeyEncodable {
			if member.Eq != nil && member.Eq.Kind == ir.EqUncomparable {
				return fmt.Sprintf("gort$.goKeyUnhashable(%q)", member.Eq.Display), nil
			}
			// Comparable in Go but without a generated encoding: the
			// machine-verified claim is that no VALUE of this type is
			// ever boxed (finalize checks the corpus-complete log).
			p.module.ClaimKeyUnreachable(member.K)
			return fmt.Sprintf("gort$.goKeyUnreachable(%q)", member.K), nil
		}
		return "$v.v.goKey$()", nil
	}
	return scalarKeyComponentFor(member.ValueCarrier, "$v.v"), nil
}

// scalarKeyComponentFor selects the statically typed key encoder for a
// known scalar carrier spelling.
func scalarKeyComponentFor(carrier, access string) string {
	switch carrier {
	case "string":
		return "gort$.goKeyString(" + access + ")"
	case "boolean", "bool":
		return "gort$.goKeyBool(" + access + ")"
	case "number", "float64", "float32":
		return "gort$.goKeyFloat(" + access + ")"
	}
	// Every remaining scalar carrier is an integer form (bigint or
	// number-carried narrow int).
	return "gort$.goKeyInt(" + access + ")"
}

// compositeKeyComponent spells one boxed composite's key: pointer
// composites by identity, primitive arrays element-wise, and every
// uncomparable composite through Go's exact unhashable panic.
func compositeKeyComponent(composite BoxedComposite) string {
	prefix := fmt.Sprintf("%q + ", "c:"+composite.Canon+"|")
	if composite.Eq != nil {
		switch composite.Eq.Kind {
		case ir.EqIdentity:
			return prefix + "gort$.goKeyId($v.v)"
		case ir.EqArray:
			return prefix + "gort$.goKeyArray($v.v, ($e) => " + scalarKeyComponentFor(compositeElemCarrier(composite), "$e") + ")"
		}
	}
	return fmt.Sprintf("gort$.goKeyUnhashable(%q)", composite.T.Go)
}

// compositeElemCarrier names a primitive-array composite's element
// carrier for the typed key encoder.
func compositeElemCarrier(composite BoxedComposite) string {
	if composite.T.Elem == nil {
		return ""
	}
	switch {
	case composite.T.Elem.Kind == ir.KindString:
		return "string"
	case composite.T.Elem.Kind == ir.KindBool:
		return "boolean"
	case composite.T.Elem.Kind.Float():
		return "number"
	}
	return "int"
}
