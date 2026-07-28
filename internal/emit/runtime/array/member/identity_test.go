package arraymember

import "testing"

func TestCatalogIsClosedAndPinned(t *testing.T) {
	expected := []struct {
		identity Identity
		name     string
	}{
		{Zero, "zero"},
		{Literal, "literal"},
		{Copy, "copy"},
		{Get, "get"},
		{Set, "set"},
		{Length, "length"},
	}
	actual := All()
	if len(actual) != len(expected) {
		t.Fatalf("array runtime members = %d, want %d", len(actual), len(expected))
	}
	for index, want := range expected {
		if actual[index] != want.identity {
			t.Errorf(
				"array runtime member %d = %d, want %d",
				index,
				actual[index],
				want.identity,
			)
		}
		if actual[index].Name() != want.name {
			t.Errorf(
				"array runtime member %d name = %q, want %q",
				actual[index],
				actual[index].Name(),
				want.name,
			)
		}
	}
}

func TestInvalidIdentityCannotProduceAName(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("invalid array runtime member produced a name")
		}
	}()
	_ = Identity(255).Name()
}
