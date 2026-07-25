package semantic

import "fmt"

func readBinaryTypes(
	decoder *binaryShardDecoder,
	expected int,
) (packageTypeStore, error) {
	count, err := readExpectedRecordCount(
		decoder, "types", expected,
	)
	if err != nil {
		return packageTypeStore{}, err
	}
	store := packageTypeStore{
		records: make([]storedType, 0, count),
	}
	for index := 0; index < count; index++ {
		if err := readBinaryTypeRecord(decoder, &store); err != nil {
			return packageTypeStore{}, fmt.Errorf(
				"decode semantic binary type %d: %w",
				index,
				err,
			)
		}
	}
	return store, nil
}

func readBinaryTypeRecord(
	decoder *binaryShardDecoder,
	store *packageTypeStore,
) error {
	id, err := readIdentityReference[typeRef](decoder, "type id")
	if err != nil {
		return err
	}
	kind, err := readUnsignedAs[TypeKind](decoder, "type kind")
	if err != nil {
		return err
	}
	record := storedType{id: id, kind: kind}
	switch kind {
	case TypeBasic:
		value, readErr := readUnsignedAs[BasicKind](
			decoder, "basic type kind",
		)
		if readErr != nil {
			return readErr
		}
		store.basic = append(store.basic, value)
		record.payload = uint64(len(store.basic))
	case TypeNamed, TypeAlias:
		payload, readErr := readBinaryNominalType(decoder, store)
		if readErr != nil {
			return readErr
		}
		store.nominal = append(store.nominal, payload)
		record.payload = uint64(len(store.nominal))
	case TypeParameter:
		payload, readErr := readBinaryTypeParameter(decoder)
		if readErr != nil {
			return readErr
		}
		store.parameters = append(store.parameters, payload)
		record.payload = uint64(len(store.parameters))
	case TypePointer, TypeSlice:
		element, readErr := readIdentityReference[typeRef](
			decoder, "element type",
		)
		if readErr != nil {
			return readErr
		}
		store.elements = append(store.elements, element)
		record.payload = uint64(len(store.elements))
	case TypeArray:
		payload, readErr := readBinaryArrayType(decoder)
		if readErr != nil {
			return readErr
		}
		store.arrays = append(store.arrays, payload)
		record.payload = uint64(len(store.arrays))
	case TypeMap:
		payload, readErr := readBinaryMapType(decoder)
		if readErr != nil {
			return readErr
		}
		store.maps = append(store.maps, payload)
		record.payload = uint64(len(store.maps))
	case TypeChannel:
		payload, readErr := readBinaryChannelType(decoder)
		if readErr != nil {
			return readErr
		}
		store.channels = append(store.channels, payload)
		record.payload = uint64(len(store.channels))
	case TypeSignature:
		payload, readErr := readBinarySignature(decoder, store)
		if readErr != nil {
			return readErr
		}
		store.signatures = append(store.signatures, payload)
		record.payload = uint64(len(store.signatures))
	case TypeStruct:
		payload, readErr := readBinaryFields(decoder, store)
		if readErr != nil {
			return readErr
		}
		store.structs = append(store.structs, payload)
		record.payload = uint64(len(store.structs))
	case TypeInterface:
		payload, readErr := readBinaryInterface(decoder, store)
		if readErr != nil {
			return readErr
		}
		store.interfaces = append(store.interfaces, payload)
		record.payload = uint64(len(store.interfaces))
	case TypeTuple:
		payload, readErr := readBinaryTypeRelation(decoder, store)
		if readErr != nil {
			return readErr
		}
		store.tuples = append(store.tuples, payload)
		record.payload = uint64(len(store.tuples))
	case TypeUnion:
		payload, readErr := readBinaryTerms(decoder, store)
		if readErr != nil {
			return readErr
		}
		store.unions = append(store.unions, payload)
		record.payload = uint64(len(store.unions))
	default:
		return fmt.Errorf(
			"semantic binary type kind %d is invalid", kind,
		)
	}
	store.records = append(store.records, record)
	return nil
}

