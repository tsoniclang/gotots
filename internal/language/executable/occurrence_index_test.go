package executable

import "testing"

func TestOccurrenceIdentityIndexPreservesExactKeysAcrossGrowth(
	t *testing.T,
) {
	store := newOccurrenceStore()
	for ordinal := 0; ordinal < 128; ordinal++ {
		key := occurrenceKey{
			file:  uint32(ordinal%3 + 1),
			start: ordinal * 17,
			end:   ordinal*17 + ordinal%5,
			kind:  uint16(ordinal%57 + 1),
		}
		store.appendIdentityRecord(key)
		reference := occurrenceRef(store.identityCount)
		if err := store.byIdentity.insert(store, reference); err != nil {
			t.Fatal(err)
		}
		if got := store.byIdentity.reference(store, key); got != reference {
			t.Fatalf(
				"executable occurrence key %d resolved to %d, want %d",
				ordinal,
				got,
				reference,
			)
		}
	}
	for reference := occurrenceRef(1); uint32(reference) <= store.identityCount; reference++ {
		record := store.identityRecord(reference)
		key := occurrenceKey{
			file:  record.file,
			start: record.start,
			end:   record.end,
			kind:  record.kind,
		}
		if got := store.byIdentity.reference(store, key); got != reference {
			t.Fatalf(
				"executable occurrence key resolved to %d, want %d after growth",
				got,
				reference,
			)
		}
	}
}

func TestOccurrenceIdentityIndexRejectsDuplicateKey(
	t *testing.T,
) {
	store := newOccurrenceStore()
	key := occurrenceKey{file: 1, start: 10, end: 20, kind: 3}
	store.appendIdentityRecord(key)
	if err := store.byIdentity.insert(store, 1); err != nil {
		t.Fatal(err)
	}
	store.appendIdentityRecord(key)
	if err := store.byIdentity.insert(store, 2); err == nil {
		t.Fatal("executable occurrence index admitted a duplicate key")
	}
}
