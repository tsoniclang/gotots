package stagecheck

import (
	"reflect"

	"github.com/tsoniclang/gotots/internal/language/semantic"
)

// VerifyFinalizedStage2 proves that the finalized semantic model cannot expose
// or retain the transient Go syntax/checker graph.
func VerifyFinalizedStage2(model *semantic.Model) error {
	if model == nil {
		return semanticVerificationError(
			"finalization", "semantic model is absent",
		)
	}
	if path := rawFacadePath(
		reflect.TypeOf(model),
		map[reflect.Type]bool{},
	); path != "" {
		return semanticVerificationError(
			"finalization",
			"raw transient facade remains reachable at "+path,
		)
	}
	return nil
}
