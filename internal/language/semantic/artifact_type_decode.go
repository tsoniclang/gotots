package semantic

import "fmt"

type wireTypeDecoder struct {
	identities wireIdentityDecoder
	references uint64
	fields     uint64
	methods    uint64
	terms      uint64
}

func (decoder *wireTypeDecoder) record(
	encoded wireTypeRecord,
) (Type, error) {
	id, err := decoder.identities.typeID(encoded.ID)
	if err != nil {
		return Type{}, err
	}
	kind := TypeKind(encoded.Kind)
	payload := encoded.Payload
	if err := requireSinglePayload(
		"type",
		payload.Tag,
		encoded.Kind,
		payload.Basic != nil,
		payload.Nominal != nil,
		payload.Parameter != nil,
		payload.Element != nil,
		payload.Array != nil,
		payload.Map != nil,
		payload.Channel != nil,
		payload.Signature != nil,
		payload.Struct != nil,
		payload.Interface != nil,
		payload.Tuple != nil,
		payload.Union != nil,
	); err != nil {
		return Type{}, err
	}
	spec := TypeSpec{Kind: kind}
	switch kind {
	case TypeBasic:
		if payload.Basic == nil {
			return Type{}, wrongPayload("type", kind)
		}
		spec.Basic = BasicKind(payload.Basic.Kind)
	case TypeNamed, TypeAlias:
		if payload.Nominal == nil {
			return Type{}, wrongPayload("type", kind)
		}
		if err := decoder.decodeNominal(
			&spec, *payload.Nominal,
		); err != nil {
			return Type{}, err
		}
	case TypeParameter:
		if payload.Parameter == nil {
			return Type{}, wrongPayload("type", kind)
		}
		if err := decoder.decodeParameter(
			&spec, *payload.Parameter,
		); err != nil {
			return Type{}, err
		}
	case TypePointer, TypeSlice:
		if payload.Element == nil {
			return Type{}, wrongPayload("type", kind)
		}
		spec.Element, err = decoder.identities.typeID(
			payload.Element.Element,
		)
	case TypeArray:
		if payload.Array == nil {
			return Type{}, wrongPayload("type", kind)
		}
		spec.Element, err = decoder.identities.typeID(
			payload.Array.Element,
		)
		spec.Length = payload.Array.Length
	case TypeMap:
		if payload.Map == nil {
			return Type{}, wrongPayload("type", kind)
		}
		spec.Key, err = decoder.identities.typeID(
			payload.Map.Key,
		)
		if err == nil {
			spec.Element, err = decoder.identities.typeID(
				payload.Map.Element,
			)
		}
	case TypeChannel:
		if payload.Channel == nil {
			return Type{}, wrongPayload("type", kind)
		}
		spec.Element, err = decoder.identities.typeID(
			payload.Channel.Element,
		)
		spec.Direction = ChannelDirection(
			payload.Channel.Direction,
		)
	case TypeSignature:
		if payload.Signature == nil {
			return Type{}, wrongPayload("type", kind)
		}
		spec.Signature, err = decoder.decodeSignature(
			*payload.Signature,
		)
	case TypeStruct:
		if payload.Struct == nil {
			return Type{}, wrongPayload("type", kind)
		}
		spec.Fields, err = decoder.fieldRange(
			payload.Struct.Fields,
		)
	case TypeInterface:
		if payload.Interface == nil {
			return Type{}, wrongPayload("type", kind)
		}
		if err := decoder.decodeInterface(
			&spec, *payload.Interface,
		); err != nil {
			return Type{}, err
		}
	case TypeTuple:
		if payload.Tuple == nil {
			return Type{}, wrongPayload("type", kind)
		}
		spec.Elements, err = decoder.referenceRange(
			"type tuple elements",
			payload.Tuple.Elements,
		)
	case TypeUnion:
		if payload.Union == nil {
			return Type{}, wrongPayload("type", kind)
		}
		spec.Terms, err = decoder.termRange(
			payload.Union.Terms,
		)
	default:
		return Type{}, fmt.Errorf(
			"semantic wire type kind %d is invalid", encoded.Kind,
		)
	}
	if err != nil {
		return Type{}, err
	}
	record, err := NewType(spec)
	if err != nil {
		return Type{}, err
	}
	if record.ID() != id {
		return Type{}, fmt.Errorf(
			"semantic wire type payload disagrees with identity %s",
			id,
		)
	}
	return record, nil
}

func (decoder *wireTypeDecoder) decodeNominal(
	spec *TypeSpec,
	encoded wireNominalType,
) error {
	var err error
	spec.Declaration, err = decoder.identities.declaration(
		encoded.Declaration,
	)
	if err != nil {
		return err
	}
	spec.Arguments, err = decoder.referenceRange(
		"nominal type arguments",
		encoded.Arguments,
	)
	if err != nil {
		return err
	}
	target, err := decoder.identities.typeID(encoded.Target)
	if err != nil {
		return err
	}
	spec.Methods, err = decoder.methodRange(encoded.Methods)
	if err != nil {
		return err
	}
	if spec.Kind == TypeNamed {
		spec.Underlying = target
	} else {
		spec.Target = target
	}
	return nil
}

func (decoder *wireTypeDecoder) decodeParameter(
	spec *TypeSpec,
	encoded wireTypeParameter,
) error {
	declaration, err := decoder.identities.declaration(
		encoded.Declaration,
	)
	if err != nil {
		return err
	}
	definition, err := decoder.identities.definition(
		encoded.Definition,
	)
	if err != nil {
		return err
	}
	spec.Parameter, err = NewTypeParameterOwner(
		declaration,
		definition,
		TypeParameterRole(encoded.Role),
		encoded.Ordinal,
	)
	if err != nil {
		return err
	}
	spec.Constraint, err = decoder.identities.typeID(
		encoded.Constraint,
	)
	return err
}

func (decoder *wireTypeDecoder) decodeSignature(
	encoded wireSignatureType,
) (Signature, error) {
	receiver, err := decoder.identities.typeID(encoded.Receiver)
	if err != nil {
		return Signature{}, err
	}
	receiverParameters, err := decoder.referenceRange(
		"signature receiver type parameters",
		encoded.ReceiverTypeParameters,
	)
	if err != nil {
		return Signature{}, err
	}
	typeParameters, err := decoder.referenceRange(
		"signature type parameters",
		encoded.TypeParameters,
	)
	if err != nil {
		return Signature{}, err
	}
	parameters, err := decoder.referenceRange(
		"signature parameters",
		encoded.Parameters,
	)
	if err != nil {
		return Signature{}, err
	}
	results, err := decoder.referenceRange(
		"signature results",
		encoded.Results,
	)
	if err != nil {
		return Signature{}, err
	}
	return Signature{
		Receiver:               receiver,
		ReceiverTypeParameters: receiverParameters,
		TypeParameters:         typeParameters,
		Parameters:             parameters,
		Results:                results,
		Variadic:               encoded.Variadic,
	}, nil
}

func (decoder *wireTypeDecoder) decodeInterface(
	spec *TypeSpec,
	encoded wireInterfaceType,
) error {
	var err error
	spec.Methods, err = decoder.methodRange(encoded.Methods)
	if err != nil {
		return err
	}
	spec.Embeddeds, err = decoder.referenceRange(
		"interface embedded types",
		encoded.Embeddeds,
	)
	if err != nil {
		return err
	}
	spec.Terms, err = decoder.termRange(encoded.Terms)
	if err != nil {
		return err
	}
	spec.TypeSet = TypeSetKind(encoded.TypeSet)
	spec.Comparable = encoded.Comparable
	return nil
}
