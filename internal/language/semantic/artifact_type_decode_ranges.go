package semantic

import "github.com/tsoniclang/gotots/internal/identity"

func (decoder *wireTypeDecoder) referenceRange(
	name string,
	encoded wireReferenceRange[wireTypeReference],
) ([]identity.SemanticTypeID, error) {
	return decodeReferenceRange(
		name,
		encoded,
		&decoder.references,
		decoder.identities.typeID,
	)
}

func (decoder *wireTypeDecoder) fieldRange(
	encoded wireTypeFieldRange,
) ([]TypeField, error) {
	if encoded.Start != decoder.fields ||
		encoded.Count != uint64(len(encoded.Values)) {
		return nil, wrongWireRange(
			"type fields",
			encoded.Start,
			encoded.Count,
			len(encoded.Values),
			decoder.fields,
		)
	}
	out := make([]TypeField, 0, len(encoded.Values))
	for _, value := range encoded.Values {
		pkg, err := decoder.identities.packageID(value.Package)
		if err != nil {
			return nil, err
		}
		typeID, err := decoder.identities.typeID(value.Type)
		if err != nil {
			return nil, err
		}
		out = append(out, TypeField{
			Name:     value.Name,
			Package:  pkg,
			Type:     typeID,
			Embedded: value.Embedded,
			Tag:      value.Tag,
			Ordinal:  value.Ordinal,
		})
	}
	decoder.fields += encoded.Count
	return out, nil
}

func (decoder *wireTypeDecoder) methodRange(
	encoded wireTypeMethodRange,
) ([]TypeMethod, error) {
	if encoded.Start != decoder.methods ||
		encoded.Count != uint64(len(encoded.Values)) {
		return nil, wrongWireRange(
			"type methods",
			encoded.Start,
			encoded.Count,
			len(encoded.Values),
			decoder.methods,
		)
	}
	out := make([]TypeMethod, 0, len(encoded.Values))
	for _, value := range encoded.Values {
		pkg, err := decoder.identities.packageID(value.Package)
		if err != nil {
			return nil, err
		}
		signature, err := decoder.identities.typeID(
			value.Signature,
		)
		if err != nil {
			return nil, err
		}
		out = append(out, TypeMethod{
			Name:      value.Name,
			Package:   pkg,
			Signature: signature,
			Ordinal:   value.Ordinal,
		})
	}
	decoder.methods += encoded.Count
	return out, nil
}

func (decoder *wireTypeDecoder) termRange(
	encoded wireTypeTermRange,
) ([]TypeTerm, error) {
	if encoded.Start != decoder.terms ||
		encoded.Count != uint64(len(encoded.Values)) {
		return nil, wrongWireRange(
			"type terms",
			encoded.Start,
			encoded.Count,
			len(encoded.Values),
			decoder.terms,
		)
	}
	out := make([]TypeTerm, 0, len(encoded.Values))
	for _, value := range encoded.Values {
		typeID, err := decoder.identities.typeID(value.Type)
		if err != nil {
			return nil, err
		}
		out = append(out, TypeTerm{
			Tilde: value.Tilde,
			Type:  typeID,
		})
	}
	decoder.terms += encoded.Count
	return out, nil
}

func wrongWireRange(
	name string,
	start uint64,
	count uint64,
	values int,
	cursor uint64,
) error {
	return &artifactError{
		reason: name +
			" range is not a canonical contiguous partition",
	}
}
