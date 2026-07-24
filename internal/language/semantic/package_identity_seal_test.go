package semantic

import (
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/identity"
)

func TestIdentityAdmissionRejectsInvalidComponents(t *testing.T) {
	pkg := semanticWirePackage(t)
	table := pkg.identities
	table.operations = append(
		[]storedOperationIdentity(nil),
		table.operations...,
	)
	table.operations[0].definition =
		definitionRef(len(table.definitions) + 1)
	if _, err := admitPackageIdentityTable(table); err == nil ||
		!strings.Contains(err.Error(), "component references are invalid") {
		t.Fatalf(
			"identity admission error = %v, want invalid components",
			err,
		)
	}
}

func TestNormalizedStoresRequireAdmittedIdentityTable(t *testing.T) {
	field, found := reflect.TypeFor[normalizedPackageStores]().
		FieldByName("identities")
	if !found {
		t.Fatal("normalized package stores omit identities")
	}
	if field.Type != reflect.TypeFor[admittedPackageIdentityTable]() {
		t.Fatalf(
			"normalized identity storage type = %s, want admitted table",
			field.Type,
		)
	}
}

func TestCanonicalizeComponentsHandlesEveryPermutationCycle(
	t *testing.T,
) {
	tests := []struct {
		name   string
		values []int
		want   []int
		remap  []uint64
	}{
		{
			name:   "already-canonical",
			values: []int{1, 2, 3, 4},
			want:   []int{1, 2, 3, 4},
			remap:  []uint64{0, 1, 2, 3, 4},
		},
		{
			name:   "single-cycle",
			values: []int{2, 3, 1},
			want:   []int{1, 2, 3},
			remap:  []uint64{0, 2, 3, 1},
		},
		{
			name:   "two-cycles",
			values: []int{2, 1, 4, 3},
			want:   []int{1, 2, 3, 4},
			remap:  []uint64{0, 2, 1, 4, 3},
		},
		{
			name:   "reverse",
			values: []int{5, 4, 3, 2, 1},
			want:   []int{1, 2, 3, 4, 5},
			remap:  []uint64{0, 5, 4, 3, 2, 1},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			values := append([]int(nil), test.values...)
			got, remap := canonicalizeComponents(
				values,
				func(left int, right int) bool {
					return left < right
				},
			)
			if !reflect.DeepEqual(got, test.want) ||
				!reflect.DeepEqual(remap, test.remap) {
				t.Fatalf(
					"canonicalize(%v) = %v/%v, want %v/%v",
					test.values,
					got,
					remap,
					test.want,
					test.remap,
				)
			}
		})
	}
}

func TestSealedIdentityProjectionIsAllocationFree(t *testing.T) {
	pkg := semanticWirePackage(t)
	var occurrence identity.OccurrenceID
	var definition identity.DefinitionID
	var semanticType identity.SemanticTypeID
	var operation identity.OperationID
	var occurrenceReference occurrenceRef
	var definitionReference definitionRef
	var typeReference typeRef
	var operationReference operationRef
	allocations := testing.AllocsPerRun(1000, func() {
		occurrence = pkg.identities.occurrence(1)
		definition = pkg.identities.definition(1)
		semanticType = pkg.identities.typeID(1)
		operation = pkg.identities.operation(1)
		occurrenceReference =
			pkg.identities.occurrenceReference(occurrence)
		definitionReference =
			pkg.identities.definitionReference(definition)
		typeReference =
			pkg.identities.typeReference(semanticType)
		operationReference =
			pkg.identities.operationReference(operation)
	})
	runtime.KeepAlive(occurrence)
	runtime.KeepAlive(definition)
	runtime.KeepAlive(semanticType)
	runtime.KeepAlive(operation)
	if allocations != 0 {
		t.Fatalf(
			"sealed identity projection allocates %.2f times per visit",
			allocations,
		)
	}
	if occurrence.IsZero() ||
		definition.IsZero() ||
		semanticType.IsZero() ||
		operation.IsZero() ||
		occurrenceReference == 0 ||
		definitionReference == 0 ||
		typeReference == 0 ||
		operationReference == 0 {
		t.Fatal("sealed identity projection returned a zero identity")
	}
}

func TestSealedIdentityTableRetainsOnlyCompactComponents(t *testing.T) {
	fullIdentities := map[reflect.Type]bool{
		reflect.TypeFor[identity.ModuleID]():              true,
		reflect.TypeFor[identity.Owner]():                 true,
		reflect.TypeFor[identity.PackageID]():             true,
		reflect.TypeFor[identity.FileID]():                true,
		reflect.TypeFor[identity.SpanID]():                true,
		reflect.TypeFor[identity.OccurrenceID]():          true,
		reflect.TypeFor[identity.DefinitionID]():          true,
		reflect.TypeFor[identity.SemanticTypeID]():        true,
		reflect.TypeFor[identity.SemanticDeclarationID](): true,
		reflect.TypeFor[identity.SemanticBindingID]():     true,
		reflect.TypeFor[identity.OperationID]():           true,
		reflect.TypeFor[identity.UnsupportedID]():         true,
	}
	if path, found := retainedFullIdentity(
		reflect.TypeFor[packageIdentityTable](),
		fullIdentities,
		map[reflect.Type]bool{},
	); found {
		t.Fatalf(
			"normalized identity table retains full identity at %s",
			path,
		)
	}
	type invalidIdentityTable struct {
		Operations []identity.OperationID
	}
	if _, found := retainedFullIdentity(
		reflect.TypeFor[invalidIdentityTable](),
		fullIdentities,
		map[reflect.Type]bool{},
	); !found {
		t.Fatal("full-identity retention control was not detected")
	}
}

