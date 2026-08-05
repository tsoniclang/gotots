package environmentcontract

import (
	"reflect"
	"testing"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
)

func TestExecutableClosureRejectsProfileBoundary(t *testing.T) {
	failure := &ClosureError{}
	expandClosureEdges(
		"func:example.Root",
		[]string{"provider#boundary@1"},
		map[string]gostdlib.ImplementationDocument{
			"provider#boundary@1": {
				Identity:    "provider#boundary@1",
				Disposition: gostdlib.DispositionProfileBoundary,
			},
		},
		make(map[string]struct{}),
		failure,
	)
	want := []string{"func:example.Root -> provider#boundary@1"}
	if !reflect.DeepEqual(failure.UsedProfileBoundaries, want) {
		t.Fatalf(
			"used profile boundaries = %v; want %v",
			failure.UsedProfileBoundaries,
			want,
		)
	}
	if got := failure.Error(); got != "used-provider closure: used provider profile boundaries: func:example.Root -> provider#boundary@1" {
		t.Fatalf("closure error = %q", got)
	}
}

func TestClosureDispositionClassesStayDistinct(t *testing.T) {
	failure := &ClosureError{}
	recordUsedDisposition(
		"profile",
		gostdlib.DispositionProfileBoundary,
		failure,
	)
	recordUsedDisposition(
		"placeholder",
		gostdlib.DispositionPlaceholder,
		failure,
	)
	if !reflect.DeepEqual(failure.UsedProfileBoundaries, []string{"profile"}) {
		t.Fatalf("profile boundaries = %v", failure.UsedProfileBoundaries)
	}
	if !reflect.DeepEqual(failure.UsedPlaceholders, []string{"placeholder"}) {
		t.Fatalf("placeholders = %v", failure.UsedPlaceholders)
	}
}
