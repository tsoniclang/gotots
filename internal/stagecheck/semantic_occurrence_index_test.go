package stagecheck

import "testing"

func TestSemanticOccurrenceIndexPreservesExactKeysAcrossGrowth(
	t *testing.T,
) {
	index := newSemanticOccurrenceIndex(1)
	keys := make([]semanticOccurrenceKey, 0, 128)
	for ordinal := 0; ordinal < 128; ordinal++ {
		key := semanticOccurrenceKey{
			file:  uint32(ordinal%3 + 1),
			start: ordinal * 17,
			end:   ordinal*17 + ordinal%5,
			kind:  uint16(ordinal%57 + 1),
		}
		keys = append(keys, key)
		reference := semanticOccurrenceRef(len(keys))
		if err := index.insert(keys, reference); err != nil {
			t.Fatal(err)
		}
		if got := index.reference(keys, key); got != reference {
			t.Fatalf(
				"semantic occurrence key %d resolved to %d, want %d",
				ordinal, got, reference,
			)
		}
	}
	for ordinal, key := range keys {
		want := semanticOccurrenceRef(ordinal + 1)
		if got := index.reference(keys, key); got != want {
			t.Fatalf(
				"semantic occurrence key %d resolved to %d, want %d after growth",
				ordinal, got, want,
			)
		}
	}
}

func TestSemanticOccurrenceIndexRejectsDuplicateKey(
	t *testing.T,
) {
	index := newSemanticOccurrenceIndex(1)
	key := semanticOccurrenceKey{
		file: 1, start: 10, end: 20, kind: 3,
	}
	keys := []semanticOccurrenceKey{key}
	if err := index.insert(keys, 1); err != nil {
		t.Fatal(err)
	}
	keys = append(keys, key)
	if err := index.insert(keys, 2); err == nil {
		t.Fatal("semantic occurrence index admitted a duplicate key")
	}
}
