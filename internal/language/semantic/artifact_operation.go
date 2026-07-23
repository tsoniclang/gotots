package semantic

import (
	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
)

func decodeOperation(encoded wireOperation) (Operation, error) {
	id, err := identity.ParseOperationID(encoded.ID)
	if err != nil {
		return Operation{}, err
	}
	resultType, err := parseOptionalType(encoded.ResultType)
	if err != nil {
		return Operation{}, err
	}
	expectedType, err := parseOptionalType(encoded.ExpectedType)
	if err != nil {
		return Operation{}, err
	}
	constant, err := decodeConstant(encoded.Constant)
	if err != nil {
		return Operation{}, err
	}
	object, err := decodeObjectReference(encoded.Object)
	if err != nil {
		return Operation{}, err
	}
	selection, err := decodeSelection(encoded.Selection)
	if err != nil {
		return Operation{}, err
	}
	instance, err := decodeInstance(encoded.Instance)
	if err != nil {
		return Operation{}, err
	}
	operands, err := parseOccurrences(encoded.Operands)
	if err != nil {
		return Operation{}, err
	}
	definitions, err := parseDefinitions(encoded.Definitions)
	if err != nil {
		return Operation{}, err
	}
	var implicit []ImplicitOperation
	for _, encoded := range encoded.Implicit {
		site, parseErr := identity.ParseOccurrenceID(encoded.Site)
		if parseErr != nil {
			return Operation{}, parseErr
		}
		source, parseErr := parseOptionalType(encoded.Source)
		if parseErr != nil {
			return Operation{}, parseErr
		}
		target, parseErr := parseOptionalType(encoded.Target)
		if parseErr != nil {
			return Operation{}, parseErr
		}
		record, parseErr := NewImplicitOperation(
			catalog.ImplicitOp(encoded.Kind),
			site,
			encoded.Ordinal,
			source,
			target,
		)
		if parseErr != nil {
			return Operation{}, parseErr
		}
		implicit = append(implicit, record)
	}
	controlTarget, err := parseOptionalOperation(
		encoded.ControlTarget,
	)
	if err != nil {
		return Operation{}, err
	}
	label, err := parseOptionalBinding(encoded.Label)
	if err != nil {
		return Operation{}, err
	}
	return NewOperation(OperationSpec{
		ID:            id,
		Kind:          OperationKind(encoded.Kind),
		Syntax:        catalog.Kind(encoded.Syntax),
		Variant:       catalog.Variant(encoded.Variant),
		Role:          catalog.Role(encoded.Role),
		Token:         catalog.TokenKind(encoded.Token),
		Mode:          ValueMode(encoded.Mode),
		Arity:         ResultArity(encoded.Arity),
		Place:         PlaceKind(encoded.Place),
		ResultType:    resultType,
		ExpectedType:  expectedType,
		Addressable:   encoded.Addressable,
		Assignable:    encoded.Assignable,
		HasOk:         encoded.HasOk,
		Constant:      constant,
		Object:        object,
		Selection:     selection,
		Instance:      instance,
		Operands:      operands,
		Definitions:   definitions,
		Implicit:      implicit,
		ControlTarget: controlTarget,
		Label:         label,
	})
}

func decodeObjectReference(
	encoded wireObjectReference,
) (ObjectReference, error) {
	switch ObjectReferenceKind(encoded.Kind) {
	case ObjectReferenceNone:
		return NoObjectReference(), nil
	case ObjectReferenceDeclaration:
		declaration, err := identity.ParseSemanticDeclarationID(
			encoded.Declaration,
		)
		if err != nil {
			return ObjectReference{}, err
		}
		return DeclarationReference(declaration)
	case ObjectReferenceBinding:
		binding, err := identity.ParseSemanticBindingID(
			encoded.Binding,
		)
		if err != nil {
			return ObjectReference{}, err
		}
		return BindingReference(binding)
	default:
		return ObjectReference{}, &artifactError{
			reason: "object reference has invalid kind",
		}
	}
}

func decodeSelection(
	encoded wireSelection,
) (Selection, error) {
	if encoded.Kind == 0 {
		if encoded.Receiver != "" ||
			encoded.Object != "" ||
			len(encoded.Index) != 0 ||
			encoded.Indirect {
			return Selection{}, &artifactError{
				reason: "zero selection carries fields",
			}
		}
		return Selection{}, nil
	}
	receiver, err := identity.ParseSemanticTypeID(encoded.Receiver)
	if err != nil {
		return Selection{}, err
	}
	object, err := identity.ParseSemanticDeclarationID(
		encoded.Object,
	)
	if err != nil {
		return Selection{}, err
	}
	return NewSelection(
		SelectionKind(encoded.Kind),
		receiver,
		object,
		encoded.Index,
		encoded.Indirect,
	)
}

func decodeInstance(encoded wireInstance) (Instance, error) {
	if encoded.Target.Kind == 0 {
		if len(encoded.Types) != 0 || encoded.Signature != "" {
			return Instance{}, &artifactError{
				reason: "zero instance carries fields",
			}
		}
		return Instance{}, nil
	}
	target, err := decodeObjectReference(encoded.Target)
	if err != nil {
		return Instance{}, err
	}
	types, err := parseTypes(encoded.Types)
	if err != nil {
		return Instance{}, err
	}
	signature, err := identity.ParseSemanticTypeID(
		encoded.Signature,
	)
	if err != nil {
		return Instance{}, err
	}
	return NewInstance(target, types, signature)
}

func decodeConstant(encoded wireConstant) (Constant, error) {
	if encoded.Kind == 0 {
		if encoded.Exact != "" {
			return Constant{}, &artifactError{
				reason: "zero constant carries exact value",
			}
		}
		return Constant{}, nil
	}
	return NewConstant(
		ConstantKind(encoded.Kind), encoded.Exact,
	)
}

func decodeUnsupported(
	encoded wireUnsupported,
	authority Authority,
) (Unsupported, error) {
	id, err := identity.ParseUnsupportedID(encoded.ID)
	if err != nil {
		return Unsupported{}, err
	}
	return NewUnsupported(
		id,
		UnsupportedReason(encoded.Reason),
		encoded.Evidence,
		authority,
	)
}
