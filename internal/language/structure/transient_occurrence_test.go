package structure

import (
	"go/ast"
	"reflect"
	"testing"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
)

func TestTransientOccurrenceJoinHasOneBidirectionalOwner(t *testing.T) {
	typ := reflect.TypeFor[TransientIndex]()
	for _, removed := range []string{
		"structural",
		"executable",
		"structuralID",
		"executableID",
	} {
		if _, present := typ.FieldByName(removed); present {
			t.Fatalf("transient index retains duplicate map %s", removed)
		}
	}
	for _, required := range []string{"occurrences"} {
		if _, present := typ.FieldByName(required); !present {
			t.Fatalf("transient index lacks canonical map %s", required)
		}
	}
	for _, removed := range []string{"nodes", "ids", "constructionIDs"} {
		if _, present := typ.FieldByName(removed); present {
			t.Fatalf(
				"transient index retains denormalized occurrence field %s",
				removed,
			)
		}
	}
}

func TestExecutableOccurrenceAcceptsOnlyCertifiedNodeSubstitution(
	t *testing.T,
) {
	index := &TransientIndex{
		occurrences:  newTransientOccurrenceStore(),
		counterparts: map[ast.Node]ast.Node{},
		originals:    map[ast.Node]ast.Node{},
	}
	original := ast.NewIdent("value")
	checked := ast.NewIdent("value")
	uncertified := ast.NewIdent("value")
	occurrence := testTransientOccurrence(t)
	if err := index.bindStructuralOccurrence(
		occurrence,
		original,
	); err != nil {
		t.Fatal(err)
	}
	if err := index.BindExecutableOccurrence(
		occurrence.ID(),
		uncertified,
	); err == nil {
		t.Fatal("uncertified executable node replaced structural truth")
	}
	index.counterparts[original] = checked
	index.originals[checked] = original
	if err := index.BindExecutableOccurrence(
		occurrence.ID(),
		checked,
	); err != nil {
		t.Fatal(err)
	}
	node, present := index.OccurrenceNode(occurrence.ID())
	if !present || node != checked {
		t.Fatal("certified checked counterpart was not selected")
	}
	for _, candidate := range []ast.Node{original, checked} {
		address, found := index.occurrences.reverse[candidate]
		id, err := index.occurrences.identity(address)
		if err != nil {
			t.Fatal(err)
		}
		if !found || id != occurrence.ID() {
			t.Fatalf("construction join lost canonical identity %s", id)
		}
	}
	if err := index.SealForStage2(); err != nil {
		t.Fatal(err)
	}
	if !index.Stage2Ready() || index.occurrences.reverse != nil {
		t.Fatal("Stage-2 seal retained construction reverse state")
	}
	if err := index.BindExecutableOccurrence(
		occurrence.ID(), checked,
	); err == nil {
		t.Fatal("sealed occurrence index accepted a construction binding")
	}
}

func testTransientOccurrence(t *testing.T) Occurrence {
	t.Helper()
	module, err := identity.NewModuleID("example.com/transient", "")
	if err != nil {
		t.Fatal(err)
	}
	owner, err := identity.NewModuleOwner(module)
	if err != nil {
		t.Fatal(err)
	}
	file, err := identity.NewFileID(owner, "transient.go")
	if err != nil {
		t.Fatal(err)
	}
	span, err := identity.NewSpanID(file, 0, 5)
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
			Start: Position{Line: 1, Column: 1, Offset: 0},
			End:   Position{Line: 1, Column: 6, Offset: 5},
		},
		DisplaySpan{
			Start: DisplayPosition{
				Filename: file.String(),
				Line:     1,
				Column:   1,
			},
			End: DisplayPosition{
				Filename: file.String(),
				Line:     1,
				Column:   6,
			},
		},
		catalog.TokenInvalid,
	)
	if err != nil {
		t.Fatal(err)
	}
	return occurrence
}
