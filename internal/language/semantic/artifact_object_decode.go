package semantic

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/identity"
)

type wireObjectDecoder struct {
	identities wireIdentityDecoder
	authority  Authority
	captures   uint64
}

func decodeWireConstant(
	encoded *wireConstantValue,
) (Constant, error) {
	if encoded == nil {
		return Constant{}, nil
	}
	return NewConstant(
		ConstantKind(encoded.Kind),
		encoded.Exact,
	)
}

func (decoder wireObjectDecoder) object(
	encoded wireObjectReference,
) (ObjectReference, error) {
	declaration, err := decoder.identities.declaration(
		encoded.Declaration,
	)
	if err != nil {
		return ObjectReference{}, err
	}
	binding, err := decoder.identities.binding(encoded.Binding)
	if err != nil {
		return ObjectReference{}, err
	}
	switch ObjectReferenceKind(encoded.Kind) {
	case ObjectReferenceNone:
		if !declaration.IsZero() || !binding.IsZero() {
			return ObjectReference{}, fmt.Errorf(
				"semantic no-object reference carries a target",
			)
		}
		return NoObjectReference(), nil
	case ObjectReferenceDeclaration:
		if !binding.IsZero() {
			return ObjectReference{}, fmt.Errorf(
				"semantic declaration reference carries a binding",
			)
		}
		return DeclarationReference(declaration)
	case ObjectReferenceBinding:
		if !declaration.IsZero() {
			return ObjectReference{}, fmt.Errorf(
				"semantic binding reference carries a declaration",
			)
		}
		return BindingReference(binding)
	default:
		return ObjectReference{}, fmt.Errorf(
			"semantic wire object reference kind %d is invalid",
			encoded.Kind,
		)
	}
}

func (decoder wireObjectDecoder) declaration(
	encoded wireDeclarationRecord,
) (Declaration, error) {
	id, err := decoder.identities.declaration(encoded.ID)
	if err != nil {
		return Declaration{}, err
	}
	pkg, err := decoder.identities.packageID(encoded.Package)
	if err != nil {
		return Declaration{}, err
	}
	typeID, err := decoder.identities.typeID(encoded.Type)
	if err != nil {
		return Declaration{}, err
	}
	constant, err := decodeWireConstant(encoded.Constant)
	if err != nil {
		return Declaration{}, err
	}
	return NewDeclaration(
		id,
		pkg,
		identity.SemanticObjectClass(encoded.Class),
		encoded.Name,
		typeID,
		encoded.Exported,
		constant,
		decoder.authority,
	)
}

func (decoder *wireObjectDecoder) binding(
	encoded wireBindingRecord,
) (Binding, error) {
	id, err := decoder.identities.binding(encoded.ID)
	if err != nil {
		return Binding{}, err
	}
	pkg, err := decoder.identities.packageID(encoded.Package)
	if err != nil {
		return Binding{}, err
	}
	definition, err := decoder.identities.definition(
		encoded.Definition,
	)
	if err != nil {
		return Binding{}, err
	}
	typeID, err := decoder.identities.typeID(encoded.Type)
	if err != nil {
		return Binding{}, err
	}
	source, err := decoder.identities.occurrence(encoded.Source)
	if err != nil {
		return Binding{}, err
	}
	captures, err := decodeReferenceRange(
		"binding captures",
		encoded.CapturedBy,
		&decoder.captures,
		decoder.identities.definition,
	)
	if err != nil {
		return Binding{}, err
	}
	return NewBinding(
		id,
		pkg,
		definition,
		identity.SemanticBindingRole(encoded.Role),
		encoded.Name,
		typeID,
		source,
		captures,
		decoder.authority,
	)
}

func (decoder wireObjectDecoder) unsupported(
	encoded wireUnsupportedRecord,
) (Unsupported, error) {
	id, err := decoder.identities.unsupportedID(encoded.ID)
	if err != nil {
		return Unsupported{}, err
	}
	return NewUnsupported(
		id,
		UnsupportedReason(encoded.Reason),
		encoded.Evidence,
		decoder.authority,
	)
}
