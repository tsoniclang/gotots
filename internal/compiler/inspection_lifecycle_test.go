package compiler

import (
	"testing"

	"github.com/tsoniclang/gotots/internal/source"
)

func inspectConstructsForTest(
	t *testing.T,
	request source.Request,
) (*Inspection, error) {
	t.Helper()
	inspection, err := InspectConstructs(request)
	if inspection != nil {
		t.Cleanup(func() {
			if closeErr := inspection.Close(); closeErr != nil {
				t.Errorf("close inspection: %v", closeErr)
			}
		})
	}
	return inspection, err
}
