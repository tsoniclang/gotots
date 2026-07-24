package stagecheck

import (
	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/semantic"
)

func (verifier *checkerSemanticVerifier) resolution(
	id identity.OccurrenceID,
) (semantic.OccurrenceResolution, bool) {
	record, present := verifier.actual.Resolution(id)
	if !present ||
		(verifier.localOnly &&
			!verifier.expected.localOccurrence(
				record.Occurrence(), record.Owner(),
			)) {
		return semantic.OccurrenceResolution{}, false
	}
	return record, true
}

func (verifier *checkerSemanticVerifier) operation(
	id identity.OperationID,
) (semantic.Operation, bool) {
	record, present := verifier.actual.Operation(id)
	if !present || !verifier.operationIsLocal(record) {
		return semantic.Operation{}, false
	}
	return record, true
}

func (verifier *checkerSemanticVerifier) declaration(
	id identity.SemanticDeclarationID,
) (semantic.Declaration, bool) {
	return verifier.actual.Declaration(id)
}

func (verifier *checkerSemanticVerifier) visitBindings(
	visit func(semantic.Binding) error,
) error {
	return verifier.actual.VisitBindings(func(
		record semantic.Binding,
	) error {
		if verifier.localOnly &&
			!verifier.expected.localBinding(record) {
			return nil
		}
		return visit(record)
	})
}

func (verifier *checkerSemanticVerifier) visitOperations(
	visit func(semantic.Operation) error,
) error {
	return verifier.actual.VisitOperations(func(
		record semantic.Operation,
	) error {
		if !verifier.operationIsLocal(record) {
			return nil
		}
		return visit(record)
	})
}

func (verifier *checkerSemanticVerifier) operationIsLocal(
	record semantic.Operation,
) bool {
	if !verifier.localOnly {
		return true
	}
	if record.ID().Source() {
		return verifier.expected.localOccurrence(
			record.Occurrence(), record.Definition(),
		)
	}
	_, local := verifier.expected.definitions[record.Definition()]
	return local
}
