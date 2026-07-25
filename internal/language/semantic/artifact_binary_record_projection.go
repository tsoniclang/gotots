package semantic

func decodeBinaryDefinitionValue(
	decoder *binaryShardDecoder,
	identities packageIdentityTable,
	authority Authority,
) (DefinitionSemantics, error) {
	var store packageDefinitionStore
	if err := readBinaryDefinitionRecord(decoder, &store, 1); err != nil {
		return DefinitionSemantics{}, err
	}
	return store.record(
		newPackageIdentityProjection(identities),
		packageAuthorityTable{records: []Authority{authority}},
		0,
	)
}

func decodeBinaryResolutionValue(
	decoder *binaryShardDecoder,
	identities packageIdentityTable,
) (OccurrenceResolution, error) {
	var store packageResolutionStore
	if err := readBinaryResolutionRecord(decoder, &store); err != nil {
		return OccurrenceResolution{}, err
	}
	return store.record(
		newPackageIdentityProjection(identities),
		0,
	)
}

func decodeBinaryDeclarationValue(
	decoder *binaryShardDecoder,
	identities packageIdentityTable,
	authority Authority,
) (Declaration, error) {
	record, err := readBinaryDeclarationRecord(decoder, 1)
	if err != nil {
		return Declaration{}, err
	}
	store := packageDeclarationStore{
		records: []storedDeclaration{record},
	}
	return store.record(
		newPackageIdentityProjection(identities),
		packageAuthorityTable{records: []Authority{authority}},
		0,
	)
}

func decodeBinaryBindingValue(
	decoder *binaryShardDecoder,
	identities packageIdentityTable,
	authority Authority,
) (Binding, error) {
	var store packageBindingStore
	if err := readBinaryBindingRecord(
		decoder, &store, 1,
	); err != nil {
		return Binding{}, err
	}
	return store.record(
		newPackageIdentityProjection(identities),
		packageAuthorityTable{records: []Authority{authority}},
		0,
	)
}

func decodeBinaryTypeValue(
	decoder *binaryShardDecoder,
	identities packageIdentityTable,
) (Type, error) {
	var store packageTypeStore
	if err := readBinaryTypeRecord(decoder, &store); err != nil {
		return Type{}, err
	}
	return store.record(
		newPackageIdentityProjection(identities),
		0,
	)
}

func decodeBinaryOperationValue(
	decoder *binaryShardDecoder,
	identities packageIdentityTable,
) (Operation, error) {
	var store packageOperationStore
	if err := readBinaryOperationRecord(decoder, &store); err != nil {
		return Operation{}, err
	}
	projection := newPackageOperationProjection(store, identities)
	return projection.operation(
		newPackageIdentityProjection(identities),
		0,
	)
}

func decodeBinaryUnsupportedValue(
	decoder *binaryShardDecoder,
	identities packageIdentityTable,
	authority Authority,
) (Unsupported, error) {
	id, err := readIdentityReference[unsupportedRef](
		decoder, "unsupported id",
	)
	if err != nil {
		return Unsupported{}, err
	}
	reason, err := readUnsignedAs[UnsupportedReason](
		decoder, "unsupported reason",
	)
	if err != nil {
		return Unsupported{}, err
	}
	evidence, err := decoder.text("unsupported evidence")
	if err != nil {
		return Unsupported{}, err
	}
	store := packageUnsupportedStore{
		records: []storedUnsupported{{
			id: id, reason: reason, evidence: evidence, authority: 1,
		}},
	}
	return store.record(
		newPackageIdentityProjection(identities),
		packageAuthorityTable{records: []Authority{authority}},
		0,
	)
}
