package stagecheck

import (
	"fmt"
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/identity"
)

func (
	verifier *checkerSemanticVerifier,
) deriveIndependentUnnamedSignatureBindings() error {
	seen := map[types.Object]identity.OccurrenceID{}
	for _, occurrenceReference := range verifier.expected.order {
		occurrenceID := verifier.expected.
			occurrenceRecord(occurrenceReference).ID()
		node, present := verifier.index.OccurrenceNode(occurrenceID)
		if !present {
			continue
		}
		field, fieldNode := node.(*ast.Field)
		if !fieldNode || len(field.Names) != 0 {
			continue
		}
		if verifier.independentIntrinsicContractBinding(occurrenceID) {
			continue
		}
		role, signatureField :=
			verifier.independentSignatureBindingRole(occurrenceID)
		if !signatureField {
			continue
		}
		object, err := verifier.independentUnnamedSignatureObject(
			occurrenceID, field, role,
		)
		if err != nil {
			return err
		}
		implicit, implicitPresent := verifier.view.ImplicitOf(field)
		if !implicitPresent || implicit != object {
			return fmt.Errorf(
				"unnamed signature field %s does not independently exact-join Info.Implicits and Signature",
				occurrenceID,
			)
		}
		if existing, exists := seen[object]; exists &&
			existing != occurrenceID {
			return fmt.Errorf(
				"independent unnamed signature binding has anchors %s and %s",
				existing,
				occurrenceID,
			)
		}
		seen[object] = occurrenceID
	}
	return nil
}

func (
	verifier *checkerSemanticVerifier,
) independentIntrinsicContractBinding(
	occurrenceID identity.OccurrenceID,
) bool {
	definition := verifier.expected.occurrenceOwner(occurrenceID)
	if definition.IsZero() {
		definition = verifier.expected.structuralOccurrenceOwner(occurrenceID)
	}
	node, present := verifier.definitionNode(definition)
	if !present {
		return false
	}
	function, functionDefinition := node.(*ast.FuncDecl)
	if !functionDefinition {
		return false
	}
	object := verifier.independentDefinitionObject(function.Name)
	_, builtin := object.(*types.Builtin)
	return builtin
}

func (
	verifier *checkerSemanticVerifier,
) independentSignatureBindingRole(
	occurrenceID identity.OccurrenceID,
) (identity.SemanticBindingRole, bool) {
	parentID := verifier.expected.occurrence(occurrenceID).Parent()
	parent, present := verifier.index.OccurrenceNode(parentID)
	if !present {
		return identity.SemanticBindingInvalid, false
	}
	list, fieldList := parent.(*ast.FieldList)
	if !fieldList {
		return identity.SemanticBindingInvalid, false
	}
	definition := verifier.expected.occurrenceOwner(occurrenceID)
	if definition.IsZero() {
		definition = verifier.expected.structuralOccurrenceOwner(occurrenceID)
	}
	node, present := verifier.definitionNode(definition)
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

func (
	verifier *checkerSemanticVerifier,
) independentUnnamedSignatureObject(
	occurrenceID identity.OccurrenceID,
	field *ast.Field,
	role identity.SemanticBindingRole,
) (types.Object, error) {
	definition := verifier.expected.occurrenceOwner(occurrenceID)
	if definition.IsZero() {
		definition = verifier.expected.structuralOccurrenceOwner(occurrenceID)
	}
	signature, err := verifier.independentDefinitionSignature(definition)
	if err != nil {
		return nil, err
	}
	var object *types.Var
	switch role {
	case identity.SemanticBindingReceiver:
		object = signature.Recv()
	case identity.SemanticBindingParameter:
		object, err = verifier.independentUnnamedTupleObject(
			occurrenceID, field, signature.Params(),
		)
	case identity.SemanticBindingResult:
		object, err = verifier.independentUnnamedTupleObject(
			occurrenceID, field, signature.Results(),
		)
	}
	if err != nil {
		return nil, err
	}
	if object == nil || object.Name() != "" || object.Type() == nil {
		return nil, fmt.Errorf(
			"unnamed signature field %s has invalid independent checker binding",
			occurrenceID,
		)
	}
	return object, nil
}

func (
	verifier *checkerSemanticVerifier,
) independentDefinitionSignature(
	definition identity.DefinitionID,
) (*types.Signature, error) {
	if definition.IsZero() {
		return nil, fmt.Errorf(
			"unnamed signature field has no owning definition",
		)
	}
	node, present := verifier.definitionNode(definition)
	if !present {
		return nil, fmt.Errorf(
			"definition %s has no checker signature root",
			definition,
		)
	}
	var typ types.Type
	switch node := node.(type) {
	case *ast.FuncDecl:
		object, defined := verifier.view.DefOf(node.Name)
		if !defined || object == nil {
			return nil, fmt.Errorf(
				"function definition %s has no checker object",
				definition,
			)
		}
		typ = object.Type()
	case *ast.FuncLit:
		value, typed := verifier.view.TypeOf(node)
		if !typed {
			return nil, fmt.Errorf(
				"function literal definition %s has no checker type",
				definition,
			)
		}
		typ = value.Type
	default:
		return nil, fmt.Errorf(
			"definition %s has unnamed signature field under %T",
			definition,
			node,
		)
	}
	signature, signatureType :=
		types.Unalias(typ).Underlying().(*types.Signature)
	if !signatureType {
		return nil, fmt.Errorf(
			"definition %s has non-signature checker type %T",
			definition,
			typ,
		)
	}
	return signature, nil
}

func (
	verifier *checkerSemanticVerifier,
) independentUnnamedTupleObject(
	occurrenceID identity.OccurrenceID,
	field *ast.Field,
	tuple *types.Tuple,
) (*types.Var, error) {
	parentID := verifier.expected.occurrence(occurrenceID).Parent()
	parent, present := verifier.index.OccurrenceNode(parentID)
	if !present {
		return nil, fmt.Errorf(
			"unnamed signature field %s has no independent parent",
			occurrenceID,
		)
	}
	list, fieldList := parent.(*ast.FieldList)
	if !fieldList || tuple == nil {
		return nil, fmt.Errorf(
			"unnamed signature field %s has no independent field-list tuple",
			occurrenceID,
		)
	}
	offset := -1
	width := 0
	for _, candidate := range list.List {
		if candidate == field {
			offset = width
		}
		count := len(candidate.Names)
		if count == 0 {
			count = 1
		}
		width += count
	}
	if offset < 0 || width != tuple.Len() || offset >= tuple.Len() {
		return nil, fmt.Errorf(
			"unnamed signature field %s does not independently exact-join its checker tuple",
			occurrenceID,
		)
	}
	return tuple.At(offset), nil
}
