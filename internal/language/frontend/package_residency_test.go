package frontend

import (
	"reflect"
	"testing"

	"github.com/tsoniclang/gotots/internal/identity"
)

func TestStageInputCannotRetainDerivedPackageInputs(t *testing.T) {
	assertNoPackageInputCollection(t, reflect.TypeOf(stageInput{}))

	type eagerPackageInputs struct {
		packageList []*packageInput
		byPackage   map[identity.PackageID]*packageInput
	}
	if !hasPackageInputCollection(reflect.TypeOf(eagerPackageInputs{})) {
		t.Fatal("package-input residency gate does not detect its negative control")
	}
}

func assertNoPackageInputCollection(t *testing.T, typ reflect.Type) {
	t.Helper()
	if hasPackageInputCollection(typ) {
		t.Fatalf("%s retains derived package inputs", typ)
	}
}

func hasPackageInputCollection(typ reflect.Type) bool {
	input := reflect.TypeOf((*packageInput)(nil))
	for index := 0; index < typ.NumField(); index++ {
		field := typ.Field(index).Type
		switch field.Kind() {
		case reflect.Array, reflect.Slice:
			if field.Elem() == input {
				return true
			}
		case reflect.Map:
			if field.Key() == input || field.Elem() == input {
				return true
			}
		}
	}
	return false
}
