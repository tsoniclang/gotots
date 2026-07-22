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
