package semantic

type wireDefinitionEncoder struct {
	identities   wireIdentityEncoder
	store        packageDefinitionStore
	declarations uint64
	bindings     uint64
	initializers uint64
}

func (encoder *wireDefinitionEncoder) record(
	index int,
) (wireDefinitionRecord, error) {
	stored := encoder.store.records[index]
	id, err := encoder.identities.definition(stored.id)
	if err != nil {
		return wireDefinitionRecord{}, err
	}
	pkg, err := encoder.identities.packageID(stored.pkg)
	if err != nil {
		return wireDefinitionRecord{}, err
	}
	bindings, err := encodeReferenceRange(
		encoder.store.bindingRelations,
		stored.bindings.start,
		stored.bindings.count,
		&encoder.bindings,
		encoder.identities.binding,
	)
	if err != nil {
		return wireDefinitionRecord{}, err
	}
	payload, err := encoder.payload(stored)
	if err != nil {
		return wireDefinitionRecord{}, err
	}
	return wireDefinitionRecord{
		ID:       id,
		Package:  pkg,
		Form:     uint8(stored.form),
		Name:     stored.name,
		Bindings: bindings,
		Payload:  payload,
	}, nil
}

func (encoder *wireDefinitionEncoder) payload(
	stored storedDefinition,
) (wireDefinitionPayload, error) {
	out := wireDefinitionPayload{Tag: uint8(stored.form)}
	switch stored.form {
	case DefinitionFormCallable:
		value, err := payloadAt(encoder.store.callable, stored.payload)
		if err != nil {
			return wireDefinitionPayload{}, err
		}
		declarations, err := encodeReferenceRange(
			encoder.store.declarationRelations,
			value.declarations.start,
			value.declarations.count,
			&encoder.declarations,
			encoder.identities.declaration,
		)
		if err != nil {
			return wireDefinitionPayload{}, err
		}
		signature, err := encoder.identities.typeID(value.signature)
		if err != nil {
			return wireDefinitionPayload{}, err
		}
		receiver, err := encoder.identities.binding(value.receiver)
		if err != nil {
			return wireDefinitionPayload{}, err
		}
		out.Callable = &wireCallableDefinition{
			Declarations: declarations,
			Signature:    signature,
			Receiver:     receiver,
		}
	case DefinitionFormInitializer:
		value, err := payloadAt(encoder.store.initializers, stored.payload)
		if err != nil {
			return wireDefinitionPayload{}, err
		}
		declarations, err := encodeReferenceRange(
			encoder.store.declarationRelations,
			value.declarations.start,
			value.declarations.count,
			&encoder.declarations,
			encoder.identities.declaration,
		)
		if err != nil {
			return wireDefinitionPayload{}, err
		}
		entries, err := encodeReferenceRange(
			encoder.store.initializerEntries,
			value.entries.start,
			value.entries.count,
			&encoder.initializers,
			encoder.identities.occurrence,
		)
		if err != nil {
			return wireDefinitionPayload{}, err
		}
		out.Initializer = &wireInitializerDefinition{
			Declarations: declarations,
			Entries:      entries,
		}
	case DefinitionFormBodyless:
		value, err := payloadAt(encoder.store.bodyless, stored.payload)
		if err != nil {
			return wireDefinitionPayload{}, err
		}
		declaration, err := encoder.identities.declaration(
			value.declaration,
		)
		if err != nil {
			return wireDefinitionPayload{}, err
		}
		signature, err := encoder.identities.typeID(value.signature)
		if err != nil {
			return wireDefinitionPayload{}, err
		}
		receiver, err := encoder.identities.binding(value.receiver)
		if err != nil {
			return wireDefinitionPayload{}, err
		}
		out.Bodyless = &wireBodylessDefinition{
			Declaration: declaration,
			Signature:   signature,
			Receiver:    receiver,
		}
	case DefinitionFormImplicit:
		value, err := payloadAt(encoder.store.implicit, stored.payload)
		if err != nil {
			return wireDefinitionPayload{}, err
		}
		out.Implicit = &wireImplicitDefinition{
			Operation: uint8(value.operation),
		}
	case DefinitionFormSynthetic:
		value, err := payloadAt(encoder.store.synthetic, stored.payload)
		if err != nil {
			return wireDefinitionPayload{}, err
		}
		declaration, err := encoder.identities.declaration(
			value.declaration,
		)
		if err != nil {
			return wireDefinitionPayload{}, err
		}
		signature, err := encoder.identities.typeID(value.signature)
		if err != nil {
			return wireDefinitionPayload{}, err
		}
		out.Synthetic = &wireSyntheticDefinition{
			Declaration: declaration,
			Signature:   signature,
		}
	default:
		return wireDefinitionPayload{}, &artifactError{
			reason: "semantic definition has invalid normalized payload",
		}
	}
	return out, nil
}
