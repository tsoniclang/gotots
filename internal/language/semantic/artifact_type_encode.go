package semantic

type wireTypeEncoder struct {
	identities wireIdentityEncoder
	store      packageTypeStore
	references uint64
	fields     uint64
	methods    uint64
	terms      uint64
}

func (encoder *wireTypeEncoder) record(
	index int,
) (wireTypeRecord, error) {
	stored := encoder.store.records[index]
	id, err := encoder.identities.typeID(stored.id)
	if err != nil {
		return wireTypeRecord{}, err
	}
	payload, err := encoder.payload(stored)
	if err != nil {
		return wireTypeRecord{}, err
	}
	return wireTypeRecord{
		ID:      id,
		Kind:    uint8(stored.kind),
		Payload: payload,
	}, nil
}

func (encoder *wireTypeEncoder) payload(
	stored storedType,
) (wireTypePayload, error) {
	out := wireTypePayload{Tag: uint8(stored.kind)}
	switch stored.kind {
	case TypeBasic:
		value, err := payloadAt(encoder.store.basic, stored.payload)
		if err != nil {
			return wireTypePayload{}, err
		}
		out.Basic = &wireBasicType{Kind: uint8(value)}
	case TypeNamed, TypeAlias:
		value, err := payloadAt(encoder.store.nominal, stored.payload)
		if err != nil {
			return wireTypePayload{}, err
		}
		payload, err := encoder.nominal(value)
		if err != nil {
			return wireTypePayload{}, err
		}
		out.Nominal = &payload
	case TypeParameter:
		value, err := payloadAt(
			encoder.store.parameters, stored.payload,
		)
		if err != nil {
			return wireTypePayload{}, err
		}
		payload, err := encoder.parameter(value)
		if err != nil {
			return wireTypePayload{}, err
		}
		out.Parameter = &payload
	case TypePointer, TypeSlice:
		value, err := payloadAt(encoder.store.elements, stored.payload)
		if err != nil {
			return wireTypePayload{}, err
		}
		element, err := encoder.identities.typeID(value)
		if err != nil {
			return wireTypePayload{}, err
		}
		out.Element = &wireElementType{Element: element}
	case TypeArray:
		value, err := payloadAt(encoder.store.arrays, stored.payload)
		if err != nil {
			return wireTypePayload{}, err
		}
		element, err := encoder.identities.typeID(value.element)
		if err != nil {
			return wireTypePayload{}, err
		}
		out.Array = &wireArrayType{
			Element: element, Length: value.length,
		}
	case TypeMap:
		value, err := payloadAt(encoder.store.maps, stored.payload)
		if err != nil {
			return wireTypePayload{}, err
		}
		key, err := encoder.identities.typeID(value.key)
		if err != nil {
			return wireTypePayload{}, err
		}
		element, err := encoder.identities.typeID(value.element)
		if err != nil {
			return wireTypePayload{}, err
		}
		out.Map = &wireMapType{Key: key, Element: element}
	case TypeChannel:
		value, err := payloadAt(encoder.store.channels, stored.payload)
		if err != nil {
			return wireTypePayload{}, err
		}
		element, err := encoder.identities.typeID(value.element)
		if err != nil {
			return wireTypePayload{}, err
		}
		out.Channel = &wireChannelType{
			Element:   element,
			Direction: uint8(value.direction),
		}
	case TypeSignature:
		value, err := payloadAt(
			encoder.store.signatures, stored.payload,
		)
		if err != nil {
			return wireTypePayload{}, err
		}
		payload, err := encoder.signature(value)
		if err != nil {
			return wireTypePayload{}, err
		}
		out.Signature = &payload
	case TypeStruct:
		value, err := payloadAt(encoder.store.structs, stored.payload)
		if err != nil {
			return wireTypePayload{}, err
		}
		fields, err := encoder.fieldRange(value)
		if err != nil {
			return wireTypePayload{}, err
		}
		out.Struct = &wireStructType{Fields: fields}
	case TypeInterface:
		value, err := payloadAt(
			encoder.store.interfaces, stored.payload,
		)
		if err != nil {
			return wireTypePayload{}, err
		}
		payload, err := encoder.interfaceType(value)
		if err != nil {
			return wireTypePayload{}, err
		}
		out.Interface = &payload
	case TypeTuple:
		value, err := payloadAt(encoder.store.tuples, stored.payload)
		if err != nil {
			return wireTypePayload{}, err
		}
		elements, err := encoder.referenceRange(value)
		if err != nil {
			return wireTypePayload{}, err
		}
		out.Tuple = &wireTupleType{Elements: elements}
	case TypeUnion:
		value, err := payloadAt(encoder.store.unions, stored.payload)
		if err != nil {
			return wireTypePayload{}, err
		}
		terms, err := encoder.termRange(value)
		if err != nil {
			return wireTypePayload{}, err
		}
		out.Union = &wireUnionType{Terms: terms}
	default:
		return wireTypePayload{}, &artifactError{
			reason: "semantic type has invalid normalized payload",
		}
	}
	return out, nil
}

