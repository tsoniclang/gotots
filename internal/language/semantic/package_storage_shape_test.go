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
			"normalized cores are not smaller than detached public values: cores=%v public=%d/%d/%d",
			sizes,
			unsafe.Sizeof(DefinitionSemantics{}),
			unsafe.Sizeof(Type{}),
			unsafe.Sizeof(Operation{}),
		)
	}
}

var operationProjectionSink identity.OccurrenceID

func TestOperationVisitorDoesNotCloneRelationArenas(t *testing.T) {
	empty := semanticWirePackage(t)
	large := semanticWirePackage(t)
	record := &large.operations.records[0]
	reference := record.idOccurrence(large.identities)
	const count = 4096
	large.operations.operands = make([]occurrenceRef, count)
	for index := range large.operations.operands {
		large.operations.operands[index] = reference
	}
	record.operands = occurrenceRefRange{
		start: 0,
		count: count,
	}
	large.operationView = newPackageOperationProjection(
		large.operations,
		large.identities,
	)
	allocations := func(pkg Package) float64 {
		return testing.AllocsPerRun(10, func() {
			err := pkg.VisitOperations(func(
				operation Operation,
			) error {
				for index := 0; index <
					operation.OperandCount(); index++ {
					operationProjectionSink, _ =
						operation.Operand(index)
				}
				return nil
			})
			if err != nil {
				panic(err)
			}
		})
	}
	emptyAllocations := allocations(empty)
	largeAllocations := allocations(large)
	if largeAllocations != emptyAllocations {
		t.Fatalf(
			"operation relation traversal allocations grow with relation size: empty=%.0f large=%.0f",
			emptyAllocations,
			largeAllocations,
		)
	}
}

func TestBinaryStoredRecordsContainReferencesRatherThanSemanticIdentities(
	t *testing.T,
) {
	roots := []reflect.Type{
		reflect.TypeOf(storedDefinition{}),
		reflect.TypeOf(storedResolution{}),
		reflect.TypeOf(storedDeclaration{}),
		reflect.TypeOf(storedBinding{}),
		reflect.TypeOf(storedType{}),
		reflect.TypeOf(storedOperation{}),
		reflect.TypeOf(storedUnsupported{}),
	}
	for _, root := range roots {
		if path, forbidden := forbiddenWireCarrier(
			root,
			map[reflect.Type]bool{},
		); forbidden {
			t.Errorf(
				"binary stored record %s retains semantic carrier at %s",
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
	switch current {
	case reflect.TypeFor[identity.ModuleID](),
		reflect.TypeFor[identity.Owner](),
		reflect.TypeFor[identity.PackageID](),
		reflect.TypeFor[identity.FileID](),
		reflect.TypeFor[identity.SpanID](),
		reflect.TypeFor[identity.OccurrenceID](),
		reflect.TypeFor[identity.DefinitionID](),
		reflect.TypeFor[identity.SemanticTypeID](),
		reflect.TypeFor[identity.SemanticDeclarationID](),
		reflect.TypeFor[identity.SemanticBindingID](),
		reflect.TypeFor[identity.OperationID](),
		reflect.TypeFor[identity.UnsupportedID]():
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
