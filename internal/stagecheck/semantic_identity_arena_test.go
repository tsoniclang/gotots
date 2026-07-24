package stagecheck

import (
	"reflect"
	"testing"

	"github.com/tsoniclang/gotots/internal/identity"
)

func TestSemanticVerifierRelationsUseDenseLocalIdentityArenas(
	t *testing.T,
) {
	assertSemanticFieldType(
		t, reflect.TypeFor[semanticOccurrenceStore](), "records",
		reflect.TypeFor[[]semanticExpectedOccurrence](),
	)
	if cap(newSemanticOccurrenceStore(19).records) != 19 {
		t.Fatal(
			"semantic-verifier occurrence arena ignored its exact capacity",
		)
	}
	for _, name := range []string{"owner", "structuralOwner"} {
		assertSemanticFieldType(
			t, reflect.TypeFor[semanticExpectedOccurrence](), name,
			reflect.TypeFor[semanticDefinitionRef](),
		)
	}
	assertSemanticFieldType(
		t, reflect.TypeFor[semanticPackageExpectation](), "order",
		reflect.TypeFor[[]semanticOccurrenceRef](),
	)
	assertSemanticFieldType(
		t, reflect.TypeFor[checkerSemanticVerifier](), "children",
		reflect.TypeFor[[][]semanticOccurrenceRef](),
	)
	assertSemanticFieldType(
		t, reflect.TypeFor[checkerSemanticVerifier](),
		"compileTimeAnchor",
		reflect.TypeFor[[]semanticOccurrenceRef](),
	)

	type fullIdentityRelations struct {
		Owner    identity.DefinitionID
		Children []identity.OccurrenceID
	}
	if !hasVerifierFullIdentityRelation(
		reflect.TypeFor[fullIdentityRelations](),
	) {
		t.Fatal(
			"semantic-verifier identity gate missed its full-identity control",
		)
	}
	if hasVerifierFullIdentityRelation(
		reflect.TypeFor[semanticExpectedOccurrence](),
	) {
		t.Fatal(
			"semantic expected occurrence retains a full identity relation",
		)
	}
}

func assertSemanticFieldType(
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

func hasVerifierFullIdentityRelation(record reflect.Type) bool {
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