func TestIdentityBuilderRetainsNoFullIdentityCollections(t *testing.T) {
	fullIdentities := map[reflect.Type]bool{
		reflect.TypeFor[identity.ModuleID]():              true,
		reflect.TypeFor[identity.Owner]():                 true,
		reflect.TypeFor[identity.PackageID]():             true,
		reflect.TypeFor[identity.FileID]():                true,
		reflect.TypeFor[identity.SpanID]():                true,
		reflect.TypeFor[identity.OccurrenceID]():          true,
		reflect.TypeFor[identity.DefinitionID]():          true,
		reflect.TypeFor[identity.SemanticTypeID]():        true,
		reflect.TypeFor[identity.SemanticDeclarationID](): true,
		reflect.TypeFor[identity.SemanticBindingID]():     true,
		reflect.TypeFor[identity.OperationID]():           true,
		reflect.TypeFor[identity.UnsupportedID]():         true,
	}
	roots := []reflect.Type{
		reflect.TypeFor[packageIdentityBuilder](),
		reflect.TypeFor[binarySemanticShard](),
	}
	for _, root := range roots {
		if path, found := retainedFullIdentityCollection(
			root,
			fullIdentities,
			map[reflect.Type]bool{},
			false,
		); found {
			t.Errorf(
				"%s retains a full-identity collection at %s",
				root,
				path,
			)
		}
	}
	type invalidIdentityBuilder struct {
		Definitions map[identity.DefinitionID]uint64
	}
	if _, found := retainedFullIdentityCollection(
		reflect.TypeFor[invalidIdentityBuilder](),
		fullIdentities,
		map[reflect.Type]bool{},
		false,
	); !found {
		t.Fatal("full-identity builder-map control was not detected")
	}
}

func TestIdentityProjectionCursorHasConstantShape(t *testing.T) {
	typ := reflect.TypeFor[packageIdentityProjection]()
	for index := 0; index < typ.NumField(); index++ {
		field := typ.Field(index)
		if field.Name == "table" {
			continue
		}
		if path, found := retainedCollection(
			field.Type,
			map[reflect.Type]bool{},
		); found {
			t.Errorf(
				"identity projection cursor field %s retains collection at %s",
				field.Name,
				path,
			)
		}
	}
	type invalidProjectionCursor struct {
		Operations []identity.OperationID
	}
	invalid := reflect.TypeFor[invalidProjectionCursor]()
	if invalid.Field(0).Type.Kind() != reflect.Slice {
		t.Fatal("projection-cursor control was not detected")
	}
}

func retainedFullIdentity(
	current reflect.Type,
	full map[reflect.Type]bool,
	seen map[reflect.Type]bool,
) (string, bool) {
	for current.Kind() == reflect.Pointer {
		current = current.Elem()
	}
	if full[current] {
		return current.String(), true
	}
	if seen[current] {
		return "", false
	}
	seen[current] = true
	switch current.Kind() {
	case reflect.Array, reflect.Slice:
		if path, found := retainedFullIdentity(
			current.Elem(), full, seen,
		); found {
			return current.String() + " -> " + path, true
		}
	case reflect.Struct:
		for index := 0; index < current.NumField(); index++ {
			field := current.Field(index)
			if path, found := retainedFullIdentity(
				field.Type, full, seen,
			); found {
				return current.String() + "." +
					field.Name + " -> " + path, true
			}
		}
	}
	return "", false
}

func retainedCollection(
	current reflect.Type,
	seen map[reflect.Type]bool,
) (string, bool) {
	for current.Kind() == reflect.Pointer {
		current = current.Elem()
	}
	if seen[current] {
		return "", false
	}
	seen[current] = true
	switch current.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice:
		return current.String(), true
	case reflect.Struct:
		for index := 0; index < current.NumField(); index++ {
			field := current.Field(index)
			if path, found := retainedCollection(
				field.Type, seen,
			); found {
				return current.String() + "." +
					field.Name + " -> " + path, true
			}
		}
	}
	return "", false
}

func retainedFullIdentityCollection(
	current reflect.Type,
	full map[reflect.Type]bool,
	seen map[reflect.Type]bool,
	insideCollection bool,
) (string, bool) {
	for current.Kind() == reflect.Pointer {
		current = current.Elem()
	}
	if insideCollection && full[current] {
		return current.String(), true
	}
	if seen[current] {
		return "", false
	}
	seen[current] = true
	switch current.Kind() {
	case reflect.Array, reflect.Slice:
		if path, found := retainedFullIdentityCollection(
			current.Elem(), full, seen, true,
		); found {
			return current.String() + " -> " + path, true
		}
	case reflect.Map:
		if path, found := retainedFullIdentityCollection(
			current.Key(), full, seen, true,
		); found {
			return current.String() + " key -> " + path, true
		}
		if path, found := retainedFullIdentityCollection(
			current.Elem(), full, seen, true,
		); found {
			return current.String() + " value -> " + path, true
		}
	case reflect.Struct:
		for index := 0; index < current.NumField(); index++ {
			field := current.Field(index)
			if path, found := retainedFullIdentityCollection(
				field.Type, full, seen, insideCollection,
			); found {
				return current.String() + "." +
					field.Name + " -> " + path, true
			}
		}
	}
	return "", false
}
