package semantic

import "fmt"

func writeBinaryOperations(
	encoder *binaryShardEncoder,
	pkg Package,
	measurement *semanticShardMeasurement,
) {
	store := pkg.operations
	encoder.count(len(store.records))
	identities := newPackageIdentityProjection(pkg.identities)
	for _, record := range store.records {
		start := encoder.written
		encoder.unsigned(uint64(record.id))
		encoder.unsigned(uint64(record.kind))
		encoder.unsigned(uint64(record.syntax))
		encoder.unsigned(uint64(record.variant))
		encoder.unsigned(uint64(record.role))
		encoder.unsigned(uint64(record.token))
		encoder.unsigned(uint64(record.mode))
		encoder.unsigned(uint64(record.arity))
		encoder.unsigned(uint64(record.place))
		encoder.unsigned(uint64(record.resultType))
		encoder.unsigned(uint64(record.expectedType))
		encoder.boolean(record.addressable)
		encoder.boolean(record.assignable)
		encoder.boolean(record.hasOk)
		writeBinaryOperationConstant(encoder, store, record.constant)
		writeBinaryOperationObject(encoder, store, record.object)
		writeBinaryOperationSelection(encoder, store, record.selection)
		writeBinaryOperationInstance(encoder, store, record.instance)
		writeReferenceRange(
			encoder,
			store.operands,
			record.operands.start,
			record.operands.count,
		)
		writeReferenceRange(
			encoder,
			store.definitions,
			record.definitions.start,
			record.definitions.count,
		)
		writeBinaryImplicitOperations(encoder, store, record.implicit)
		encoder.unsigned(uint64(record.controlTarget))
		encoder.unsigned(uint64(record.label))
		measurement.consider(
			&measurement.operationTail,
			identities.operation(record.id).String(),
			encoder.written-start,
		)
	}
}

func writeBinaryOperationConstant(
	encoder *binaryShardEncoder,
	store packageOperationStore,
	reference constantRef,
) {
	encoder.boolean(reference != 0)
	if reference == 0 {
		return
	}
	payload := store.constants[reference-1]
	encoder.unsigned(uint64(payload.kind))
	encoder.text(payload.exact)
}

func writeBinaryOperationObject(
	encoder *binaryShardEncoder,
	store packageOperationStore,
	reference objectReferenceRef,
) {
	encoder.boolean(reference != 0)
	if reference == 0 {
		return
	}
	writeBinaryStoredObject(encoder, store.objects[reference-1])
}

func writeBinaryStoredObject(
	encoder *binaryShardEncoder,
	object storedObjectReference,
) {
	encoder.unsigned(uint64(object.kind))
	encoder.unsigned(uint64(object.declaration))
	encoder.unsigned(uint64(object.binding))
}

func writeBinaryOperationSelection(
	encoder *binaryShardEncoder,
	store packageOperationStore,
	reference selectionRef,
) {
	encoder.boolean(reference != 0)
	if reference == 0 {
		return
	}
	selection := store.selections[reference-1]
	encoder.unsigned(uint64(selection.kind))
	encoder.unsigned(uint64(selection.receiver))
	encoder.unsigned(uint64(selection.object))
	if selection.index.start > uint64(len(store.selectionIndexes)) ||
		selection.index.count >
			uint64(len(store.selectionIndexes))-selection.index.start {
		if encoder.err == nil {
			encoder.err = fmt.Errorf(
				"semantic binary selection index range is invalid",
			)
		}
		return
	}
	encoder.unsigned(selection.index.count)
	values := store.selectionIndexes[selection.index.start : selection.index.start+selection.index.count]
	for _, index := range values {
		encoder.signed(int64(index))
	}
	encoder.boolean(selection.indirect)
}

func writeBinaryOperationInstance(
	encoder *binaryShardEncoder,
	store packageOperationStore,
	reference instanceRef,
) {
	encoder.boolean(reference != 0)
	if reference == 0 {
		return
	}
	instance := store.instances[reference-1]
	if instance.target == 0 {
		if encoder.err == nil {
			encoder.err = fmt.Errorf(
				"semantic binary instance target is absent",
			)
		}
		return
	}
	writeBinaryStoredObject(
		encoder, store.objects[instance.target-1],
	)
	writeReferenceRange(
		encoder,
		store.instanceTypes,
		instance.types.start,
		instance.types.count,
	)
	encoder.unsigned(uint64(instance.signature))
}

func writeBinaryImplicitOperations(
	encoder *binaryShardEncoder,
	store packageOperationStore,
	relation implicitOperationRange,
) {
	if relation.start > uint64(len(store.implicit)) ||
		relation.count > uint64(len(store.implicit))-relation.start {
		if encoder.err == nil {
			encoder.err = fmt.Errorf(
				"semantic binary implicit-operation range is invalid",
			)
		}
		return
	}
	encoder.unsigned(relation.count)
	values := store.implicit[relation.start : relation.start+relation.count]
	for _, implicit := range values {
		encoder.unsigned(uint64(implicit.kind))
		encoder.unsigned(uint64(implicit.site))
		encoder.signed(int64(implicit.ordinal))
		encoder.unsigned(uint64(implicit.source))
		encoder.unsigned(uint64(implicit.target))
	}
}
