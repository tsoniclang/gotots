package semantic

func (encoder *wireTypeEncoder) referenceRange(
	value typeRefRange,
) (wireReferenceRange[wireTypeReference], error) {
	return encodeReferenceRange(
		encoder.store.typeRelations,
		value.start,
		value.count,
		&encoder.references,
		encoder.identities.typeID,
	)
}

func (encoder *wireTypeEncoder) fieldRange(
	value typeFieldRange,
) (wireTypeFieldRange, error) {
	selected, err := relationSlice(
		encoder.store.fields, value.start, value.count,
	)
	if err != nil {
		return wireTypeFieldRange{}, err
	}
	out := wireTypeFieldRange{
		Start:  encoder.fields,
		Count:  value.count,
		Values: make([]wireTypeField, 0, len(selected)),
	}
	for _, field := range selected {
		pkg, encodeErr := encoder.identities.packageID(field.pkg)
		if encodeErr != nil {
			return wireTypeFieldRange{}, encodeErr
		}
		typeID, encodeErr := encoder.identities.typeID(field.typeID)
		if encodeErr != nil {
			return wireTypeFieldRange{}, encodeErr
		}
		out.Values = append(out.Values, wireTypeField{
			Name:     field.name,
			Package:  pkg,
			Type:     typeID,
			Embedded: field.embedded,
			Tag:      field.tag,
			Ordinal:  field.ordinal,
		})
	}
	encoder.fields += value.count
	return out, nil
}

func (encoder *wireTypeEncoder) methodRange(
	value typeMethodRange,
) (wireTypeMethodRange, error) {
	selected, err := relationSlice(
		encoder.store.methods, value.start, value.count,
	)
	if err != nil {
		return wireTypeMethodRange{}, err
	}
	out := wireTypeMethodRange{
		Start:  encoder.methods,
		Count:  value.count,
		Values: make([]wireTypeMethod, 0, len(selected)),
	}
	for _, method := range selected {
		pkg, encodeErr := encoder.identities.packageID(method.pkg)
		if encodeErr != nil {
			return wireTypeMethodRange{}, encodeErr
		}
		signature, encodeErr := encoder.identities.typeID(
			method.signature,
		)
		if encodeErr != nil {
			return wireTypeMethodRange{}, encodeErr
		}
		out.Values = append(out.Values, wireTypeMethod{
			Name:      method.name,
			Package:   pkg,
			Signature: signature,
			Ordinal:   method.ordinal,
		})
	}
	encoder.methods += value.count
	return out, nil
}

func (encoder *wireTypeEncoder) termRange(
	value typeTermRange,
) (wireTypeTermRange, error) {
	selected, err := relationSlice(
		encoder.store.terms, value.start, value.count,
	)
	if err != nil {
		return wireTypeTermRange{}, err
	}
	out := wireTypeTermRange{
		Start:  encoder.terms,
		Count:  value.count,
		Values: make([]wireTypeTerm, 0, len(selected)),
	}
	for _, term := range selected {
		typeID, encodeErr := encoder.identities.typeID(term.typeID)
		if encodeErr != nil {
			return wireTypeTermRange{}, encodeErr
		}
		out.Values = append(out.Values, wireTypeTerm{
			Tilde: term.tilde, Type: typeID,
		})
	}
	encoder.terms += value.count
	return out, nil
}
