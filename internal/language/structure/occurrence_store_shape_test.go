package structure

import (
	"reflect"
	"testing"
	"unsafe"
)

func TestOccurrenceStoreRecordKeepsOnlyActiveDisplayPayload(t *testing.T) {
	if size := unsafe.Sizeof(occurrenceStoreRecord{}); size > 96 {
		t.Fatalf(
			"occurrence store record is %d bytes; want at most 96",
			size,
		)
	}
	record := reflect.TypeFor[occurrenceStoreRecord]()
	for _, field := range []string{
		"displayStartFile",
		"displayStartLine",
		"displayStartColumn",
		"displayEndFile",
		"displayEndLine",
		"displayEndColumn",
	} {
		if _, present := record.FieldByName(field); present {
			t.Fatalf(
				"occurrence record retains inactive display field %s",
				field,
			)
		}
	}
	if _, present := record.FieldByName("display"); !present {
		t.Fatal("occurrence record lacks sparse display reference")
	}
}

func TestOccurrenceStoreRoundTripsSparseDisplayOverrides(t *testing.T) {
	base := testTransientOccurrence(t)
	adjusted := testStoreOccurrence(t, base, 10, 15)
	display := DisplaySpan{
		Start: DisplayPosition{
			Filename: "generated.go",
			Line:     100,
			Column:   3,
		},
		End: DisplayPosition{
			Filename: "generated.go",
			Line:     100,
			Column:   8,
		},
	}
	adjusted, err := NewOccurrence(
		adjusted.ID(),
		adjusted.Kind(),
		adjusted.Parent(),
		adjusted.Edge(),
		adjusted.Ordinal(),
		adjusted.Span(),
		display,
		adjusted.Token(),
	)
	if err != nil {
		t.Fatal(err)
	}
	builder, err := NewOccurrenceStoreBuilder(
		base.ID().Span().File(),
		2,
	)
	if err != nil {
		t.Fatal(err)
	}
	baseIndex, err := builder.Append(base)
	if err != nil {
		t.Fatal(err)
	}
	adjustedIndex, err := builder.Append(adjusted)
	if err != nil {
		t.Fatal(err)
	}
	store, err := builder.Seal()
	if err != nil {
		t.Fatal(err)
	}
	if len(store.displaySpans) != 1 {
		t.Fatalf(
			"display override count = %d, want 1",
			len(store.displaySpans),
		)
	}
	baseReference, err := store.Reference(baseIndex)
	if err != nil {
		t.Fatal(err)
	}
	if baseReference.Display() != base.Display() {
		t.Fatal("ordinary display span did not derive from physical evidence")
	}
	adjustedReference, err := store.Reference(adjustedIndex)
	if err != nil {
		t.Fatal(err)
	}
	if adjustedReference.Display() != display {
		t.Fatalf(
			"adjusted display = %+v, want %+v",
			adjustedReference.Display(),
			display,
		)
	}
}
