package semantic

type wireResolutionEncoder struct {
	identities wireIdentityEncoder
	store      packageResolutionStore
}

func (encoder wireResolutionEncoder) record(
	index int,
) (wireResolutionRecord, error) {
	stored := encoder.store.records[index]
	occurrence, err := encoder.identities.occurrence(stored.occurrence)
	if err != nil {
		return wireResolutionRecord{}, err
	}
	owner, err := encoder.identities.definition(stored.owner)
	if err != nil {
		return wireResolutionRecord{}, err
	}
	payload, err := encoder.payload(stored)
	if err != nil {
		return wireResolutionRecord{}, err
	}
	return wireResolutionRecord{
		Occurrence: occurrence,
		Owner:      owner,
		Syntax:     uint16(stored.syntax),
		Role:       uint16(stored.role),
		Variant:    uint16(stored.variant),
		Domain:     uint8(stored.domain),
		Kind:       uint8(stored.kind),
		Payload:    payload,
	}, nil
}

func (encoder wireResolutionEncoder) payload(
	stored storedResolution,
) (wireResolutionPayload, error) {
	out := wireResolutionPayload{Tag: uint8(stored.kind)}
	switch stored.kind {
	case ResolutionStructuralOnly:
		value, err := payloadAt(
			encoder.store.structural, stored.payload,
		)
		if err != nil {
			return wireResolutionPayload{}, err
		}
		declaration, err := encoder.identities.declaration(
			value.declaration,
		)
		if err != nil {
			return wireResolutionPayload{}, err
		}
		typeID, err := encoder.identities.typeID(value.typeID)
		if err != nil {
			return wireResolutionPayload{}, err
		}
		out.Structural = &wireStructuralResolution{
			Disposition: uint8(value.disposition),
			Declaration: declaration,
			Type:        typeID,
		}
	case ResolutionDefinitionComponent:
		value, err := payloadAt(
			encoder.store.definitionComponents, stored.payload,
		)
		if err != nil {
			return wireResolutionPayload{}, err
		}
		definition, err := encoder.identities.definition(
			value.definition,
		)
		if err != nil {
			return wireResolutionPayload{}, err
		}
		out.DefinitionComponent = &wireDefinitionComponentResolution{
			Component:  uint8(value.component),
			Definition: definition,
		}
	case ResolutionDeclaration:
		value, err := payloadAt(
			encoder.store.declarations, stored.payload,
		)
		if err != nil {
			return wireResolutionPayload{}, err
		}
		reference, err := encoder.identities.declaration(value)
		if err != nil {
			return wireResolutionPayload{}, err
		}
		out.Declaration = &wireDeclarationReferencePayload{
			Reference: reference,
		}
	case ResolutionBinding:
		value, err := payloadAt(encoder.store.bindings, stored.payload)
		if err != nil {
			return wireResolutionPayload{}, err
		}
		reference, err := encoder.identities.binding(value)
		if err != nil {
			return wireResolutionPayload{}, err
		}
		out.Binding = &wireBindingReferencePayload{Reference: reference}
	case ResolutionType:
		value, err := payloadAt(encoder.store.types, stored.payload)
		if err != nil {
			return wireResolutionPayload{}, err
		}
		reference, err := encoder.identities.typeID(value)
		if err != nil {
			return wireResolutionPayload{}, err
		}
		out.Type = &wireTypeReferencePayload{Reference: reference}
	case ResolutionOperation:
		value, err := payloadAt(
			encoder.store.operations, stored.payload,
		)
		if err != nil {
			return wireResolutionPayload{}, err
		}
		reference, err := encoder.identities.operation(value)
		if err != nil {
			return wireResolutionPayload{}, err
		}
		out.Operation = &wireOperationReferencePayload{
			Reference: reference,
		}
	case ResolutionUnsupported:
		value, err := payloadAt(
			encoder.store.unsupported, stored.payload,
		)
		if err != nil {
			return wireResolutionPayload{}, err
		}
		reference, err := encoder.identities.unsupported(value)
		if err != nil {
			return wireResolutionPayload{}, err
		}
		out.Unsupported = &wireUnsupportedReferencePayload{
			Reference: reference,
		}
	default:
		return wireResolutionPayload{}, &artifactError{
			reason: "semantic resolution has invalid normalized payload",
		}
	}
	return out, nil
}