func (encoder *wireTypeEncoder) nominal(
	value storedNominalType,
) (wireNominalType, error) {
	declaration, err := encoder.identities.declaration(value.declaration)
	if err != nil {
		return wireNominalType{}, err
	}
	arguments, err := encoder.referenceRange(value.arguments)
	if err != nil {
		return wireNominalType{}, err
	}
	target, err := encoder.identities.typeID(value.target)
	if err != nil {
		return wireNominalType{}, err
	}
	methods, err := encoder.methodRange(value.methods)
	if err != nil {
		return wireNominalType{}, err
	}
	return wireNominalType{
		Declaration: declaration,
		Arguments:   arguments,
		Target:      target,
		Methods:     methods,
	}, nil
}

func (encoder *wireTypeEncoder) parameter(
	value storedTypeParameter,
) (wireTypeParameter, error) {
	declaration, err := encoder.identities.declaration(value.declaration)
	if err != nil {
		return wireTypeParameter{}, err
	}
	definition, err := encoder.identities.definition(value.definition)
	if err != nil {
		return wireTypeParameter{}, err
	}
	constraint, err := encoder.identities.typeID(value.constraint)
	if err != nil {
		return wireTypeParameter{}, err
	}
	return wireTypeParameter{
		Declaration: declaration,
		Definition:  definition,
		Role:        uint8(value.role),
		Ordinal:     value.ordinal,
		Constraint:  constraint,
	}, nil
}

func (encoder *wireTypeEncoder) signature(
	value storedSignature,
) (wireSignatureType, error) {
	receiver, err := encoder.identities.typeID(value.receiver)
	if err != nil {
		return wireSignatureType{}, err
	}
	receiverParameters, err := encoder.referenceRange(
		value.receiverTypeParameters,
	)
	if err != nil {
		return wireSignatureType{}, err
	}
	typeParameters, err := encoder.referenceRange(
		value.typeParameters,
	)
	if err != nil {
		return wireSignatureType{}, err
	}
	parameters, err := encoder.referenceRange(value.parameters)
	if err != nil {
		return wireSignatureType{}, err
	}
	results, err := encoder.referenceRange(value.results)
	if err != nil {
		return wireSignatureType{}, err
	}
	return wireSignatureType{
		Receiver:               receiver,
		ReceiverTypeParameters: receiverParameters,
		TypeParameters:         typeParameters,
		Parameters:             parameters,
		Results:                results,
		Variadic:               value.variadic,
	}, nil
}

func (encoder *wireTypeEncoder) interfaceType(
	value storedInterfaceType,
) (wireInterfaceType, error) {
	methods, err := encoder.methodRange(value.methods)
	if err != nil {
		return wireInterfaceType{}, err
	}
	embeddeds, err := encoder.referenceRange(value.embeddeds)
	if err != nil {
		return wireInterfaceType{}, err
	}
	terms, err := encoder.termRange(value.terms)
	if err != nil {
		return wireInterfaceType{}, err
	}
	return wireInterfaceType{
		Methods:    methods,
		Embeddeds:  embeddeds,
		Terms:      terms,
		TypeSet:    uint8(value.typeSet),
		Comparable: value.comparable,
	}, nil
}
