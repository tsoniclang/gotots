package catalog

import "testing"

// TestKindTableIsTotal proves the descriptor table is exact-size and complete:
// every valid Kind has a non-empty name and a valid category, and the count of
// valid kinds equals the terminal sentinel minus the invalid zero value. A
// Kind added to the enum without a descriptor leaves a zero-value entry that
// fails here.
func TestKindTableIsTotal(t *testing.T) {
	all := All()
	if got, want := len(all), int(numKinds)-1; got != want {
		t.Fatalf("All() returned %d kinds, want %d (numKinds-1)", got, want)
	}
	seenNames := map[string]Kind{}
	for _, k := range all {
		if !k.Valid() {
			t.Errorf("All() yielded invalid kind %d", uint16(k))
			continue
		}
		name := k.Name()
		if name == "" {
			t.Errorf("kind %d has empty descriptor name", uint16(k))
		}
		if prior, dup := seenNames[name]; dup {
			t.Errorf("kinds %d and %d share name %q", uint16(prior), uint16(k), name)
		}
		seenNames[name] = k
		if !k.Category().Valid() {
			t.Errorf("kind %s has invalid category %d", name, uint8(k.Category()))
		}
	}
}

// TestKindSentinelsInvalid proves the boundary values are not usable kinds.
func TestKindSentinelsInvalid(t *testing.T) {
	if KindInvalid.Valid() {
		t.Error("KindInvalid must not be valid")
	}
	if numKinds.Valid() {
		t.Error("terminal sentinel numKinds must not be valid")
	}
	if got := KindInvalid.Name(); got != "" {
		t.Errorf("KindInvalid.Name() = %q, want empty", got)
	}
	if got := numKinds.String(); got == "" {
		t.Error("String() of the sentinel must still render for diagnostics")
	}
}

// TestCategoryClosed proves the category enum is closed with a terminal
// sentinel and named members.
func TestCategoryClosed(t *testing.T) {
	for c := CategoryInvalid + 1; c < numCategories; c++ {
		if !c.Valid() {
			t.Errorf("category %d should be valid", uint8(c))
		}
		if categoryNames[c] == "" {
			t.Errorf("category %d has no name", uint8(c))
		}
	}
	if CategoryInvalid.Valid() || numCategories.Valid() {
		t.Error("category sentinels must not be valid")
	}
}
