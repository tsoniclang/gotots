package stagecheck

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/language/semantic"
)

func (verifier *checkerSemanticVerifier) verifyBindingRecord(
	record semantic.Binding,
	expected *checkerBindingCandidate,
) error {
	if record.Package() != verifier.expected.id ||
		record.Definition() != expected.definition ||
		record.Role() != expected.role ||
		record.Name() != expected.name ||
		record.Source() != expected.source {
		return fmt.Errorf(
			"semantic=%s/%s/%s/%q/%s checker=%s/%s/%s/%q/%s",
			record.Package(),
			record.Definition(),
			record.Role(),
			record.Name(),
			record.Source(),
			verifier.expected.id,
			expected.definition,
			expected.role,
			expected.name,
			expected.source,
		)
	}
	if expected.typ == nil {
		if !record.Type().IsZero() {
			return fmt.Errorf("typeless checker binding carries a type")
		}
		return nil
	}
	return verifier.types.verify(record.Type(), expected.typ)
}
