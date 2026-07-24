package semantic

import "fmt"

func (validator normalizedPackageValidator) validateTypes() error {
	for _, record := range validator.pkg.types.records {
		if err := validator.validateTypeRecord(record); err != nil {
			return normalizedRecordError(
				validator.pkg.identities.typeID(record.id),
				err,
			)
		}
	}
	return nil
}

func (validator normalizedPackageValidator) validateTypeRecord(
	record storedType,
) error {
	store := validator.pkg.types
	switch record.kind {
	case TypeBasic:
		_, err := payloadAt(store.basic, record.payload)
		return err
	case TypeNamed, TypeAlias:
		payload, err := payloadAt(store.nominal, record.payload)
		if err != nil {
			return err
		}
		if err := validator.requireDeclaration(
			payload.declaration,
		); err != nil {
			return err
		}
		if err := validator.requireTypeRange(
			payload.arguments,
		); err != nil {
			return err
		}
		if err := validator.requireType(payload.target); err != nil {
			return err
		}
		return validator.validateTypeMethods(payload.methods)
	case TypeParameter:
		payload, err := payloadAt(store.parameters, record.payload)
		if err != nil {
			return err
		}
		if err := validator.requireDeclaration(
			payload.declaration,
		); err != nil {
			return err
		}
		if err := validator.requireDefinitionIdentity(
			payload.definition,
		); err != nil {
			return err
		}
		return validator.requireType(payload.constraint)
	case TypePointer, TypeSlice:
		payload, err := payloadAt(store.elements, record.payload)
		if err != nil {
			return err
		}
		return validator.requireType(payload)
	case TypeArray:
		payload, err := payloadAt(store.arrays, record.payload)
		if err != nil {
			return err
		}
		return validator.requireType(payload.element)
	case TypeMap:
		payload, err := payloadAt(store.maps, record.payload)
		if err != nil {
			return err
		}
		if err := validator.requireType(payload.key); err != nil {
			return err
		}
		return validator.requireType(payload.element)
	case TypeChannel:
		payload, err := payloadAt(store.channels, record.payload)
		if err != nil {
			return err
		}
		return validator.requireType(payload.element)
	case TypeSignature:
		payload, err := payloadAt(store.signatures, record.payload)
		if err != nil {
			return err
		}
		if err := validator.requireType(payload.receiver); err != nil {
			return err
		}
		ranges := [...]typeRefRange{
			payload.receiverTypeParameters,
			payload.typeParameters,
			payload.parameters,
			payload.results,
		}
		for _, relation := range ranges {
			if err := validator.requireTypeRange(
				relation,
			); err != nil {
				return err
			}
		}
		return nil
	case TypeStruct:
		payload, err := payloadAt(store.structs, record.payload)
		if err != nil {
			return err
		}
		return validator.validateTypeFields(payload)
	case TypeInterface:
		payload, err := payloadAt(store.interfaces, record.payload)
		if err != nil {
			return err
		}
		if err := validator.validateTypeMethods(
			payload.methods,
		); err != nil {
			return err
		}
		if err := validator.requireTypeRange(
			payload.embeddeds,
		); err != nil {
			return err
		}
		return validator.validateTypeTerms(payload.terms)
	case TypeTuple:
		payload, err := payloadAt(store.tuples, record.payload)
		if err != nil {
			return err
		}
		return validator.requireTypeRange(payload)
	case TypeUnion:
		payload, err := payloadAt(store.unions, record.payload)
		if err != nil {
			return err
		}
		return validator.validateTypeTerms(payload)
	default:
		return fmt.Errorf("type kind %d is invalid", record.kind)
	}
}

func (validator normalizedPackageValidator) requireTypeRange(
	relation typeRefRange,
) error {
	types, present := storedRelation(
		validator.pkg.types.typeRelations,
		relation.start,
		relation.count,
	)
	if !present {
		return fmt.Errorf("type-reference range is invalid")
	}
	for _, typeID := range types {
		if err := validator.requireType(typeID); err != nil {
			return err
		}
	}
	return nil
}

func (validator normalizedPackageValidator) validateTypeFields(
	relation typeFieldRange,
) error {
	fields, present := storedRelation(
		validator.pkg.types.fields,
		relation.start,
		relation.count,
	)
	if !present {
		return fmt.Errorf("type-field range is invalid")
	}
	for index, field := range fields {
		if field.ordinal != index ||
			!validator.validOptionalPackage(field.pkg) {
			return fmt.Errorf(
				"type field %d is not canonical", index,
			)
		}
		if err := validator.requireType(field.typeID); err != nil {
			return err
		}
	}
	return nil
}

func (validator normalizedPackageValidator) validateTypeMethods(
	relation typeMethodRange,
) error {
	methods, present := storedRelation(
		validator.pkg.types.methods,
		relation.start,
		relation.count,
	)
	if !present {
		return fmt.Errorf("type-method range is invalid")
	}
	for index, method := range methods {
		if method.ordinal != index ||
			!validator.validOptionalPackage(method.pkg) {
			return fmt.Errorf(
				"type method %d is not canonical", index,
			)
		}
		if index != 0 {
			previous := methods[index-1]
			if previous.pkg > method.pkg ||
				(previous.pkg == method.pkg &&
					previous.name >= method.name) {
				return fmt.Errorf(
					"type methods are not canonical at %d",
					index,
				)
			}
		}
		if err := validator.validateMethodSignature(
			method.signature,
		); err != nil {
			return err
		}
	}
	return nil
}

func (validator normalizedPackageValidator) validateMethodSignature(
	reference typeRef,
) error {
	if err := validator.requireType(reference); err != nil {
		return err
	}
	record, present := validator.pkg.types.storedType(reference)
	if !present || record.kind != TypeSignature {
		return fmt.Errorf("type method has a non-signature type")
	}
	signature, err := payloadAt(
		validator.pkg.types.signatures, record.payload,
	)
	if err != nil {
		return err
	}
	if signature.receiver != 0 ||
		signature.receiverTypeParameters.count != 0 {
		return fmt.Errorf(
			"method descriptor signature retains receiver",
		)
	}
	return nil
}

func (validator normalizedPackageValidator) validateTypeTerms(
	relation typeTermRange,
) error {
	terms, present := storedRelation(
		validator.pkg.types.terms,
		relation.start,
		relation.count,
	)
	if !present {
		return fmt.Errorf("type-term range is invalid")
	}
	for _, term := range terms {
		if err := validator.requireType(term.typeID); err != nil {
			return err
		}
	}
	return nil
}

func (validator normalizedPackageValidator) validOptionalPackage(
	reference packageRef,
) bool {
	return reference == 0 ||
		uint64(reference) <=
			uint64(len(validator.pkg.identities.packages))
}
