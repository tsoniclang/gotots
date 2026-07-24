package semantic

import (
	"reflect"
	"strings"
	"testing"
	"unsafe"

	"github.com/tsoniclang/gotots/internal/identity"
)

func TestNormalizedRecordCoresStayIndependentOfInactivePayloads(
	t *testing.T,
) {
	sizes := map[string]uintptr{
		"definition-core": unsafe.Sizeof(storedDefinition{}),
		"type-core":       unsafe.Sizeof(storedType{}),
		"operation-core":  unsafe.Sizeof(storedOperation{}),
	}
	limits := map[string]uintptr{
		"definition-core": 80,
		"type-core":       24,
		"operation-core":  192,
	}
	for name, size := range sizes {
		if size > limits[name] {
			t.Fatalf(
				"%s is %d bytes, exceeds compact-core limit %d",
				name,
				size,
				limits[name],
			)
		}
	}
	if sizes["definition-core"] >=
		unsafe.Sizeof(DefinitionSemantics{}) ||
		sizes["type-core"] >= unsafe.Sizeof(Type{}) ||
		sizes["operation-core"] >= unsafe.Sizeof(Operation{}) {
		t.Fatalf(
			"normalized cores are not smaller than public projections: cores=%v public=%d/%d/%d",
			sizes,
			unsafe.Sizeof(DefinitionSemantics{}),
			unsafe.Sizeof(Type{}),
			unsafe.Sizeof(Operation{}),
		)
	}
}

func TestWireRecordsContainReferencesRatherThanSemanticIdentities(
	t *testing.T,
) {
	roots := []reflect.Type{
		reflect.TypeOf(wireDefinitionRecord{}),
		reflect.TypeOf(wireResolutionRecord{}),
		reflect.TypeOf(wireDeclarationRecord{}),
		reflect.TypeOf(wireBindingRecord{}),
		reflect.TypeOf(wireTypeRecord{}),
		reflect.TypeOf(wireOperationRecord{}),
		reflect.TypeOf(wireUnsupportedRecord{}),
	}
	for _, root := range roots {
		if path, forbidden := forbiddenWireCarrier(
			root,
			map[reflect.Type]bool{},
		); forbidden {
			t.Errorf(
				"wire record %s retains semantic carrier at %s",
				root,
				path,
			)
		}
	}
	type invalidWireRecord struct {
		Definition identity.DefinitionID
	}
	if _, forbidden := forbiddenWireCarrier(
		reflect.TypeOf(invalidWireRecord{}),
		map[reflect.Type]bool{},
	); !forbidden {
		t.Fatal(
			"wire-carrier gate did not detect a full semantic identity",
		)
	}
}

func forbiddenWireCarrier(
	current reflect.Type,
	seen map[reflect.Type]bool,
) (string, bool) {
	for current.Kind() == reflect.Pointer {
		current = current.Elem()
	}
	if current.PkgPath() ==
		"github.com/tsoniclang/gotots/internal/identity" {
		return current.String(), true
	}
	if current.PkgPath() ==
		"github.com/tsoniclang/gotots/internal/language/semantic" &&
		!strings.HasPrefix(current.Name(), "wire") {
		switch current {
		case reflect.TypeOf(DefinitionSemantics{}),
			reflect.TypeOf(OccurrenceResolution{}),
			reflect.TypeOf(Declaration{}),
			reflect.TypeOf(Binding{}),
			reflect.TypeOf(Type{}),
			reflect.TypeOf(Operation{}),
			reflect.TypeOf(Unsupported{}):
			return current.String(), true
		}
	}
	if seen[current] {
		return "", false
	}
	seen[current] = true
	switch current.Kind() {
	case reflect.Array, reflect.Slice:
		path, forbidden := forbiddenWireCarrier(
			current.Elem(),
			seen,
		)
		if forbidden {
			return current.String() + " -> " + path, true
		}
	case reflect.Struct:
		for index := 0; index < current.NumField(); index++ {
			field := current.Field(index)
			path, forbidden := forbiddenWireCarrier(
				field.Type,
				seen,
			)
			if forbidden {
				return current.String() + "." +
					field.Name + " -> " + path, true
			}
		}
	}
	return "", false
}
