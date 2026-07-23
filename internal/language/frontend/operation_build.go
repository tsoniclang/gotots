package frontend

import (
	"fmt"
	"go/ast"
	"go/constant"
	"go/types"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
	"github.com/tsoniclang/gotots/internal/language/semantic"
	"github.com/tsoniclang/gotots/internal/language/typesemantics"
)

func (builder *packageBuilder) buildOperation(
	item pendingOperation,
) (semantic.Operation, error) {
	id := builder.operationByOccurrence[item.record.occurrence.ID()]
	mode, arity, place, resultType, addressable, assignable, value, err :=
		builder.operationValue(item)
	if err != nil {
		return semantic.Operation{}, err
	}
	expectedType, err := builder.optionalType(item.context.expected)
	if err != nil {
		return semantic.Operation{}, err
	}
	object, err := builder.operationObject(item)
	if err != nil {
		return semantic.Operation{}, err
	}
	selection, err := builder.operationSelection(item)
	if err != nil {
		return semantic.Operation{}, err
	}
	instance, err := builder.operationInstance(item)
	if err != nil {
		return semantic.Operation{}, err
	}
	operands := builder.operationOperands(item.record)
	definitions := builder.operationDefinitions(item.record)
	controlTarget, label, err := builder.operationControl(item)
	if err != nil {
		return semantic.Operation{}, err
	}
	implicit, err := builder.implicitEffects(item, operands, selection)
	if err != nil {
		return semantic.Operation{}, err
	}
	return semantic.NewOperation(semantic.OperationSpec{
		ID: id, Kind: item.kind,
		Syntax:  item.record.occurrence.Kind(),
		Variant: item.variant,
		Role:    item.record.occurrence.Role(),
		Token:   item.record.occurrence.Token(),
		Mode:    mode, Arity: arity, Place: place,
		ResultType: resultType, ExpectedType: expectedType,
		Addressable: addressable, Assignable: assignable,
		HasOk:    item.context.commaOK || value.HasOk(),
		Constant: constantFromTypeAndValue(value),
		Object:   object, Selection: selection, Instance: instance,
		Operands: operands, Definitions: definitions,
		Implicit: implicit, ControlTarget: controlTarget, Label: label,
	})
}

func (builder *packageBuilder) operationValue(
	item pendingOperation,
) (
	semantic.ValueMode,
	semantic.ResultArity,
	semantic.PlaceKind,
	identity.SemanticTypeID,
	bool,
	bool,
	types.TypeAndValue,
	error,
) {
	noValue := func() (
		semantic.ValueMode,
		semantic.ResultArity,
		semantic.PlaceKind,
		identity.SemanticTypeID,
		bool,
		bool,
		types.TypeAndValue,
		error,
	) {
		return semantic.ValueModeNone, semantic.ResultArityZero,
			semantic.PlaceNone, identity.SemanticTypeID{},
			false, false, types.TypeAndValue{}, nil
	}
	expression, expressionNode := item.record.node.(ast.Expr)
	if !expressionNode {
		return noValue()
	}
	if item.kind == semantic.OperationKeyedElement {
		return noValue()
	}
	if identifier, ok := expression.(*ast.Ident); ok && identifier.Name == "_" && operationNeedsPlace(item) {
		return semantic.ValueModePlace,
			semantic.ResultArityOne,
			semantic.PlaceBlank,
			identity.SemanticTypeID{},
			false, true, types.TypeAndValue{}, nil
	}
	value, present := builder.input.loaded.CheckerView().
		TypeOf(expression)
	if !present {
		return builder.operationValueWithoutType(item)
	}
	if value.IsType() {
		return semantic.ValueModeType, semantic.ResultArityOne,
			semantic.PlaceNone, identity.SemanticTypeID{},
			false, false, value, fmt.Errorf(
				"type-valued occurrence %s was classified as an operation",
				item.record.occurrence.ID(),
			)
	}
	if value.IsBuiltin() {
		return semantic.ValueModeBuiltin, semantic.ResultArityOne,
			semantic.PlaceNone, identity.SemanticTypeID{},
			false, false, value, nil
	}
	if value.IsNil() {
		return semantic.ValueModeNil, semantic.ResultArityOne,
			semantic.PlaceNone, identity.SemanticTypeID{},
			false, false, value, nil
	}
	if value.IsVoid() {
		return semantic.ValueModeVoid, semantic.ResultArityZero,
			semantic.PlaceNone, identity.SemanticTypeID{},
			false, false, value, nil
	}
	typeID, err := builder.types.build(value.Type)
	if err != nil {
		return semantic.ValueModeInvalid, semantic.ResultArityInvalid,
			semantic.PlaceInvalid, identity.SemanticTypeID{},
			false, false, value, fmt.Errorf(
				"operation %s result type %s: %w",
				item.record.occurrence.ID(), value.Type, err,
			)
	}
	mode := semantic.ValueModeValue
	place := semantic.PlaceNone
	assignable := value.Assignable()
	if operationNeedsPlace(item) {
		mode = semantic.ValueModePlace
		place, err = builder.operationPlace(item)
		if err != nil {
			return semantic.ValueModeInvalid,
				semantic.ResultArityInvalid,
				semantic.PlaceInvalid,
				identity.SemanticTypeID{},
				false, false, value, err
		}
		if place == semantic.PlaceBlank {
			typeID = identity.SemanticTypeID{}
			assignable = true
		} else {
			assignable = true
		}
	}
	arity := item.context.arity
	if item.context.commaOK || value.HasOk() {
		arity = semantic.ResultArityCommaOk
	} else if _, tuple := types.Unalias(value.Type).(*types.Tuple); tuple {
		mode = semantic.ValueModeTuple
		arity = semantic.ResultArityTuple
	}
	return mode, arity, place, typeID,
		value.Addressable(), assignable, value, nil
}

