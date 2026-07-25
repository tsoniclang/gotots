package frontend

import (
	"reflect"
	"testing"

	"github.com/tsoniclang/gotots/internal/identity"
)

func TestPackageRelationsUseDenseLocalIdentityArenas(t *testing.T) {
	assertFieldType(
		t, reflect.TypeFor[definitionStore](), "records",
		reflect.TypeFor[[]definitionInput](),
	)
	assertFieldType(
		t, reflect.TypeFor[occurrenceStore](), "records",
		reflect.TypeFor[[]occurrenceInput](),
	)
	if cap(newOccurrenceStore(19).records) != 19 {
		t.Fatal("package occurrence arena ignored its exact capacity")
	}
	if cap(newOccurrenceStore(19).keys) != 19 {
		t.Fatal("package occurrence key arena ignored its exact capacity")
	}
	storeType := reflect.TypeFor[occurrenceStore]()
	for index := 0; index < storeType.NumField(); index++ {
		field := storeType.Field(index)
		if field.Type.Kind() == reflect.Map &&
			field.Type.Key() ==
				reflect.TypeFor[localOccurrenceKey]() {
			t.Fatalf(
				"package occurrence arena retains full identity map %s",
				field.Name,
			)
		}
	}
	if cap(newDefinitionStore(7).records) != 7 {
		t.Fatal("package definition arena ignored its exact capacity")
	}
	assertFieldType(
		t, reflect.TypeFor[occurrenceInput](), "owner",
		reflect.TypeFor[packageDefinitionRef](),
	)
	assertFieldType(
		t, reflect.TypeFor[occurrenceStore](), "children",
		reflect.TypeFor[[]packageOccurrenceRef](),
	)
	assertFieldType(
		t, reflect.TypeFor[occurrenceStore](), "parents",
		reflect.TypeFor[[]packageOccurrenceRef](),
	)
	assertFieldType(
		t, reflect.TypeFor[occurrenceStore](), "childRanges",
		reflect.TypeFor[[]occurrenceRelationRange](),
	)
	if _, present := reflect.TypeFor[occurrenceInput]().
		FieldByName("children"); present {
		t.Fatal("occurrence record retains an unbounded child slice")
	}
	assertFieldType(
		t, reflect.TypeFor[packageInput](), "order",
		reflect.TypeFor[[]packageOccurrenceRef](),
	)
	for _, name := range []string{
		"breakTarget",
		"continueTarget",
		"fallthroughTarget",
	} {
		assertFieldType(
			t, reflect.TypeFor[occurrenceContext](), name,
			reflect.TypeFor[packageOccurrenceRef](),
		)
	}

	type fullIdentityRelations struct {
		Owner    identity.DefinitionID
		Children []identity.OccurrenceID
	}
	if !hasFullIdentityRelation(
		reflect.TypeFor[fullIdentityRelations](),
	) {
		t.Fatal(
			"package-local identity gate missed its full-identity control",
		)
	}
	for _, record := range []reflect.Type{
		reflect.TypeFor[occurrenceInput](),
		reflect.TypeFor[occurrenceContext](),
	} {
		if hasFullIdentityRelation(record) {
			t.Fatalf("%s retains a full identity relation", record)
		}
	}
}

func assertFieldType(
	t *testing.T,
	record reflect.Type,
	name string,
	want reflect.Type,
) {
	t.Helper()
	field, present := record.FieldByName(name)
	if !present {
		t.Fatalf("%s has no field %s", record, name)
	}
	if field.Type != want {
		t.Fatalf(
			"%s.%s has type %s, want %s",
			record, name, field.Type, want,
		)
	}
}

func hasFullIdentityRelation(record reflect.Type) bool {
	occurrence := reflect.TypeFor[identity.OccurrenceID]()
	definition := reflect.TypeFor[identity.DefinitionID]()
	for index := 0; index < record.NumField(); index++ {
		field := record.Field(index).Type
		if field == occurrence || field == definition {
			return true
		}
		if field.Kind() == reflect.Slice &&
			(field.Elem() == occurrence ||
				field.Elem() == definition) {
			return true
		}
	}
	return false
}
