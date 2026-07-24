package semantic

type wireObjectEncoder struct {
	identities wireIdentityEncoder
	captures   uint64
}

func (encoder wireObjectEncoder) constant(
	kind ConstantKind,
	exact string,
) *wireConstantValue {
	if kind == ConstantInvalid && exact == "" {
		return nil
	}
	return &wireConstantValue{Kind: uint8(kind), Exact: exact}
}

func (encoder wireObjectEncoder) declaration(
	store packageDeclarationStore,
	index int,
) (wireDeclarationRecord, error) {
	stored := store.records[index]
	id, err := encoder.identities.declaration(stored.id)
	if err != nil {
		return wireDeclarationRecord{}, err
	}
	pkg, err := encoder.identities.packageID(stored.pkg)
	if err != nil {
		return wireDeclarationRecord{}, err
	}
	typeID, err := encoder.identities.typeID(stored.typeID)
	if err != nil {
		return wireDeclarationRecord{}, err
	}
	return wireDeclarationRecord{
		ID:       id,
		Package:  pkg,
		Class:    uint8(stored.class),
		Name:     stored.name,
		Type:     typeID,
		Exported: stored.exported,
		Constant: encoder.constant(
			stored.constantKind, stored.constant,
		),
	}, nil
}

func (encoder *wireObjectEncoder) binding(
	store packageBindingStore,
	index int,
) (wireBindingRecord, error) {
	stored := store.records[index]
	id, err := encoder.identities.binding(stored.id)
	if err != nil {
		return wireBindingRecord{}, err
	}
	pkg, err := encoder.identities.packageID(stored.pkg)
	if err != nil {
		return wireBindingRecord{}, err
	}
	definition, err := encoder.identities.definition(stored.definition)
	if err != nil {
		return wireBindingRecord{}, err
	}
	typeID, err := encoder.identities.typeID(stored.typeID)
	if err != nil {
		return wireBindingRecord{}, err
	}
	source, err := encoder.identities.occurrence(stored.source)
	if err != nil {
		return wireBindingRecord{}, err
	}
	captures, err := encodeReferenceRange(
		store.captures,
		stored.captures.start,
		stored.captures.count,
		&encoder.captures,
		encoder.identities.definition,
	)
	if err != nil {
		return wireBindingRecord{}, err
	}
	return wireBindingRecord{
		ID:         id,
		Package:    pkg,
		Definition: definition,
		Role:       uint8(stored.role),
		Name:       stored.name,
		Type:       typeID,
		Source:     source,
		CapturedBy: captures,
	}, nil
}

func (encoder wireObjectEncoder) unsupported(
	store packageUnsupportedStore,
	index int,
) (wireUnsupportedRecord, error) {
	stored := store.records[index]
	id, err := encoder.identities.unsupported(stored.id)
	if err != nil {
		return wireUnsupportedRecord{}, err
	}
	return wireUnsupportedRecord{
		ID:       id,
		Reason:   uint8(stored.reason),
		Evidence: stored.evidence,
	}, nil
}
