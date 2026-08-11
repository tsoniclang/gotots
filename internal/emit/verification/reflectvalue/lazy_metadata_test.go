package reflectvalue_test

import (
	"strings"
	"testing"
)

func TestReflectionMetadataAndValueOperationsAreDeferred(t *testing.T) {
	project := t.TempDir()
	emission := compileReflectFixture(
		t,
		project,
		`package reflectvalue

import "reflect"

func Facts() int64 { return reflect.ValueOf(int64(1)).Int() }
`,
		[]string{"Facts"},
	)
	artifacts := materializeArtifacts(t, emission, t.TempDir())
	for _, required := range []string{
		".$create(() => ({",
		".$registerValue(",
		", () => ({",
	} {
		if !strings.Contains(artifacts.printed, required) {
			t.Fatalf(
				"reflection artifact lacks lazy constructor %q:\n%s",
				required,
				artifacts.printed,
			)
		}
	}
}
