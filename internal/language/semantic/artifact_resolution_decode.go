package semantic

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/language/catalog"
)

type wireResolutionDecoder struct {
	identities wireIdentityDecoder
}

func (decoder wireResolutionDecoder) record(
	encoded wireResolutionRecord,
) (OccurrenceResolution, error) {
	occurrence, err := decoder.identities.occurrence(
		encoded.Occurrence,
	)
	if err != nil {
		return OccurrenceResolution{}, err
	}
	owner, err := decoder.identities.definition(encoded.Owner)
	if err != nil {
		return OccurrenceResolution{}, err
	}
	if err := requireSinglePayload(
		"resolution",
		encoded.Payload.Tag,
		encoded.Kind,
		encoded.Payload.Structural != nil,
		encoded.Payload.DefinitionComponent != nil,
		encoded.Payload.Declaration != nil,
		encoded.Payload.Binding != nil,
		encoded.Payload.Type != nil,
		encoded.Payload.Operation != nil,
		encoded.Payload.Unsupported != nil,
	); err != nil {
		return OccurrenceResolution{}, err
	}
	spec := ResolutionSpec{
		Occurrence: occurrence,
		Owner:      owner,
		Syntax:     catalog.Kind(encoded.Syntax),
		Role:       catalog.Role(encoded.Role),
		Variant:    catalog.Variant(encoded.Variant),
		Domain:     catalog.ResolutionDomain(encoded.Domain),
		Kind:       ResolutionKind(encoded.Kind),
	}
	switch spec.Kind {
	case ResolutionStructuralOnly:
		value := encoded.Payload.Structural
		if value == nil {
			return OccurrenceResolution{}, wrongPayload(
				"resolution", spec.Kind,
			)
		}
		declaration, decodeErr := decoder.identities.declaration(
			value.Declaration,
		)
		if decodeErr != nil {
			return OccurrenceResolution{}, decodeErr
		}
		typeID, decodeErr := decoder.identities.typeID(value.Type)
		if decodeErr != nil {
			return OccurrenceResolution{}, decodeErr
		}
		spec.Structural, err = NewStructuralEvidence(
			StructuralDisposition(value.Disposition),
			declaration,
			typeID,
		)
	case ResolutionDefinitionComponent:
		value := encoded.Payload.DefinitionComponent
		if value == nil {
			return OccurrenceResolution{}, wrongPayload(
				"resolution", spec.Kind,
			)
		}
		spec.Component = DefinitionComponentKind(value.Component)
		spec.Definition, err = decoder.identities.definition(
			value.Definition,
		)
	case ResolutionDeclaration:
		value := encoded.Payload.Declaration
		if value == nil {
			return OccurrenceResolution{}, wrongPayload(
				"resolution", spec.Kind,
			)
		}
		spec.Declaration, err = decoder.identities.declaration(
			value.Reference,
		)
	case ResolutionBinding:
		value := encoded.Payload.Binding
		if value == nil {
			return OccurrenceResolution{}, wrongPayload(
				"resolution", spec.Kind,
			)
		}
		spec.Binding, err = decoder.identities.binding(
			value.Reference,
		)
	case ResolutionType:
		value := encoded.Payload.Type
		if value == nil {
			return OccurrenceResolution{}, wrongPayload(
				"resolution", spec.Kind,
			)
		}
		spec.Type, err = decoder.identities.typeID(value.Reference)
	case ResolutionOperation:
		value := encoded.Payload.Operation
		if value == nil {
			return OccurrenceResolution{}, wrongPayload(
				"resolution", spec.Kind,
			)
		}
		spec.Operation, err = decoder.identities.operation(
			value.Reference,
		)
	case ResolutionUnsupported:
		value := encoded.Payload.Unsupported
		if value == nil {
			return OccurrenceResolution{}, wrongPayload(
				"resolution", spec.Kind,
			)
		}
		spec.Unsupported, err = decoder.identities.unsupportedID(
			value.Reference,
		)
	default:
		return OccurrenceResolution{}, fmt.Errorf(
			"semantic wire resolution kind %d is invalid",
			encoded.Kind,
		)
	}
	if err != nil {
		return OccurrenceResolution{}, err
	}
	return NewOccurrenceResolution(spec)
}