func (builder *packageBuilder) operationValueWithoutType(
	item pendingOperation,
) (
	semantic.ValueMode,
	semantic.ResultArity,
	semantic.PlaceKind,
	identity.SemanticTypeID,
	bool,
	bool,
	types.TypeAndValue,
	error,
) {
	identifier, identifierNode := item.record.node.(*ast.Ident)
	if identifierNode {
		if identifier.Name == "_" && operationNeedsPlace(item) {
			return semantic.ValueModePlace,
				semantic.ResultArityOne,
				semantic.PlaceBlank,
				identity.SemanticTypeID{},
				false, true, types.TypeAndValue{}, nil
		}
		if object := builder.identifierObject(identifier); object != nil {
			switch object.(type) {
			case *types.PkgName:
				return semantic.ValueModePackage,
					semantic.ResultArityOne,
					semantic.PlaceNone,
					identity.SemanticTypeID{},
					false, false, types.TypeAndValue{}, nil
			case *types.Label:
				return semantic.ValueModeLabel,
					semantic.ResultArityOne,
					semantic.PlaceNone,
					identity.SemanticTypeID{},
					false, false, types.TypeAndValue{}, nil
			case *types.Builtin:
				return semantic.ValueModeBuiltin,
					semantic.ResultArityOne,
					semantic.PlaceNone,
					identity.SemanticTypeID{},
					false, false, types.TypeAndValue{}, nil
			}
			typeID, err := builder.types.build(object.Type())
			if err != nil {
				return semantic.ValueModeInvalid,
					semantic.ResultArityInvalid,
					semantic.PlaceInvalid,
					identity.SemanticTypeID{},
					false, false, types.TypeAndValue{}, err
			}
			if operationNeedsPlace(item) {
				return semantic.ValueModePlace,
					semantic.ResultArityOne,
					semantic.PlaceBinding,
					typeID,
					true, true, types.TypeAndValue{}, nil
			}
			return semantic.ValueModeValue,
				semantic.ResultArityOne,
				semantic.PlaceNone,
				typeID,
				false, false, types.TypeAndValue{}, nil
		}
	}
	return semantic.ValueModeInvalid,
		semantic.ResultArityInvalid,
		semantic.PlaceInvalid,
		identity.SemanticTypeID{},
		false, false, types.TypeAndValue{}, &Error{
			Package:    builder.input.id,
			Definition: item.record.owner,
			Occurrence: item.record.occurrence.ID(),
			Kind:       item.record.occurrence.Kind(),
			Reason:     "expression operation has no checker type or exact object",
		}
}

