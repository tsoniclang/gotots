package semantic

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
)

type wireOperationDecoder struct {
	identities       wireIdentityDecoder
	operands         uint64
	definitions      uint64
	implicit         uint64
	selectionIndexes uint64
	instanceTypes    uint64
}

func (decoder *wireOperationDecoder) record(
	encoded wireOperationRecord,
) (Operation, error) {
	id, err := decoder.identities.operation(encoded.ID)
	if err != nil {
		return Operation{}, err
	}
	resultType, err := decoder.identities.typeID(encoded.ResultType)
	if err != nil {
		return Operation{}, err
	}
	expectedType, err := decoder.identities.typeID(
		encoded.ExpectedType,
	)
	if err != nil {
		return Operation{}, err
	}
	constant, err := decodeWireConstant(encoded.Constant)
	if err != nil {
		return Operation{}, err
	}
	object, err := decoder.decodeObject(encoded.Object)
	if err != nil {
		return Operation{}, err
	}
	selection, err := decoder.decodeSelection(encoded.Selection)
	if err != nil {
		return Operation{}, err
	}
	instance, err := decoder.decodeInstance(encoded.Instance)
	if err != nil {
		return Operation{}, err
	}
	operands, err := decodeReferenceRange(
		"operation operands",
		encoded.Operands,
		&decoder.operands,
		decoder.identities.occurrence,
	)
	if err != nil {
		return Operation{}, err
	}
	definitions, err := decodeReferenceRange(
		"operation definitions",
		encoded.Definitions,
		&decoder.definitions,
		decoder.identities.definition,
	)
	if err != nil {
		return Operation{}, err
	}
	implicit, err := decoder.decodeImplicit(encoded.Implicit)
	if err != nil {
		return Operation{}, err
	}
	controlTarget, err := decoder.identities.operation(
		encoded.ControlTarget,
	)
	if err != nil {
		return Operation{}, err
	}
	label, err := decoder.identities.binding(encoded.Label)
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

func (decoder wireOperationDecoder) decodeObject(
	encoded *wireObjectReference,
) (ObjectReference, error) {
	if encoded == nil {
		return NoObjectReference(), nil
	}
	objects := wireObjectDecoder{identities: decoder.identities}
	return objects.object(*encoded)
}

func (decoder *wireOperationDecoder) decodeSelection(
	encoded *wireSelection,
) (Selection, error) {
	if encoded == nil {
		return Selection{}, nil
	}
	receiver, err := decoder.identities.typeID(encoded.Receiver)
	if err != nil {
		return Selection{}, err
	}
	object, err := decoder.identities.declaration(encoded.Object)
	if err != nil {
		return Selection{}, err
	}
	indexes, err := decodeIntegerRange(
		"selection index",
		encoded.Index,
		&decoder.selectionIndexes,
	)
	if err != nil {
		return Selection{}, err
	}
	return NewSelection(
		SelectionKind(encoded.Kind),
		receiver,
		object,
		indexes,
		encoded.Indirect,
	)
}

func (decoder *wireOperationDecoder) decodeInstance(
	encoded *wireInstance,
) (Instance, error) {
	if encoded == nil {
		return Instance{}, nil
	}
	objectDecoder := wireObjectDecoder{
		identities: decoder.identities,
	}
	target, err := objectDecoder.object(encoded.Target)
	if err != nil {
		return Instance{}, err
	}
	types, err := decodeReferenceRange(
		"instance types",
		encoded.Types,
		&decoder.instanceTypes,
		decoder.identities.typeID,
	)
	if err != nil {
		return Instance{}, err
	}
	signature, err := decoder.identities.typeID(encoded.Signature)
	if err != nil {
		return Instance{}, err
	}
	return NewInstance(target, types, signature)
}

func (decoder *wireOperationDecoder) decodeImplicit(
	encoded wireImplicitOperationRange,
) ([]ImplicitOperation, error) {
	if encoded.Start != decoder.implicit ||
		encoded.Count != uint64(len(encoded.Values)) {
		return nil, fmt.Errorf(
			"semantic wire implicit-operation range is not contiguous",
		)
	}
	out := make([]ImplicitOperation, 0, len(encoded.Values))
	for _, value := range encoded.Values {
		site, err := decoder.identities.occurrence(value.Site)
		if err != nil {
			return nil, err
		}
		source, err := decoder.identities.typeID(value.Source)
		if err != nil {
			return nil, err
		}
		target, err := decoder.identities.typeID(value.Target)
		if err != nil {
			return nil, err
		}
		record, err := NewImplicitOperation(
			catalog.ImplicitOp(value.Kind),
			site,
			value.Ordinal,
			source,
			target,
		)
		if err != nil {
			return nil, err
		}
		out = append(out, record)
	}
	decoder.implicit += encoded.Count
	return out, nil
}

func operationID(
	record Operation,
) identity.OperationID {
	return record.ID()
}
