// The field storage key is a RUNTIME-CARRIER identity (it decides whether
// a field is materialized as a stable GoCell). Two structurally spelling-
// alike anonymous structs whose unexported fields come from DIFFERENT
// packages are distinct types, so their field storage keys must differ —
// keying by t.String() (which does not package-qualify unexported field
// names) would collide them and bleed one package's address-taken analysis
// into the other's.
package ir

import (
	"go/token"
	"go/types"
	"testing"
)

func anonStructFieldKey(t *testing.T, pkgPath, field string) string {
	t.Helper()
	pkg := types.NewPackage(pkgPath, "p")
	v := types.NewVar(token.NoPos, pkg, field, types.Typ[types.Int])
	st := types.NewStruct([]*types.Var{v}, nil)
	key, err := fieldStorageKey(st, field)
	if err != nil {
		t.Fatalf("fieldStorageKey(%s): %v", pkgPath, err)
	}
	return key
}

func TestAnonStructFieldKeyDistinctAcrossPackages(t *testing.T) {
	// Two struct{ x int } with unexported x from different packages.
	a := anonStructFieldKey(t, "example.com/a", "x")
	b := anonStructFieldKey(t, "example.com/b", "x")
	if a == b {
		t.Fatalf("anon struct field keys collided across packages: %q == %q", a, b)
	}
	// Same package, same field: stable/identical.
	a2 := anonStructFieldKey(t, "example.com/a", "x")
	if a != a2 {
		t.Fatalf("anon struct field key is not stable: %q != %q", a, a2)
	}
}

func TestExportedAnonStructFieldKeyStableAcrossPackages(t *testing.T) {
	// An EXPORTED field is not package-qualified in Go's identity, so two
	// struct{ X int } coincide regardless of declaring package.
	a := anonStructFieldKey(t, "example.com/a", "X")
	b := anonStructFieldKey(t, "example.com/b", "X")
	if a != b {
		t.Fatalf("exported-field anon struct keys should coincide: %q != %q", a, b)
	}
}