func operationNeedsPlace(item pendingOperation) bool {
	switch item.record.occurrence.Role() {
	case catalog.RoleAssignmentTarget,
		catalog.RoleAssignablePlace,
		catalog.RoleRangeKey,
		catalog.RoleRangeValue:
		return true
	default:
		return item.kind == semantic.OperationDeclare ||
			item.kind == semantic.OperationStore
	}
}

func (builder *packageBuilder) operationPlace(
	item pendingOperation,
) (semantic.PlaceKind, error) {
	if identifier, ok := item.record.node.(*ast.Ident); ok {
		if identifier.Name == "_" {
			return semantic.PlaceBlank, nil
		}
		return semantic.PlaceBinding, nil
	}
	switch node := item.record.node.(type) {
	case *ast.SelectorExpr:
		return semantic.PlaceField, nil
	case *ast.StarExpr:
		return semantic.PlacePointerDereference, nil
	case *ast.IndexExpr:
		base := expressionType(
			builder.input.loaded.CheckerView(), node.X,
		)
		core, _ := typesemantics.Core(base)
		switch core.(type) {
		case *types.Map:
			return semantic.PlaceMapElement, nil
		case *types.Array:
			return semantic.PlaceArrayElement, nil
		case *types.Slice:
			return semantic.PlaceSliceElement, nil
		}
	}
	return semantic.PlaceInvalid, fmt.Errorf(
		"assignable occurrence %s has no exact place class",
		item.record.occurrence.ID(),
	)
}

func (builder *packageBuilder) optionalType(
	typ types.Type,
) (identity.SemanticTypeID, error) {
	if typ == nil {
		return identity.SemanticTypeID{}, nil
	}
	return builder.types.build(typ)
}

func constantFromTypeAndValue(
	value types.TypeAndValue,
) semantic.Constant {
	if value.Value == nil {
		return semantic.Constant{}
	}
	var kind semantic.ConstantKind
	switch value.Value.Kind() {
	case constant.Bool:
		kind = semantic.ConstantBool
	case constant.String:
		kind = semantic.ConstantString
	case constant.Int:
		kind = semantic.ConstantInteger
	case constant.Float:
		kind = semantic.ConstantFloat
	case constant.Complex:
		kind = semantic.ConstantComplex
	default:
		return semantic.Constant{}
	}
	out, _ := semantic.NewConstant(kind, value.Value.ExactString())
	return out
}

func (builder *packageBuilder) operationObject(
	item pendingOperation,
) (semantic.ObjectReference, error) {
	if identifier, ok := item.record.node.(*ast.Ident); ok &&
		identifier.Name == "_" {
		return semantic.NoObjectReference(), nil
	}
	if selection := operationCheckerSelection(
		builder.input.loaded.CheckerView(),
		item.record.node,
	); selection != nil {
		declaration, err :=
			builder.objects.declarationIDForSelection(selection)
		if err != nil {
			return semantic.ObjectReference{}, err
		}
		return semantic.DeclarationReference(declaration)
	}
	object := operationCheckerObject(
		builder.input.loaded.CheckerView(), item.record.node,
	)
	if object == nil {
		return semantic.NoObjectReference(), nil
	}
	return builder.objects.objectReference(object)
}

func operationCheckerSelection(
	view checkerExpressionView,
	node ast.Node,
) *types.Selection {
	var expression ast.Expr
	switch node := node.(type) {
	case *ast.CallExpr:
		expression = node.Fun
	case ast.Expr:
		expression = node
	default:
		return nil
	}
	for {
		switch current := ast.Unparen(expression).(type) {
		case *ast.SelectorExpr:
			selection, _ := view.SelectionOf(current)
			return selection
		case *ast.IndexExpr:
			expression = current.X
		case *ast.IndexListExpr:
			expression = current.X
		default:
			return nil
		}
	}
}

