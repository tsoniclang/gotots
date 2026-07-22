package identity

import "testing"

// pinnedUnitKindIDs freezes the permanent unit-kind identities; a renumber or
// insertion fails here.
var pinnedUnitKindIDs = map[UnitKind]uint8{
	UnitFuncBody: 1, UnitFuncLitBody: 2, UnitVarInitializer: 3,
	UnitBodylessDecl: 4, UnitImplicitExecutable: 5,
}

func TestUnitKindIDsArePinned(t *testing.T) {
	if len(pinnedUnitKindIDs) != int(numUnitKinds)-1 {
		t.Fatalf("pinned kinds cover %d, want %d", len(pinnedUnitKindIDs), numUnitKinds-1)
	}
	seen := map[uint8]bool{}
	for kind, want := range pinnedUnitKindIDs {
		if uint8(kind) != want {
			t.Errorf("unit kind %s has id %d, pinned to %d", kind, uint8(kind), want)
		}
		if seen[want] {
			t.Errorf("unit-kind id %d assigned twice", want)
		}
		seen[want] = true
		if !kind.Valid() || kind.String() == "" {
			t.Errorf("unit kind %d not valid/named", want)
		}
	}
	if UnitInvalid.Valid() || UnitKind(numUnitKinds).Valid() {
		t.Error("unit-kind sentinels must not be valid")
	}
	// The implicit-executable kind never constructs a span-bearing identity.
	m, _ := NewModuleID("gate.example/m", "")
	owner, _ := NewModuleOwner(m)
	f, _ := NewFileID(owner, "m.go")
	span, _ := NewSpanID(f, 0, 1)
	if _, err := NewSourceUnitID(span, UnitImplicitExecutable); err == nil {
		t.Error("implicit-executable accepted a fabricated source span")
	}
}

// TestImplicitUnitOpIDsArePinned pins the implicit-operation identities
// permanently; renumbering is an artifact-compatibility break.
func TestImplicitUnitOpIDsArePinned(t *testing.T) {
	if uint8(ImplicitOpPackageInit) != 1 {
		t.Errorf("ImplicitOpPackageInit = %d, want pinned 1", uint8(ImplicitOpPackageInit))
	}
	module, err := NewModuleID("pin.example/m", "")
	if err != nil {
		t.Fatal(err)
	}
	owner, err := NewModuleOwner(module)
	if err != nil {
		t.Fatal(err)
	}
	pkg, err := NewPackageID(owner, "pin.example/m")
	if err != nil {
		t.Fatal(err)
	}
	id, err := NewImplicitUnitID(pkg, ImplicitOpPackageInit)
	if err != nil {
		t.Fatal(err)
	}
	if id.String() != "mod=pin.example/m::pin.example/m#implicit/package-init" {
		t.Errorf("canonical form = %s", id.String())
	}
	if _, err := NewImplicitUnitID(pkg, ImplicitUnitOp(99)); err == nil {
		t.Error("invalid implicit op accepted")
	}
	if _, err := NewImplicitUnitID(PackageID{}, ImplicitOpPackageInit); err == nil {
		t.Error("zero package accepted")
	}
}
