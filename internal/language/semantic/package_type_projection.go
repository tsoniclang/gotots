package semantic

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/identity"
)

func (store packageTypeStore) record(
	identities *packageIdentityProjection,
	index int,
) (Type, error) {
	if index < 0 || index >= len(store.records) {
		return Type{}, fmt.Errorf(
			"semantic type index %d is invalid", index,
		)
	}
	stored := store.records[index]
	spec := TypeSpec{Kind: stored.kind}
	switch stored.kind {
	case TypeBasic:
		value, err := payloadAt(store.basic, stored.payload)
		if err != nil {
			return Type{}, err
		}
		spec.Basic = value
	case TypeNamed, TypeAlias:
		value, err := payloadAt(store.nominal, stored.payload)
		if err != nil {
			return Type{}, err
		}
		spec.Declaration = identities.declaration(value.declaration)
		spec.Arguments, err = store.projectTypes(
			identities, value.arguments,
		)
		if err != nil {
			return Type{}, err
		}
		if stored.kind == TypeNamed {
			spec.Underlying = identities.typeID(value.target)
			spec.Methods, err = store.projectMethods(
				identities, value.methods,
			)
			if err != nil {
				return Type{}, err
			}
		} else {
			spec.Target = identities.typeID(value.target)
		}
	case TypeParameter:
		value, err := payloadAt(store.parameters, stored.payload)
		if err != nil {
			return Type{}, err
		}
		spec.Parameter = TypeParameterOwner{
			declaration: identities.declaration(value.declaration),
			definition:  identities.definition(value.definition),
			role:        value.role,
			ordinal:     value.ordinal,
		}
		spec.Constraint = identities.typeID(value.constraint)
	case TypePointer, TypeSlice:
		value, err := payloadAt(store.elements, stored.payload)
		if err != nil {
			return Type{}, err
		}
		spec.Element = identities.typeID(value)
	case TypeArray:
		value, err := payloadAt(store.arrays, stored.payload)
		if err != nil {
			return Type{}, err
		}
		spec.Element = identities.typeID(value.element)
		spec.Length = value.length
	case TypeMap:
		value, err := payloadAt(store.maps, stored.payload)
		if err != nil {
			return Type{}, err
		}
		spec.Key = identities.typeID(value.key)
		spec.Element = identities.typeID(value.element)
	case TypeChannel:
		value, err := payloadAt(store.channels, stored.payload)
		if err != nil {
			return Type{}, err
		}
		spec.Element = identities.typeID(value.element)
		spec.Direction = value.direction
	case TypeSignature:
		value, err := payloadAt(store.signatures, stored.payload)
		if err != nil {
			return Type{}, err
		}
		spec.Signature, err = store.projectSignature(
			identities, value,
		)
		if err != nil {
			return Type{}, err
		}
	case TypeStruct:
		value, err := payloadAt(store.structs, stored.payload)
		if err != nil {
			return Type{}, err
		}
		spec.Fields, err = store.projectFields(
			identities, value,
		)
		if err != nil {
			return Type{}, err
		}
	case TypeInterface:
		value, err := payloadAt(store.interfaces, stored.payload)
		if err != nil {
			return Type{}, err
		}
		spec.Methods, err = store.projectMethods(
			identities, value.methods,
		)
		if err != nil {
			return Type{}, err
		}
		spec.Embeddeds, err = store.projectTypes(
			identities, value.embeddeds,
		)
		if err != nil {
			return Type{}, err
		}
		spec.Terms, err = store.projectTerms(
			identities, value.terms,
		)
		if err != nil {
			return Type{}, err
		}
		spec.TypeSet = value.typeSet
		spec.Comparable = value.comparable
	case TypeTuple:
		value, err := payloadAt(store.tuples, stored.payload)
		if err != nil {
			return Type{}, err
		}
		spec.Elements, err = store.projectTypes(identities, value)
		if err != nil {
			return Type{}, err
		}
	case TypeUnion:
		value, err := payloadAt(store.unions, stored.payload)
		if err != nil {
			return Type{}, err
		}
		spec.Terms, err = store.projectTerms(identities, value)
		if err != nil {
			return Type{}, err
		}
	default:
		return Type{}, fmt.Errorf(
			"semantic type has invalid kind %d", stored.kind,
		)
	}
	return Type{
		id:        identities.typeID(stored.id),
		spec:      spec,
		canonical: encodeTypeSpec(spec),
	}, nil
}

func (store packageTypeStore) projectTypes(
	identities *packageIdentityProjection,
	relation typeRefRange,
) ([]identity.SemanticTypeID, error) {
	return relationValues(
		store.typeRelations,
		relation.start,
		relation.count,
		identities.typeID,
	)
}

func (store packageTypeStore) projectSignature(
	identities *packageIdentityProjection,
	value storedSignature,
) (Signature, error) {
	var out Signature
	var err error
	out.Receiver = identities.typeID(value.receiver)
	out.ReceiverTypeParameters, err = store.projectTypes(
		identities, value.receiverTypeParameters,
	)
	if err != nil {
		return Signature{}, err
	}
	out.TypeParameters, err = store.projectTypes(
		identities, value.typeParameters,
	)
	if err != nil {
		return Signature{}, err
	}
	out.Parameters, err = store.projectTypes(
		identities, value.parameters,
	)
	if err != nil {
		return Signature{}, err
	}
	out.Results, err = store.projectTypes(
		identities, value.results,
	)
	if err != nil {
		return Signature{}, err
	}
	out.Variadic = value.variadic
	return out, nil
}

func (store packageTypeStore) projectFields(
	identities *packageIdentityProjection,
	relation typeFieldRange,
) ([]TypeField, error) {
	return relationValues(
		store.fields,
		relation.start,
		relation.count,
		func(value storedTypeField) TypeField {
			return TypeField{
				Name:     value.name,
				Package:  identities.packageID(value.pkg),
				Type:     identities.typeID(value.typeID),
				Embedded: value.embedded,
				Tag:      value.tag,
				Ordinal:  value.ordinal,
			}
		},
	)
}

func (store packageTypeStore) projectMethods(
	identities *packageIdentityProjection,
	relation typeMethodRange,
) ([]TypeMethod, error) {
	return relationValues(
		store.methods,
		relation.start,
		relation.count,
		func(value storedTypeMethod) TypeMethod {
			return TypeMethod{
				Name:      value.name,
				Package:   identities.packageID(value.pkg),
				Signature: identities.typeID(value.signature),
				Ordinal:   value.ordinal,
			}
		},
	)
}

func (store packageTypeStore) projectTerms(
	identities *packageIdentityProjection,
	relation typeTermRange,
) ([]TypeTerm, error) {
	return relationValues(
		store.terms,
		relation.start,
		relation.count,
		func(value storedTypeTerm) TypeTerm {
			return TypeTerm{
				Tilde: value.tilde,
				Type:  identities.typeID(value.typeID),
			}
		},
	)
}