func operationCheckerObject(
	view checkerExpressionView,
	node ast.Node,
) types.Object {
	switch node := node.(type) {
	case *ast.Ident:
		if object, present := view.DefOf(node); present {
			return object
		}
		object, _ := view.UseOf(node)
		return object
	case *ast.SelectorExpr:
		return selectorObject(view, node)
	case *ast.CallExpr:
		return expressionObject(view, node.Fun)
	case *ast.IndexExpr:
		return expressionObject(view, node.X)
	case *ast.IndexListExpr:
		return expressionObject(view, node.X)
	default:
		return nil
	}
}

func expressionObject(
	view checkerExpressionView,
	expression ast.Expr,
) types.Object {
	switch expression := ast.Unparen(expression).(type) {
	case *ast.Ident:
		object, _ := view.UseOf(expression)
		return object
	case *ast.SelectorExpr:
		return selectorObject(view, expression)
	case *ast.IndexExpr:
		return expressionObject(view, expression.X)
	case *ast.IndexListExpr:
		return expressionObject(view, expression.X)
	default:
		return nil
	}
}

func (builder *packageBuilder) operationSelection(
	item pendingOperation,
) (semantic.Selection, error) {
	selector, ok := item.record.node.(*ast.SelectorExpr)
	if !ok {
		return semantic.Selection{}, nil
	}
	checkerSelection, present := builder.input.loaded.CheckerView().
		SelectionOf(selector)
	if !present {
		return semantic.Selection{}, nil
	}
	var kind semantic.SelectionKind
	switch checkerSelection.Kind() {
	case types.FieldVal:
		kind = semantic.SelectionField
	case types.MethodVal:
		kind = semantic.SelectionMethodValue
	case types.MethodExpr:
		kind = semantic.SelectionMethodExpression
	default:
		return semantic.Selection{}, fmt.Errorf(
			"selector %s has unknown checker selection kind %d",
			item.record.occurrence.ID(), checkerSelection.Kind(),
		)
	}
	receiver, err := builder.types.build(checkerSelection.Recv())
	if err != nil {
		return semantic.Selection{}, err
	}
	declaration, err :=
		builder.objects.declarationIDForSelection(
			checkerSelection,
		)
	if err != nil {
		return semantic.Selection{}, err
	}
	return semantic.NewSelection(
		kind, receiver, declaration,
		checkerSelection.Index(), checkerSelection.Indirect(),
	)
}

func (builder *packageBuilder) operationInstance(
	item pendingOperation,
) (semantic.Instance, error) {
	if item.kind != semantic.OperationGenericInstantiate &&
		item.kind != semantic.OperationCall {
		return semantic.Instance{}, nil
	}
	identifier := genericIdentifier(item.record.node)
	if identifier == nil {
		return semantic.Instance{}, nil
	}
	instance, present := builder.input.loaded.CheckerView().
		InstanceOf(identifier)
	if !present {
		return semantic.Instance{}, nil
	}
	object, present := builder.input.loaded.CheckerView().
		UseOf(identifier)
	if !present {
		return semantic.Instance{}, fmt.Errorf(
			"generic instance %s has no checker object",
			item.record.occurrence.ID(),
		)
	}
	target, err := builder.objects.objectReference(object)
	if err != nil {
		return semantic.Instance{}, err
	}
	arguments, err := builder.types.typeList(instance.TypeArgs)
	if err != nil {
		return semantic.Instance{}, err
	}
	signature, err := builder.types.build(instance.Type)
	if err != nil {
		return semantic.Instance{}, err
	}
	return semantic.NewInstance(target, arguments, signature)
}

func genericIdentifier(node ast.Node) *ast.Ident {
	var expression ast.Expr
	switch node := node.(type) {
	case ast.Expr:
		expression = node
	case *ast.CallExpr:
		expression = node.Fun
	default:
		return nil
	}
	switch expression := ast.Unparen(expression).(type) {
	case *ast.Ident:
		return expression
	case *ast.SelectorExpr:
		return expression.Sel
	case *ast.IndexExpr:
		return genericIdentifier(expression.X)
	case *ast.IndexListExpr:
		return genericIdentifier(expression.X)
	case *ast.CallExpr:
		return genericIdentifier(expression.Fun)
	default:
		return nil
	}
}
