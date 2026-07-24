package structure

import (
	"testing"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
)

func TestOccurrenceStoreOwnsExactLocalIdentityIndex(t *testing.T) {
	base := testTransientOccurrence(t)
	builder, err := NewOccurrenceStoreBuilder(
		base.ID().Span().File(),
		1,
	)
	if err != nil {
		t.Fatal(err)
	}
	const count = 257
	identities := make([]identity.OccurrenceID, 0, count)
	for index := 0; index < count; index++ {
		occurrence := testStoreOccurrence(t, base, index*3, index*3+2)
		if _, err := builder.Append(occurrence); err != nil {
			t.Fatal(err)
		}
		identities = append(identities, occurrence.ID())
	}
	store, err := builder.Seal()
	if err != nil {
		t.Fatal(err)
	}
	for index, id := range identities {
		reference, present := store.reference(id)
		if !present || reference.ID() != id ||
			reference.index != OccurrenceIndex(index+1) {
			t.Fatalf("identity %s resolved to %d, present=%t", id, reference.index, present)
		}
	}
	missing := testStoreOccurrence(t, base, count*3, count*3+2)
	if _, present := store.reference(missing.ID()); present {
		t.Fatal("local identity index admitted an absent occurrence")
	}
}

func TestOccurrenceStoreRejectsDuplicateIdentityAtSeal(t *testing.T) {
	occurrence := testTransientOccurrence(t)
	builder, err := NewOccurrenceStoreBuilder(
		occurrence.ID().Span().File(),
		2,
	)
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if _, err := builder.Append(occurrence); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := builder.Seal(); err == nil {
		t.Fatal("occurrence store accepted duplicate canonical identity")
	}
}

func testStoreOccurrence(
	t *testing.T,
	base Occurrence,
	start int,
	end int,
) Occurrence {
	t.Helper()
	span, err := identity.NewSpanID(base.ID().Span().File(), start, end)
	if err != nil {
		t.Fatal(err)
	}
	id, err := identity.NewOccurrenceID(
		span,
		uint16(catalog.KindIdent),
	)
	if err != nil {
		t.Fatal(err)
	}
	occurrence, err := NewOccurrence(
		id,
		catalog.KindIdent,
		identity.OccurrenceID{},
		catalog.EdgeInvalid,
		0,
		Span{
			Start: Position{Line: 1, Column: start + 1, Offset: start},
			End:   Position{Line: 1, Column: end + 1, Offset: end},
		},
		DisplaySpan{
			Start: DisplayPosition{
				Filename: base.Display().Start.Filename,
				Line:     1,
				Column:   start + 1,
			},
			End: DisplayPosition{
				Filename: base.Display().End.Filename,
				Line:     1,
				Column:   end + 1,
			},
		},
		catalog.TokenInvalid,
	)
	if err != nil {
		t.Fatal(err)
	}
	return occurrence
}
