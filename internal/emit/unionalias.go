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
		for _, member := range t.IfaceMembers {
			if !member.Extern && member.Pkg != "" && p.module.Withheld(member.Pkg) {
				return true
			}
		}
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
		out = append(out, member)
	}
	return out
}