func readBinaryNominalType(
	decoder *binaryShardDecoder,
	store *packageTypeStore,
) (storedNominalType, error) {
	declaration, err := readIdentityReference[declarationRef](
		decoder, "nominal declaration",
	)
	if err != nil {
		return storedNominalType{}, err
	}
	arguments, err := readBinaryTypeRelation(decoder, store)
	if err != nil {
		return storedNominalType{}, err
	}
	target, err := readIdentityReference[typeRef](
		decoder, "nominal target",
	)
	if err != nil {
		return storedNominalType{}, err
	}
	methods, err := readBinaryMethods(decoder, store)
	return storedNominalType{
		declaration: declaration,
		arguments:   arguments,
		target:      target,
		methods:     methods,
	}, err
}

func readBinaryTypeParameter(
	decoder *binaryShardDecoder,
) (storedTypeParameter, error) {
	declaration, err := readIdentityReference[declarationRef](
		decoder, "type parameter declaration",
	)
	if err != nil {
		return storedTypeParameter{}, err
	}
	definition, err := readIdentityReference[definitionRef](
		decoder, "type parameter definition",
	)
	if err != nil {
		return storedTypeParameter{}, err
	}
	role, err := readUnsignedAs[TypeParameterRole](
		decoder, "type parameter role",
	)
	if err != nil {
		return storedTypeParameter{}, err
	}
	ordinal, err := readSignedInt(decoder, "type parameter ordinal")
	if err != nil {
		return storedTypeParameter{}, err
	}
	constraint, err := readIdentityReference[typeRef](
		decoder, "type parameter constraint",
	)
	return storedTypeParameter{
		declaration: declaration, definition: definition,
		role: role, ordinal: ordinal, constraint: constraint,
	}, err
}

func readBinaryArrayType(
	decoder *binaryShardDecoder,
) (storedArrayType, error) {
	element, err := readIdentityReference[typeRef](
		decoder, "array element",
	)
	if err != nil {
		return storedArrayType{}, err
	}
	length, err := decoder.signed("array length")
	return storedArrayType{element: element, length: length}, err
}

func readBinaryMapType(
	decoder *binaryShardDecoder,
) (storedMapType, error) {
	key, err := readIdentityReference[typeRef](decoder, "map key")
	if err != nil {
		return storedMapType{}, err
	}
	element, err := readIdentityReference[typeRef](
		decoder, "map element",
	)
	return storedMapType{key: key, element: element}, err
}

func readBinaryChannelType(
	decoder *binaryShardDecoder,
) (storedChannelType, error) {
	element, err := readIdentityReference[typeRef](
		decoder, "channel element",
	)
	if err != nil {
		return storedChannelType{}, err
	}
	direction, err := readUnsignedAs[ChannelDirection](
		decoder, "channel direction",
	)
	return storedChannelType{
		element: element, direction: direction,
	}, err
}

func readBinarySignature(
	decoder *binaryShardDecoder,
	store *packageTypeStore,
) (storedSignature, error) {
	receiver, err := readIdentityReference[typeRef](
		decoder, "signature receiver",
	)
	if err != nil {
		return storedSignature{}, err
	}
	receiverParameters, err := readBinaryTypeRelation(decoder, store)
	if err != nil {
		return storedSignature{}, err
	}
	typeParameters, err := readBinaryTypeRelation(decoder, store)
	if err != nil {
		return storedSignature{}, err
	}
	parameters, err := readBinaryTypeRelation(decoder, store)
	if err != nil {
		return storedSignature{}, err
	}
	results, err := readBinaryTypeRelation(decoder, store)
	if err != nil {
		return storedSignature{}, err
	}
	variadic, err := decoder.boolean("signature variadic")
	return storedSignature{
		receiver:               receiver,
		receiverTypeParameters: receiverParameters,
		typeParameters:         typeParameters,
		parameters:             parameters,
		results:                results,
		variadic:               variadic,
	}, err
}

