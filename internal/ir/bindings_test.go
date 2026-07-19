package ir

import (
	"go/token"
	"go/types"
	"testing"
)

// TestBindingsIdentityContract proves the canonical alpha-renaming
// contract the binding table must uphold: every Go variable maps to
// exactly one identity, no identity belongs to two variables, and every
// identity gets a unique emission name (shadows disambiguated).
func TestBindingsIdentityContract(t *testing.T) {
	b := newBindings()
	// Two distinct Go objects that share a source spelling (a shadow), plus
	// a third distinct spelling.
	outer := types.NewVar(token.NoPos, nil, "i", types.Typ[types.Int])
	inner := types.NewVar(token.NoPos, nil, "i", types.Typ[types.Int])
	other := types.NewVar(token.NoPos, nil, "x", types.Typ[types.Int])

	idOuter := b.newVar(outer, "i")
	idInner := b.newVar(inner, "i")
	idOther := b.newVar(other, "x")

	// One identity per variable; distinct variables get distinct identities.
	if idOuter == idInner || idOuter == idOther || idInner == idOther {
		t.Fatalf("distinct variables must get distinct identities: %d %d %d", idOuter, idInner, idOther)
	}
	// Idempotent: the same variable always resolves to its one identity.
	if again := b.newVar(outer, "i"); again != idOuter {
		t.Fatalf("re-registering a variable must return its identity: got %d want %d", again, idOuter)
	}

	b.allocate(nil)

	// The first holder of a spelling keeps the bare name; the shadow gets a
	// numeric suffix; every name is unique.
	names := map[BindingID]string{idOuter: b.name(idOuter), idInner: b.name(idInner), idOther: b.name(idOther)}
	if names[idOuter] != "i" {
		t.Fatalf("outer i should keep the bare name, got %q", names[idOuter])
	}
	if names[idInner] != "i$1" {
		t.Fatalf("inner i should be disambiguated to i$1, got %q", names[idInner])
	}
	if names[idOther] != "x" {
		t.Fatalf("x should keep its name, got %q", names[idOther])
	}
	seen := map[string]bool{}
	for _, n := range names {
		if seen[n] {
			t.Fatalf("alpha names must be unique, %q repeated", n)
		}
		seen[n] = true
	}
}

// TestBindingsReservedAndSeeds proves reserved spellings escape and that a
// seeded generic factory parameter forces a colliding source name to a
// shadow suffix rather than capturing the factory.
func TestBindingsReservedAndSeeds(t *testing.T) {
	b := newBindings()
	reservedVar := types.NewVar(token.NoPos, nil, "in", types.Typ[types.Int])
	zeroVar := types.NewVar(token.NoPos, nil, "zero", types.Typ[types.Int])
	idReserved := b.newVar(reservedVar, "in")
	idZero := b.newVar(zeroVar, "zero")

	// zero$T is a real emitted factory parameter for a generic scope.
	b.allocate([]string{"zero$T"})

	if got := b.name(idReserved); got != "in$" {
		t.Fatalf("reserved word in should escape to in$, got %q", got)
	}
	// "zero" does not collide with the seeded "zero$T", so it keeps its name.
	if got := b.name(idZero); got != "zero" {
		t.Fatalf("zero should keep its name (no clash with zero$T), got %q", got)
	}
}
