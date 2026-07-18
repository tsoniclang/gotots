package ir

import "testing"

// The enum's own bounds are the single authoritative definition. These
// pins prove that a newly declared kind cannot evade validation: the
// range [KindUnsupportedInvalid+1, kindEnd) covers every real kind, and
// the name table must be exactly total over that range — an added const
// with no kindName entry fails both the membership loop and the count
// equality, and a String() fallthrough is impossible for a real kind.

func TestAllUnsupportedKindsCoversTheEnumRange(t *testing.T) {
	got := AllUnsupportedKinds()
	want := int(kindEnd) - 1 // exclude KindUnsupportedInvalid(0); kindEnd is exclusive
	if len(got) != want {
		t.Fatalf("AllUnsupportedKinds returned %d kinds; the enum range has %d", len(got), want)
	}
	for i, k := range got {
		if int(k) != i+1 {
			t.Fatalf("AllUnsupportedKinds[%d] = %d; range must be dense from 1", i, k)
		}
	}
}

func TestKindNameIsTotalOverTheEnum(t *testing.T) {
	// Every real kind has a stable class key; String() never falls through
	// to the invalid sentinel. len(kindName) must equal the range exactly,
	// so neither a missing entry nor a phantom entry survives.
	if len(kindName) != int(kindEnd)-1 {
		t.Fatalf("kindName has %d entries; the enum range has %d — a kind is missing or extra",
			len(kindName), int(kindEnd)-1)
	}
	for _, k := range AllUnsupportedKinds() {
		name, ok := kindName[k]
		if !ok {
			t.Errorf("kind %d has no kindName entry", k)
			continue
		}
		if name == "" {
			t.Errorf("kind %d has an empty class key", k)
		}
		if k.String() == "invalid-unsupported-kind" {
			t.Errorf("kind %d String() fell through to the invalid sentinel", k)
		}
	}
}

func TestKindNameKeysAreUnique(t *testing.T) {
	// Distinct kinds must carry distinct class keys — a duplicate key would
	// conflate two producers into one inventory family.
	byName := map[string]UnsupportedKind{}
	for _, k := range AllUnsupportedKinds() {
		name := kindName[k]
		if prior, dup := byName[name]; dup {
			t.Errorf("class key %q is shared by kinds %d and %d", name, prior, k)
		}
		byName[name] = k
	}
}
