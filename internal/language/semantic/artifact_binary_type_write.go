package semantic

import "fmt"

func writeBinaryTypes(
	encoder *binaryShardEncoder,
	pkg Package,
	measurement *semanticShardMeasurement,
) {
	store := pkg.types
	encoder.count(len(store.records))
	for _, record := range store.records {
		start := encoder.written
		encoder.unsigned(uint64(record.id))
		encoder.unsigned(uint64(record.kind))
		writeBinaryTypePayload(encoder, store, record)
		measurement.considerType(
			record.id,
			encoder.written-start,
		)
	}
}

func writeBinaryTypePayload(
	encoder *binaryShardEncoder,
	store packageTypeStore,
	record storedType,
) {
	switch record.kind {
	case TypeBasic:
		encoder.unsigned(uint64(store.basic[record.payload-1]))
	case TypeNamed, TypeAlias:
		writeBinaryNominalType(
			encoder, store, store.nominal[record.payload-1],
		)
	case TypeParameter:
		payload := store.parameters[record.payload-1]
		encoder.unsigned(uint64(payload.declaration))
		encoder.unsigned(uint64(payload.definition))
		encoder.unsigned(uint64(payload.role))
		encoder.signed(int64(payload.ordinal))
		encoder.unsigned(uint64(payload.constraint))
	case TypePointer, TypeSlice:
		encoder.unsigned(uint64(store.elements[record.payload-1]))
	case TypeArray:
		payload := store.arrays[record.payload-1]
		encoder.unsigned(uint64(payload.element))
		encoder.signed(payload.length)
	case TypeMap:
		payload := store.maps[record.payload-1]
		encoder.unsigned(uint64(payload.key))
		encoder.unsigned(uint64(payload.element))
	case TypeChannel:
		payload := store.channels[record.payload-1]
		encoder.unsigned(uint64(payload.element))
		encoder.unsigned(uint64(payload.direction))
	case TypeSignature:
		writeBinarySignature(
			encoder, store, store.signatures[record.payload-1],
		)
	case TypeStruct:
		writeBinaryFields(
			encoder, store, store.structs[record.payload-1],
		)
	case TypeInterface:
		writeBinaryInterface(
			encoder, store, store.interfaces[record.payload-1],
		)
	case TypeTuple:
		payload := store.tuples[record.payload-1]
		writeReferenceRange(
			encoder,
			store.typeRelations,
			payload.start,
			payload.count,
		)
	case TypeUnion:
		writeBinaryTerms(
			encoder, store, store.unions[record.payload-1],
		)
	default:
		if encoder.err == nil {
			encoder.err = fmt.Errorf(
				"semantic binary type kind %d is invalid",
				record.kind,
			)
		}
	}
}

func writeBinaryNominalType(
	encoder *binaryShardEncoder,
	store packageTypeStore,
	payload storedNominalType,
) {
	encoder.unsigned(uint64(payload.declaration))
	writeReferenceRange(
		encoder,
		store.typeRelations,
		payload.arguments.start,
		payload.arguments.count,
	)
	encoder.unsigned(uint64(payload.target))
	writeBinaryMethods(encoder, store, payload.methods)
}

func writeBinarySignature(
	encoder *binaryShardEncoder,
	store packageTypeStore,
	payload storedSignature,
) {
	encoder.unsigned(uint64(payload.receiver))
	ranges := [...]typeRefRange{
		payload.receiverTypeParameters,
		payload.typeParameters,
		payload.parameters,
		payload.results,
	}
	for _, relation := range ranges {
		writeReferenceRange(
			encoder,
			store.typeRelations,
			relation.start,
			relation.count,
		)
	}
	encoder.boolean(payload.variadic)
}

func writeBinaryFields(
	encoder *binaryShardEncoder,
	store packageTypeStore,
	relation typeFieldRange,
) {
	if relation.start > uint64(len(store.fields)) ||
		relation.count > uint64(len(store.fields))-relation.start {
		if encoder.err == nil {
			encoder.err = fmt.Errorf(
				"semantic binary type-field range is invalid",
			)
		}
		return
	}
	encoder.unsigned(relation.count)
	values := store.fields[relation.start : relation.start+relation.count]
	for _, field := range values {
		encoder.text(field.name)
		encoder.unsigned(uint64(field.pkg))
		encoder.unsigned(uint64(field.typeID))
		encoder.boolean(field.embedded)
		encoder.text(field.tag)
		encoder.signed(int64(field.ordinal))
	}
}

func writeBinaryMethods(
	encoder *binaryShardEncoder,
	store packageTypeStore,
	relation typeMethodRange,
) {
	if relation.start > uint64(len(store.methods)) ||
		relation.count > uint64(len(store.methods))-relation.start {
		if encoder.err == nil {
			encoder.err = fmt.Errorf(
				"semantic binary type-method range is invalid",
			)
		}
		return
	}
	encoder.unsigned(relation.count)
	values := store.methods[relation.start : relation.start+relation.count]
	for _, method := range values {
		encoder.text(method.name)
		encoder.unsigned(uint64(method.pkg))
		encoder.unsigned(uint64(method.signature))
		encoder.signed(int64(method.ordinal))
	}
}

func writeBinaryTerms(
	encoder *binaryShardEncoder,
	store packageTypeStore,
	relation typeTermRange,
) {
	if relation.start > uint64(len(store.terms)) ||
		relation.count > uint64(len(store.terms))-relation.start {
		if encoder.err == nil {
			encoder.err = fmt.Errorf(
				"semantic binary type-term range is invalid",
			)
		}
		return
	}
	encoder.unsigned(relation.count)
	values := store.terms[relation.start : relation.start+relation.count]
	for _, term := range values {
		encoder.boolean(term.tilde)
		encoder.unsigned(uint64(term.typeID))
	}
}

func writeBinaryInterface(
	encoder *binaryShardEncoder,
	store packageTypeStore,
	payload storedInterfaceType,
) {
	writeBinaryMethods(encoder, store, payload.methods)
	writeReferenceRange(
		encoder,
		store.typeRelations,
		payload.embeddeds.start,
		payload.embeddeds.count,
	)
	writeBinaryTerms(encoder, store, payload.terms)
	encoder.unsigned(uint64(payload.typeSet))
	encoder.boolean(payload.comparable)
}