func readBinaryInterface(
	decoder *binaryShardDecoder,
	store *packageTypeStore,
) (storedInterfaceType, error) {
	methods, err := readBinaryMethods(decoder, store)
	if err != nil {
		return storedInterfaceType{}, err
	}
	embeddeds, err := readBinaryTypeRelation(decoder, store)
	if err != nil {
		return storedInterfaceType{}, err
	}
	terms, err := readBinaryTerms(decoder, store)
	if err != nil {
		return storedInterfaceType{}, err
	}
	typeSet, err := readUnsignedAs[TypeSetKind](
		decoder, "interface type set",
	)
	if err != nil {
		return storedInterfaceType{}, err
	}
	comparable, err := decoder.boolean("interface comparable")
	return storedInterfaceType{
		methods: methods, embeddeds: embeddeds, terms: terms,
		typeSet: typeSet, comparable: comparable,
	}, err
}

func readBinaryTypeRelation(
	decoder *binaryShardDecoder,
	store *packageTypeStore,
) (typeRefRange, error) {
	start, count, err := readReferenceRange(
		decoder,
		"type relation",
		&store.typeRelations,
	)
	return typeRefRange{start: start, count: count}, err
}

func readBinaryFields(
	decoder *binaryShardDecoder,
	store *packageTypeStore,
) (typeFieldRange, error) {
	count, err := decoder.count("type fields")
	if err != nil {
		return typeFieldRange{}, err
	}
	start := uint64(len(store.fields))
	for index := 0; index < count; index++ {
		field, readErr := readBinaryField(decoder)
		if readErr != nil {
			return typeFieldRange{}, readErr
		}
		store.fields = append(store.fields, field)
	}
	return typeFieldRange{start: start, count: uint64(count)}, nil
}

func readBinaryField(
	decoder *binaryShardDecoder,
) (storedTypeField, error) {
	name, err := decoder.text("type field name")
	if err != nil {
		return storedTypeField{}, err
	}
	pkg, err := readIdentityReference[packageRef](
		decoder, "type field package",
	)
	if err != nil {
		return storedTypeField{}, err
	}
	typeID, err := readIdentityReference[typeRef](
		decoder, "type field type",
	)
	if err != nil {
		return storedTypeField{}, err
	}
	embedded, err := decoder.boolean("type field embedded")
	if err != nil {
		return storedTypeField{}, err
	}
	tag, err := decoder.text("type field tag")
	if err != nil {
		return storedTypeField{}, err
	}
	ordinal, err := readSignedInt(decoder, "type field ordinal")
	return storedTypeField{
		name: name, pkg: pkg, typeID: typeID,
		embedded: embedded, tag: tag, ordinal: ordinal,
	}, err
}

func readBinaryMethods(
	decoder *binaryShardDecoder,
	store *packageTypeStore,
) (typeMethodRange, error) {
	count, err := decoder.count("type methods")
	if err != nil {
		return typeMethodRange{}, err
	}
	start := uint64(len(store.methods))
	for index := 0; index < count; index++ {
		name, readErr := decoder.text("type method name")
		if readErr != nil {
			return typeMethodRange{}, readErr
		}
		pkg, readErr := readIdentityReference[packageRef](
			decoder, "type method package",
		)
		if readErr != nil {
			return typeMethodRange{}, readErr
		}
		signature, readErr := readIdentityReference[typeRef](
			decoder, "type method signature",
		)
		if readErr != nil {
			return typeMethodRange{}, readErr
		}
		ordinal, readErr := readSignedInt(
			decoder, "type method ordinal",
		)
		if readErr != nil {
			return typeMethodRange{}, readErr
		}
		store.methods = append(store.methods, storedTypeMethod{
			name: name, pkg: pkg, signature: signature,
			ordinal: ordinal,
		})
	}
	return typeMethodRange{start: start, count: uint64(count)}, nil
}

func readBinaryTerms(
	decoder *binaryShardDecoder,
	store *packageTypeStore,
) (typeTermRange, error) {
	count, err := decoder.count("type terms")
	if err != nil {
		return typeTermRange{}, err
	}
	start := uint64(len(store.terms))
	for index := 0; index < count; index++ {
		tilde, readErr := decoder.boolean("type term tilde")
		if readErr != nil {
			return typeTermRange{}, readErr
		}
		typeID, readErr := readIdentityReference[typeRef](
			decoder, "type term type",
		)
		if readErr != nil {
			return typeTermRange{}, readErr
		}
		store.terms = append(store.terms, storedTypeTerm{
			tilde: tilde, typeID: typeID,
		})
	}
	return typeTermRange{start: start, count: uint64(count)}, nil
}
