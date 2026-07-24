package semantic

import "fmt"

func (validator normalizedPackageValidator) validateOperations() error {
	for _, record := range validator.pkg.operations.records {
		evidence := validator.pkg.identities.operation(record.id)
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
		if err := validator.requireDefinition(
			identityRecord.definition,
		); err != nil {
			return normalizedRecordError(evidence, err)
		}
		if err := validator.requireType(record.resultType); err != nil {
			return normalizedRecordError(evidence, err)
		}
		if err := validator.requireType(
			record.expectedType,
		); err != nil {
			return normalizedRecordError(evidence, err)
		}
		if err := validator.validateObject(
			record.object,
		); err != nil {
			return normalizedRecordError(evidence, err)
		}
		if err := validator.validateSelection(
			record.selection,
		); err != nil {
			return normalizedRecordError(evidence, err)
		}
		if err := validator.validateInstance(
			record.instance,
		); err != nil {
			return normalizedRecordError(evidence, err)
		}
		operands, present := storedRelation(
			validator.pkg.operations.operands,
			record.operands.start,
			record.operands.count,
		)
		if !present {
			return normalizedRecordError(
				evidence,
				fmt.Errorf("operand range is invalid"),
			)
		}
		for _, occurrence := range operands {
			if err := validator.requireOccurrence(
				occurrence,
			); err != nil {
				return normalizedRecordError(evidence, err)
			}
		}
		definitions, present := storedRelation(
			validator.pkg.operations.definitions,
			record.definitions.start,
			record.definitions.count,
		)
		if !present {
			return normalizedRecordError(
				evidence,
				fmt.Errorf("definition range is invalid"),
			)
		}
		for _, definition := range definitions {
			if err := validator.requireDefinition(
				definition,
			); err != nil {
				return normalizedRecordError(evidence, err)
			}
		}
		implicit, present := storedRelation(
			validator.pkg.operations.implicit,
			record.implicit.start,
			record.implicit.count,
		)
		if !present {
			return normalizedRecordError(
				evidence,
				fmt.Errorf("implicit-operation range is invalid"),
			)
		}
		for _, operation := range implicit {
			if err := validator.requireOccurrence(
				operation.site,
			); err != nil {
				return normalizedRecordError(evidence, err)
			}
			if err := validator.requireType(
				operation.source,
			); err != nil {
				return normalizedRecordError(evidence, err)
			}
			if err := validator.requireType(
				operation.target,
			); err != nil {
				return normalizedRecordError(evidence, err)
			}
		}
		if err := validator.requireOperation(
			record.controlTarget,
		); err != nil {
			return normalizedRecordError(evidence, err)
		}
		if err := validator.requireBinding(record.label); err != nil {
			return normalizedRecordError(evidence, err)
		}
	}
	return nil
}

func (validator normalizedPackageValidator) validateObject(
	reference objectReferenceRef,
) error {
	if reference == 0 {
		return nil
	}
	object, err := payloadAt(
		validator.pkg.operations.objects,
		uint64(reference),
	)
	if err != nil {
		return err
	}
	switch object.kind {
	case ObjectReferenceDeclaration:
		return validator.requireDeclaration(object.declaration)
	case ObjectReferenceBinding:
		return validator.requireBinding(object.binding)
	default:
		return fmt.Errorf(
			"object-reference kind %d is invalid", object.kind,
		)
	}
}

func (validator normalizedPackageValidator) validateSelection(
	reference selectionRef,
) error {
	if reference == 0 {
		return nil
	}
	selection, err := payloadAt(
		validator.pkg.operations.selections,
		uint64(reference),
	)
	if err != nil {
		return err
	}
	if err := validator.requireType(selection.receiver); err != nil {
		return err
	}
	return validator.requireDeclaration(selection.object)
}

func (validator normalizedPackageValidator) validateInstance(
	reference instanceRef,
) error {
	if reference == 0 {
		return nil
	}
	instance, err := payloadAt(
		validator.pkg.operations.instances,
		uint64(reference),
	)
	if err != nil {
		return err
	}
	if err := validator.validateObject(instance.target); err != nil {
		return err
	}
	types, present := storedRelation(
		validator.pkg.operations.instanceTypes,
		instance.types.start,
		instance.types.count,
	)
	if !present {
		return fmt.Errorf("instance-type range is invalid")
	}
	for _, typeID := range types {
		if err := validator.requireType(typeID); err != nil {
			return err
		}
	}
	return validator.requireType(instance.signature)
}
