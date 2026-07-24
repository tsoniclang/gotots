package semantic

import (
	"sort"

	"github.com/tsoniclang/gotots/internal/identity"
)

func storedRecordIndex[Reference ~uint64, Record any](
	records []Record,
	reference Reference,
	recordReference func(Record) Reference,
) (int, bool) {
	if reference == 0 {
		return 0, false
	}
	index := sort.Search(len(records), func(index int) bool {
		return recordReference(records[index]) >= reference
	})
	return index, index < len(records) &&
		recordReference(records[index]) == reference
}

func (pkg Package) definitionIndex(
	id identity.DefinitionID,
) (int, bool) {
	return storedRecordIndex(
		pkg.definitions.records,
		pkg.identities.definitionReference(id),
		func(record storedDefinition) definitionRef {
			return record.id
		},
	)
}

func (pkg Package) resolutionIndex(
	id identity.OccurrenceID,
) (int, bool) {
	return storedRecordIndex(
		pkg.resolutions.records,
		pkg.identities.occurrenceReference(id),
		func(record storedResolution) occurrenceRef {
			return record.occurrence
		},
	)
}

func (pkg Package) declarationIndex(
	id identity.SemanticDeclarationID,
) (int, bool) {
	return storedRecordIndex(
		pkg.declarations.records,
		pkg.identities.declarationReference(id),
		func(record storedDeclaration) declarationRef {
			return record.id
		},
	)
}

func (pkg Package) bindingIndex(
	id identity.SemanticBindingID,
) (int, bool) {
	return storedRecordIndex(
		pkg.bindings.records,
		pkg.identities.bindingReference(id),
		func(record storedBinding) bindingRef {
			return record.id
		},
	)
}

func (pkg Package) typeIndex(
	id identity.SemanticTypeID,
) (int, bool) {
	return storedRecordIndex(
		pkg.types.records,
		pkg.identities.typeReference(id),
		func(record storedType) typeRef {
			return record.id
		},
	)
}

func (pkg Package) typeWitnessIndex(
	id identity.SemanticTypeID,
) (int, bool) {
	return storedRecordIndex(
		pkg.witnesses.records,
		pkg.identities.typeReference(id),
		func(record storedTypeWitness) typeRef {
			return record.typeID
		},
	)
}

func (pkg Package) operationIndex(
	id identity.OperationID,
) (int, bool) {
	return storedRecordIndex(
		pkg.operations.records,
		pkg.identities.operationReference(id),
		func(record storedOperation) operationRef {
			return record.id
		},
	)
}

func (pkg Package) unsupportedIndex(
	id identity.UnsupportedID,
) (int, bool) {
	return storedRecordIndex(
		pkg.unsupported.records,
		pkg.identities.unsupportedReference(id),
		func(record storedUnsupported) unsupportedRef {
			return record.id
		},
	)
}

func (pkg Package) HasDefinition(id identity.DefinitionID) bool {
	_, present := pkg.definitionIndex(id)
	return present
}

func (pkg Package) HasResolution(id identity.OccurrenceID) bool {
	_, present := pkg.resolutionIndex(id)
	return present
}

func (pkg Package) HasDeclaration(
	id identity.SemanticDeclarationID,
) bool {
	_, present := pkg.declarationIndex(id)
	return present
}

func (pkg Package) HasBinding(id identity.SemanticBindingID) bool {
	_, present := pkg.bindingIndex(id)
	return present
}

func (pkg Package) HasType(id identity.SemanticTypeID) bool {
	_, present := pkg.typeIndex(id)
	return present
}

func (pkg Package) HasOperation(id identity.OperationID) bool {
	_, present := pkg.operationIndex(id)
	return present
}

func (pkg Package) HasUnsupported(id identity.UnsupportedID) bool {
	_, present := pkg.unsupportedIndex(id)
	return present
}
