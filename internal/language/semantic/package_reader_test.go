package semantic

import (
	"reflect"
	"testing"

	"github.com/tsoniclang/gotots/internal/identity"
)

func TestPackageReaderReusesOneConstantProjectionCursor(t *testing.T) {
	pkg := semanticWirePackage(t)
	reader := pkg.Reader()
	var occurrence identity.OccurrenceID
	var operation identity.OperationID
	if err := pkg.VisitResolutions(func(
		record OccurrenceResolution,
	) error {
		if occurrence.IsZero() {
			occurrence = record.Occurrence()
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := pkg.VisitOperations(func(record Operation) error {
		if operation.IsZero() {
			operation = record.ID()
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if occurrence.IsZero() || operation.IsZero() {
		t.Fatal("reader fixture lacks resolution or operation")
	}
	resolution, present := reader.Resolution(occurrence)
	if !present || resolution.Occurrence() != occurrence {
		t.Fatal("reader did not resolve canonical occurrence")
	}
	gotOperation, present := reader.Operation(operation)
	if !present || gotOperation.ID() != operation {
		t.Fatal("reader did not resolve canonical operation")
	}
	if reader.identities == nil {
		t.Fatal("reader lacks its one projection cursor")
	}
	projection := reader.identities
	if _, present := reader.Resolution(identity.OccurrenceID{}); present {
		t.Fatal("reader accepted zero occurrence identity")
	}
	if reader.identities != projection {
		t.Fatal("reader replaced its projection cursor")
	}
	readerType := reflect.TypeFor[PackageReader]()
	for index := 0; index < readerType.NumField(); index++ {
		kind := readerType.Field(index).Type.Kind()
		if kind == reflect.Map || kind == reflect.Slice {
			t.Fatalf(
				"package reader retains record collection %s",
				readerType.Field(index).Name,
			)
		}
	}
}
