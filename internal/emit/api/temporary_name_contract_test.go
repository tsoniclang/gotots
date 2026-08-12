package api

import "testing"

func TestTemporaryNameCatalogIsTotalAndCollisionFree(t *testing.T) {
	seen := make(map[string]TemporaryKind)
	for kind := TemporaryAssignmentValue; kind < temporaryKindLimit; kind++ {
		prefix, err := TemporaryPrefix(kind)
		if err != nil {
			t.Fatalf("temporary kind %d has no prefix: %v", kind, err)
		}
		candidate := prefix + "0"
		if previous, duplicate := seen[candidate]; duplicate {
			t.Fatalf(
				"temporary kinds %d and %d produce the same binding %q",
				previous,
				kind,
				candidate,
			)
		}
		seen[candidate] = kind
	}
	if len(seen) != int(temporaryKindLimit-TemporaryAssignmentValue) {
		t.Fatalf("temporary catalog contains %d names, want %d", len(seen), temporaryKindLimit-TemporaryAssignmentValue)
	}
	for _, invalid := range []TemporaryKind{TemporaryInvalid, temporaryKindLimit} {
		if _, err := TemporaryPrefix(invalid); err == nil {
			t.Fatalf("invalid temporary kind %d was accepted", invalid)
		}
	}
}
