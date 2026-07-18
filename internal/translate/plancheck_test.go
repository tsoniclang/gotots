package translate_test

import (
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/oracle"
)

// TestPlannerSelectsDirectForms verifies the emitted code actually uses
// the native-array lowering for owner-only regions (not just that the
// oracle matches).
func TestPlannerSelectsDirectForms(t *testing.T) {
	generated, _, err := oracle.Translate(t.TempDir(), map[string]string{"fixture": `package fixture

func Case() int {
	values := []int{1, 2}
	values = append(values, 3)
	return len(values)
}
`})
	if err != nil {
		t.Fatal(err)
	}
	var core string
	for path, content := range generated.Files {
		if strings.Contains(path, "fixture/package.ts") {
			core = content
		}
	}
	for _, want := range []string{
		"let values: goabi$.GoInt[] = [(1n), (2n)];",
		"values.push((3n));",
	} {
		if !strings.Contains(core, want) {
			t.Errorf("planned direct form missing %q:\n%s", want, core)
		}
	}
	if strings.Contains(core, "goSliceFrom") || strings.Contains(core, "goSliceAppend") {
		t.Errorf("owner-only region still uses the exact carrier:\n%s", core)
	}
}
