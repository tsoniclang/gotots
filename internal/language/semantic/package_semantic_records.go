package semantic

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/identity"
)

func (validator *normalizedPackageValidator) validateDefinitions() error {
	for _, record := range validator.pkg.definitions.records {
		if record.pkg != validator.packageRef {
			return fmt.Errorf(
				"semantic definition %s has invalid package owner",
				validator.pkg.identities.definition(record.id),
			)
		}
		if err := validator.requireAuthority(
			record.authority,
		); err != nil {
			return normalizedRecordError(
				validator.pkg.identities.definition(record.id),
				err,
			)
		}
		bindings, present := storedRelation(
			validator.pkg.definitions.bindingRelations,
			record.bindings.start,
			record.bindings.count,
		)
		if !present {
			return normalizedRecordError(
				validator.pkg.identities.definition(record.id),
				fmt.Errorf("binding range is invalid"),
			)
		}
		for _, binding := range bindings {
			if err := validator.requireBinding(binding); err != nil {
				return normalizedRecordError(
					validator.pkg.identities.definition(record.id),
					err,
				)
			}
		}
		if err := validator.validateDefinitionPayload(record); err != nil {
			return normalizedRecordError(
				validator.pkg.identities.definition(record.id),
				err,
			)
		}
	}
	return nil
}

func (validator normalizedPackageValidator) validateDefinitionPayload(
	record storedDefinition,
) error {
	store := validator.pkg.definitions
	switch record.form {
	case DefinitionFormCallable:
		payload, err := payloadAt(store.callable, record.payload)
		if err != nil {
			return err
		}
		if err := validator.requireDeclarationRange(
			payload.declarations,
		); err != nil {
			return err
		}
		if err := validator.requireType(payload.signature); err != nil {
			return err
		}
		return validator.requireBinding(payload.receiver)
	case DefinitionFormInitializer:
		payload, err := payloadAt(
			store.initializers, record.payload,
		)
		if err != nil {
			return err
		}
		if err := validator.requireDeclarationRange(
			payload.declarations,
		); err != nil {
			return err
		}
		entries, present := storedRelation(
			store.initializerEntries,
			payload.entries.start,
			payload.entries.count,
		)
		if !present {
			return fmt.Errorf("initializer range is invalid")
		}
		for _, occurrence := range entries {
			if err := validator.requireOccurrence(
				occurrence,
			); err != nil {
				return err
			}
		}
		return nil
	case DefinitionFormBodyless:
		payload, err := payloadAt(store.bodyless, record.payload)
		if err != nil {
			return err
		}
		if err := validator.requireOwnedDeclaration(
			payload.declaration,
		); err != nil {
			return err
		}
		if err := validator.requireType(payload.signature); err != nil {
			return err
		}
		return validator.requireBinding(payload.receiver)
	case DefinitionFormImplicit:
		_, err := payloadAt(store.implicit, record.payload)
		return err
	case DefinitionFormSynthetic:
		payload, err := payloadAt(store.synthetic, record.payload)
		if err != nil {
			return err
		}
		if err := validator.requireOwnedDeclaration(
			payload.declaration,
		); err != nil {
			return err
		}
		return validator.requireType(payload.signature)
	default:
		return fmt.Errorf(
			"definition form %d is invalid", record.form,
		)
	}
}

func (validator normalizedPackageValidator) requireDeclarationRange(
	relation declarationRefRange,
) error {
	declarations, present := storedRelation(
		validator.pkg.definitions.declarationRelations,
		relation.start,
		relation.count,
	)
	if !present {
		return fmt.Errorf("declaration range is invalid")
	}
	for _, declaration := range declarations {
		if err := validator.requireOwnedDeclaration(
			declaration,
		); err != nil {
			return err
		}
	}
	return nil
}

func (validator *normalizedPackageValidator) validateDeclarations() error {
	for _, record := range validator.pkg.declarations.records {
		id := validator.pkg.identities.declaration(record.id)
		if record.pkg != validator.packageRef ||
			id.Form() == identity.SemanticDeclarationMember {
			return fmt.Errorf(
				"semantic declaration %s has invalid package ownership",
				id,
			)
		}
		if err := validator.requireType(record.typeID); err != nil {
			return normalizedRecordError(id, err)
		}
		if err := validator.requireAuthority(
			record.authority,
		); err != nil {
			return normalizedRecordError(id, err)
		}
	}
	return nil
}

func (validator *normalizedPackageValidator) validateBindings() error {
	for _, record := range validator.pkg.bindings.records {
		id := validator.pkg.identities.binding(record.id)
		if record.pkg != validator.packageRef {
			return fmt.Errorf(
				"semantic binding %s has invalid package owner", id,
			)
		}
		if err := validator.requireDefinition(
			record.definition,
		); err != nil {
			return normalizedRecordError(id, err)
		}
		if err := validator.requireType(record.typeID); err != nil {
			return normalizedRecordError(id, err)
		}
		captures, present := storedRelation(
			validator.pkg.bindings.captures,
			record.captures.start,
			record.captures.count,
		)
		if !present {
			return normalizedRecordError(
				id,
				fmt.Errorf("capture range is invalid"),
			)
		}
		for _, definition := range captures {
			if err := validator.requireDefinition(
				definition,
			); err != nil {
				return normalizedRecordError(id, err)
			}
		}
		if err := validator.requireAuthority(
			record.authority,
		); err != nil {
			return normalizedRecordError(id, err)
		}
	}
	return nil
}

