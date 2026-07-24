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
	canonical := reflect.TypeFor[transientCanonicalOccurrences]()
	for _, removed := range []string{"key", "keys", "positions", "records"} {
		if _, present := canonical.FieldByName(removed); present {
			t.Fatalf(
				"canonical transient binding repeats occurrence payload field %s",
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
	builder, err := NewOccurrenceStoreBuilder(
		occurrence.ID().Span().File(),
		1,
	)
	if err != nil {
		t.Fatal(err)
	}
	occurrenceIndex, err := builder.Append(occurrence)
	if err != nil {
		t.Fatal(err)
	}
	store, err := builder.Seal()
	if err != nil {
		t.Fatal(err)
	}
	if err := index.bindStructuralStore(store, []ast.Node{original}); err != nil {
		t.Fatal(err)
	}
	reference, err := store.Reference(occurrenceIndex)
	if err != nil {
		t.Fatal(err)
	}
	if err := index.BindExecutableOccurrence(
		reference,
		uncertified,
	); err == nil {
		t.Fatal("uncertified executable node replaced structural truth")
	}
	index.counterparts[original] = checked
	index.originals[checked] = original
	if err := index.BindExecutableOccurrence(
		reference,
		checked,
	); err != nil {
		t.Fatal(err)
	}
	node, present := index.OccurrenceNode(occurrence.ID())
	if !present || node != checked {
		t.Fatal("certified checked counterpart was not selected")
	}
	if err := index.SealForStage2(); err != nil {
		t.Fatal(err)
	}
	if !index.Stage2Ready() {
		t.Fatal("Stage-2 seal retained construction reverse state")
	}
	if err := index.BindExecutableOccurrence(
		reference, checked,
	); err == nil {
		t.Fatal("sealed occurrence index accepted a construction binding")
	}
}

func TestExecutableStorePromotesSupplementWithoutDuplicateIdentity(
	t *testing.T,
) {
	index := &TransientIndex{
		occurrences:  newTransientOccurrenceStore(),
		counterparts: map[ast.Node]ast.Node{},
		originals:    map[ast.Node]ast.Node{},
	}
	occurrence := testTransientOccurrence(t)
	node := ast.NewIdent("value")
	if err := index.occurrences.bindSupplement(
		occurrence.ID(),
		node,
	); err != nil {
		t.Fatal(err)
	}
	builder, err := NewOccurrenceStoreBuilder(
		occurrence.ID().Span().File(),
		1,
	)
	if err != nil {
		t.Fatal(err)
	}
	occurrenceIndex, err := builder.Append(occurrence)
	if err != nil {
		t.Fatal(err)
	}
	if err := index.BindPendingExecutableOccurrence(
		builder,
		occurrenceIndex,
		node,
	); err != nil {
		t.Fatal(err)
	}
	store, err := builder.Seal()
	if err != nil {
		t.Fatal(err)
	}
	if err := index.BindExecutableOccurrenceStore(
		builder,
		store,
	); err != nil {
		t.Fatal(err)
	}
	if err := index.SealForStage2(); err != nil {
		t.Fatal(err)
	}
	fileID := occurrence.ID().Span().File()
	fileReference := index.occurrences.filesByID[fileID]
	file := index.occurrences.files[fileReference-1]
	if len(file.supplements) != 0 ||
		file.canonical[transientOccurrenceExecutable].store != store {
		t.Fatal("supplemental identity was not transferred to canonical storage")
	}
	count, err := index.OccurrenceNodeCountForFiles(
		[]identity.FileID{occurrence.ID().Span().File()},
	)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("promoted occurrence counted %d times", count)
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
