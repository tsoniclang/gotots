package semantic

type wireOperationEncoder struct {
	identities       wireIdentityEncoder
	store            packageOperationStore
	operands         uint64
	definitions      uint64
	implicit         uint64
	selectionIndexes uint64
	instanceTypes    uint64
}

func (encoder *wireOperationEncoder) record(
	index int,
) (wireOperationRecord, error) {
	stored := encoder.store.records[index]
	id, err := encoder.identities.operation(stored.id)
	if err != nil {
		return wireOperationRecord{}, err
	}
	resultType, err := encoder.identities.typeID(stored.resultType)
	if err != nil {
		return wireOperationRecord{}, err
	}
	expectedType, err := encoder.identities.typeID(stored.expectedType)
	if err != nil {
		return wireOperationRecord{}, err
	}
	constant, err := encoder.constant(stored.constant)
	if err != nil {
		return wireOperationRecord{}, err
	}
	object, err := encoder.object(stored.object)
	if err != nil {
		return wireOperationRecord{}, err
	}
	selection, err := encoder.selection(stored.selection)
	if err != nil {
		return wireOperationRecord{}, err
	}
	instance, err := encoder.instance(stored.instance)
	if err != nil {
		return wireOperationRecord{}, err
	}
	operands, err := encodeReferenceRange(
		encoder.store.operands,
		stored.operands.start,
		stored.operands.count,
		&encoder.operands,
		encoder.identities.occurrence,
	)
	if err != nil {
		return wireOperationRecord{}, err
	}
	definitions, err := encodeReferenceRange(
		encoder.store.definitions,
		stored.definitions.start,
		stored.definitions.count,
		&encoder.definitions,
		encoder.identities.definition,
	)
	if err != nil {
		return wireOperationRecord{}, err
	}
	implicit, err := encoder.implicitRange(stored.implicit)
	if err != nil {
		return wireOperationRecord{}, err
	}
	controlTarget, err := encoder.identities.operation(
		stored.controlTarget,
	)
	if err != nil {
		return wireOperationRecord{}, err
	}
	label, err := encoder.identities.binding(stored.label)
	if err != nil {
		return wireOperationRecord{}, err
	}
	return wireOperationRecord{
		ID:            id,
		Kind:          uint16(stored.kind),
		Syntax:        uint16(stored.syntax),
		Variant:       uint16(stored.variant),
		Role:          uint16(stored.role),
		Token:         uint16(stored.token),
		Mode:          uint8(stored.mode),
		Arity:         uint8(stored.arity),
		Place:         uint8(stored.place),
		ResultType:    resultType,
		ExpectedType:  expectedType,
		Addressable:   stored.addressable,
		Assignable:    stored.assignable,
		HasOk:         stored.hasOk,
		Constant:      constant,
		Object:        object,
		Selection:     selection,
		Instance:      instance,
		Operands:      operands,
		Definitions:   definitions,
		Implicit:      implicit,
		ControlTarget: controlTarget,
		Label:         label,
	}, nil
}

func (encoder wireOperationEncoder) constant(
	reference constantRef,
) (*wireConstantValue, error) {
	if reference == 0 {
		return nil, nil
	}
	value, err := payloadAt(encoder.store.constants, uint64(reference))
	if err != nil {
		return nil, err
	}
	return &wireConstantValue{
		Kind: uint8(value.kind), Exact: value.exact,
	}, nil
}

func (encoder wireOperationEncoder) object(
	reference objectReferenceRef,
) (*wireObjectReference, error) {
	if reference == 0 {
		return nil, nil
	}
	value, err := payloadAt(encoder.store.objects, uint64(reference))
	if err != nil {
		return nil, err
	}
	declaration, err := encoder.identities.declaration(
		value.declaration,
	)
	if err != nil {
		return nil, err
	}
	binding, err := encoder.identities.binding(value.binding)
	if err != nil {
		return nil, err
	}
	return &wireObjectReference{
		Kind:        uint8(value.kind),
		Declaration: declaration,
		Binding:     binding,
	}, nil
}

func (encoder *wireOperationEncoder) selection(
	reference selectionRef,
) (*wireSelection, error) {
	if reference == 0 {
		return nil, nil
	}
	value, err := payloadAt(
		encoder.store.selections, uint64(reference),
	)
	if err != nil {
		return nil, err
	}
	receiver, err := encoder.identities.typeID(value.receiver)
	if err != nil {
		return nil, err
	}
	object, err := encoder.identities.declaration(value.object)
	if err != nil {
		return nil, err
	}
	indexes, err := encodeIntegerRange(
		encoder.store.selectionIndexes,
		value.index.start,
		value.index.count,
		&encoder.selectionIndexes,
	)
	if err != nil {
		return nil, err
	}
	return &wireSelection{
		Kind:     uint8(value.kind),
		Receiver: receiver,
		Object:   object,
		Index:    indexes,
		Indirect: value.indirect,
	}, nil
}

func (encoder *wireOperationEncoder) instance(
	reference instanceRef,
) (*wireInstance, error) {
	if reference == 0 {
		return nil, nil
	}
	value, err := payloadAt(
		encoder.store.instances, uint64(reference),
	)
	if err != nil {
		return nil, err
	}
	target, err := encoder.object(value.target)
	if err != nil {
		return nil, err
	}
	if target == nil {
		return nil, &artifactError{
			reason: "semantic instance has no normalized target",
		}
	}
	types, err := encodeReferenceRange(
		encoder.store.instanceTypes,
		value.types.start,
		value.types.count,
		&encoder.instanceTypes,
		encoder.identities.typeID,
	)
	if err != nil {
		return nil, err
	}
	signature, err := encoder.identities.typeID(value.signature)
	if err != nil {
		return nil, err
	}
	return &wireInstance{
		Target:    *target,
		Types:     types,
		Signature: signature,
	}, nil
}

func (encoder *wireOperationEncoder) implicitRange(
	reference implicitOperationRange,
) (wireImplicitOperationRange, error) {
	values, err := relationSlice(
		encoder.store.implicit,
		reference.start,
		reference.count,
	)
	if err != nil {
		return wireImplicitOperationRange{}, err
	}
	out := wireImplicitOperationRange{
		Start:  encoder.implicit,
		Count:  reference.count,
		Values: make([]wireImplicitOperation, 0, len(values)),
	}
	for _, value := range values {
		site, err := encoder.identities.occurrence(value.site)
		if err != nil {
			return wireImplicitOperationRange{}, err
		}
		source, err := encoder.identities.typeID(value.source)
		if err != nil {
			return wireImplicitOperationRange{}, err
		}
		target, err := encoder.identities.typeID(value.target)
		if err != nil {
			return wireImplicitOperationRange{}, err
		}
		out.Values = append(out.Values, wireImplicitOperation{
			Kind:    uint8(value.kind),
			Site:    site,
			Ordinal: value.ordinal,
			Source:  source,
			Target:  target,
		})
	}
	encoder.implicit += reference.count
	return out, nil
}