func (validator *normalizedPackageValidator) validateTypeWitnesses() error {
	if len(validator.pkg.witnesses.records) !=
		len(validator.pkg.types.records) {
		return fmt.Errorf(
			"semantic package %s has %d types and %d witnesses",
			validator.pkg.id,
			len(validator.pkg.types.records),
			len(validator.pkg.witnesses.records),
		)
	}
	for index, witness := range validator.pkg.witnesses.records {
		typeRecord := validator.pkg.types.records[index]
		if witness.pkg != validator.packageRef ||
			witness.typeID != typeRecord.id ||
			!referenceInSet(
				validator.index.witnesses,
				witness.typeID,
			) {
			return fmt.Errorf(
				"semantic package %s has invalid type witness %s",
				validator.pkg.id,
				validator.pkg.identities.typeID(witness.typeID),
			)
		}
		if err := validator.requireAuthority(
			witness.authority,
		); err != nil {
			return normalizedRecordError(
				validator.pkg.identities.typeID(witness.typeID),
				err,
			)
		}
	}
	return nil
}

func (validator *normalizedPackageValidator) validateUnsupported() error {
	for _, record := range validator.pkg.unsupported.records {
		id := validator.pkg.identities.unsupportedID(record.id)
		identityRecord, present := componentAt(
			validator.pkg.identities.unsupported,
			record.id,
		)
		if !present {
			return fmt.Errorf(
				"semantic unsupported identity %d is absent",
				record.id,
			)
		}
		if err := validator.requireDefinition(
			identityRecord.definition,
		); err != nil {
			return normalizedRecordError(id, err)
		}
		if err := validator.requireAuthority(
			record.authority,
		); err != nil {
			return normalizedRecordError(id, err)
		}
	}
	return nil
}

func (validator normalizedPackageValidator) validateResolutions() error {
	sourceOperations := 0
	for _, record := range validator.pkg.operations.records {
		identityRecord, present := componentAt(
			validator.pkg.identities.operations,
			record.id,
		)
		if !present {
			return fmt.Errorf(
				"semantic operation identity %d is absent",
				record.id,
			)
		}
		if identityRecord.occurrence != 0 {
			sourceOperations++
		}
	}
	operationResolutions := 0
	for _, record := range validator.pkg.resolutions.records {
		evidence := validator.pkg.identities.occurrence(
			record.occurrence,
		)
		if err := validator.requireDefinition(
			record.owner,
		); err != nil {
			return normalizedRecordError(evidence, err)
		}
		if err := validator.validateResolutionPayload(
			record,
		); err != nil {
			return normalizedRecordError(evidence, err)
		}
		if record.kind == ResolutionOperation {
			operationResolutions++
			if err := validator.validateOperationResolution(
				record,
			); err != nil {
				return normalizedRecordError(evidence, err)
			}
		}
	}
	if sourceOperations != operationResolutions {
		return fmt.Errorf(
			"semantic package %s has %d source operations and %d operation resolutions",
			validator.pkg.id,
			sourceOperations,
			operationResolutions,
		)
	}
	return nil
}

func (validator normalizedPackageValidator) validateOperationResolution(
	record storedResolution,
) error {
	operation, err := payloadAt(
		validator.pkg.resolutions.operations,
		record.payload,
	)
	if err != nil {
		return err
	}
	origin, present := componentAt(
		validator.pkg.identities.operations,
		operation,
	)
	if !present {
		return fmt.Errorf("operation identity %d is absent", operation)
	}
	if origin.implicit != 0 ||
		origin.occurrence != record.occurrence ||
		origin.definition != record.owner {
		return fmt.Errorf(
			"operation resolution differs from operation origin %s",
			validator.pkg.identities.operation(operation),
		)
	}
	return nil
}

func (validator normalizedPackageValidator) validateResolutionPayload(
	record storedResolution,
) error {
	store := validator.pkg.resolutions
	switch record.kind {
	case ResolutionStructuralOnly:
		payload, err := payloadAt(store.structural, record.payload)
		if err != nil {
			return err
		}
		if err := validator.requireDeclaration(
			payload.declaration,
		); err != nil {
			return err
		}
		return validator.requireType(payload.typeID)
	case ResolutionDefinitionComponent:
		payload, err := payloadAt(
			store.definitionComponents, record.payload,
		)
		if err != nil {
			return err
		}
		return validator.requireDefinition(payload.definition)
	case ResolutionDeclaration:
		payload, err := payloadAt(store.declarations, record.payload)
		if err != nil {
			return err
		}
		return validator.requireDeclaration(payload)
	case ResolutionBinding:
		payload, err := payloadAt(store.bindings, record.payload)
		if err != nil {
			return err
		}
		return validator.requireBinding(payload)
	case ResolutionType:
		payload, err := payloadAt(store.types, record.payload)
		if err != nil {
			return err
		}
		return validator.requireType(payload)
	case ResolutionOperation:
		payload, err := payloadAt(store.operations, record.payload)
		if err != nil {
			return err
		}
		return validator.requireOperation(payload)
	case ResolutionUnsupported:
		payload, err := payloadAt(store.unsupported, record.payload)
		if err != nil {
			return err
		}
		return validator.requireUnsupported(payload)
	default:
		return fmt.Errorf(
			"resolution kind %d is invalid", record.kind,
		)
	}
}

func normalizedRecordError[Evidence fmt.Stringer](
	evidence Evidence,
	err error,
) error {
	return fmt.Errorf("%s references %w", evidence, err)
}
