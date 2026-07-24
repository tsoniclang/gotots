package frontend

import (
	"fmt"
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/identity"
)

func (index *objectIndex) indexUnnamedSignatureBindings() error {
	seen := map[types.Object]identity.OccurrenceID{}
	for _, occurrenceReference := range index.input.order {
		record := index.input.occurrenceRecord(occurrenceReference)
		occurrenceID := record.occurrence.ID()
		field, fieldNode := record.node.(*ast.Field)
		if !fieldNode || len(field.Names) != 0 {
			continue
		}
		context := index.contexts.contextAt(occurrenceReference)
		if index.intrinsicContractBinding(occurrenceID) {
			continue
		}
		role, signatureField := index.signatureBindingRole(
			occurrenceReference, record,
		)
		if !signatureField {
			continue
		}
		if context.bindingRole != role {
			return fmt.Errorf(
				"unnamed signature field %s context role=%s, structural role=%s",
				occurrenceID,
				context.bindingRole,
				role,
			)
		}
		object, err := index.unnamedSignatureObject(
			occurrenceReference, record, field, context, role,
		)
		if err != nil {
			return err
		}
		implicit, present := index.input.loaded.CheckerView().
			ImplicitOf(field)
		if !present || implicit != object {
			return fmt.Errorf(
				"unnamed signature field %s does not exact-join Info.Implicits and Signature",
				occurrenceID,
			)
		}
		if existing, present := seen[object]; present &&
			existing != occurrenceID {
			return fmt.Errorf(
				"unnamed signature binding has anchors %s and %s",
				existing,
				occurrenceID,
			)
		}
		seen[object] = occurrenceID
		index.work.CheckerSignatureBindingVisits++
	}
	return nil
}

func (index *objectIndex) intrinsicContractBinding(
	occurrenceID identity.OccurrenceID,
) bool {
	record := index.input.occurrence(occurrenceID)
	owner := index.input.occurrenceOwner(record)
	if owner.IsZero() {
		return false
	}
	node, present := index.input.index.CheckedDefinitionNode(
		owner,
	)
	if !present {
		node, present = index.input.index.DefinitionNode(owner)
	}
	function, functionDefinition := node.(*ast.FuncDecl)
	if !present || !functionDefinition {
		return false
	}
	object := definitionObject(index.input, owner)
	if object == nil {
		object, _ = index.input.loaded.CheckerView().
			DefOf(function.Name)
	}
	_, builtin := object.(*types.Builtin)
	return builtin
}

func (index *objectIndex) signatureBindingRole(
	reference packageOccurrenceRef,
	record *occurrenceInput,
) (identity.SemanticBindingRole, bool) {
	parent := index.input.occurrenceRecord(
		index.input.occurrenceParent(reference),
	)
	if parent == nil {
		return identity.SemanticBindingInvalid, false
	}
	list, fieldList := parent.node.(*ast.FieldList)
	if !fieldList {
		return identity.SemanticBindingInvalid, false
	}
	definition := index.input.occurrenceOwner(record)
	node, present := index.input.index.CheckedDefinitionNode(definition)
	if !present {
		node, present = index.input.index.DefinitionNode(definition)
	}
	if !present {
		return identity.SemanticBindingInvalid, false
	}
	var receiver, parameters, results *ast.FieldList
	switch node := node.(type) {
	case *ast.FuncDecl:
		receiver = node.Recv
		parameters = node.Type.Params
		results = node.Type.Results
	case *ast.FuncLit:
		parameters = node.Type.Params
		results = node.Type.Results
	default:
		return identity.SemanticBindingInvalid, false
	}
	switch list {
	case receiver:
		return identity.SemanticBindingReceiver, true
	case parameters:
		return identity.SemanticBindingParameter, true
	case results:
		return identity.SemanticBindingResult, true
	default:
		return identity.SemanticBindingInvalid, false
	}
}

func (index *objectIndex) unnamedSignatureObject(
	reference packageOccurrenceRef,
	record *occurrenceInput,
	field *ast.Field,
	context occurrenceContext,
	role identity.SemanticBindingRole,
) (types.Object, error) {
	signature := context.signature
	if signature == nil {
		return nil, fmt.Errorf(
			"unnamed signature field %s has no checker signature",
			record.occurrence.ID(),
		)
	}
	var object *types.Var
	switch role {
	case identity.SemanticBindingReceiver:
		object = signature.Recv()
	case identity.SemanticBindingParameter:
		value, err := index.unnamedTupleObject(
			reference, record, field, signature.Params(),
		)
		if err != nil {
			return nil, err
		}
		object = value
	case identity.SemanticBindingResult:
		value, err := index.unnamedTupleObject(
			reference, record, field, signature.Results(),
		)
		if err != nil {
			return nil, err
		}
		object = value
	}
	if object == nil || object.Name() != "" || object.Type() == nil {
		return nil, fmt.Errorf(
			"unnamed signature field %s has invalid checker binding",
			record.occurrence.ID(),
		)
	}
	return object, nil
}

func (index *objectIndex) unnamedTupleObject(
	reference packageOccurrenceRef,
	record *occurrenceInput,
	field *ast.Field,
	tuple *types.Tuple,
) (*types.Var, error) {
	parent := index.input.occurrenceRecord(
		index.input.occurrenceParent(reference),
	)
	if parent == nil {
		return nil, fmt.Errorf(
			"unnamed signature field %s has no structural parent",
			record.occurrence.ID(),
		)
	}
	list, fieldList := parent.node.(*ast.FieldList)
	if !fieldList || tuple == nil {
		return nil, fmt.Errorf(
			"unnamed signature field %s has no field-list tuple",
			record.occurrence.ID(),
		)
	}
	offset := 0
	found := false
	total := 0
	for _, candidate := range list.List {
		width := len(candidate.Names)
		if width == 0 {
			width = 1
		}
		if candidate == field {
			offset = total
			found = true
		}
		total += width
	}
	if !found || total != tuple.Len() || offset >= tuple.Len() {
		return nil, fmt.Errorf(
			"unnamed signature field %s does not exact-join its checker tuple",
			record.occurrence.ID(),
		)
	}
	return tuple.At(offset), nil
}
