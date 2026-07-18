// The dynamic-type universe has an explicit lifecycle: COLLECTING (the
// pre-pass adds named universe types) then SEALED (immutable; unions
// resolve over the complete closed world). Mutation of the named universe
// after sealing, box-site composite observation before sealing, or a union
// cached before sealing are each construction defects that fail closed
// (panic), so the ordering is enforced structurally rather than by
// call-order convention.
package ir

import (
	"go/token"
	"go/types"
	"testing"
)

func externName(path, name string) *types.TypeName {
	pkg := types.NewPackage(path, "p")
	tn := types.NewTypeName(token.NoPos, pkg, name, nil)
	types.NewNamed(tn, types.NewStruct(nil, nil), nil)
	return tn
}

func mustPanic(t *testing.T, what string, fn func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatalf("%s: expected a panic, got none", what)
		}
	}()
	fn()
}

func TestUniverseLifecycleEnforcement(t *testing.T) {
	s := NewScope("example.com/owned")
	if s.UniverseSealed() {
		t.Fatal("a fresh scope must start unsealed (collecting)")
	}

	// Collecting phase: adding universe types is allowed; box observations
	// and union caching are premature.
	s.AddExternConcrete(externName("example.com/x", "T"))
	mustPanic(t, "AddBoxedComposite before seal", func() {
		s.AddBoxedComposite("c:example.com/x.T", Type{}, &EqPlan{Kind: EqIdentity})
	})
	mustPanic(t, "SetIfaceMemberCache before seal", func() {
		s.SetIfaceMemberCache("iface", nil)
	})

	s.SealUniverse()
	if !s.UniverseSealed() {
		t.Fatal("SealUniverse must finalize the universe")
	}

	// Sealed phase: the named universe is immutable; box observations and
	// union caching are now the only permitted writes.
	mustPanic(t, "AddConcreteType after seal", func() {
		s.AddConcreteType(externName("example.com/y", "U"))
	})
	mustPanic(t, "AddExternConcrete after seal", func() {
		s.AddExternConcrete(externName("example.com/y", "U"))
	})
	s.AddBoxedComposite("c:example.com/x.T", Type{}, &EqPlan{Kind: EqIdentity})
	s.SetIfaceMemberCache("iface", nil)
}
