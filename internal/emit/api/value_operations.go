package api

import (
	"go/ast"
	"go/token"
	"go/types"
	"slices"

	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type Values interface {
	RequiresCustomEquality(Context, types.Type) bool
	RequiresExplicitType(Context, types.Type) bool
	RequiresStructuralCopy(Context, types.Type) bool
	SupportsHash(Context, types.Type) bool
	RequiresStorageProjection(Context, types.Type) bool
	StorageType(Context, ast.Node, types.Type) (TypeEmission, error)
	ToStorage(
		Context,
		ast.Node,
		types.Type,
		ExpressionEmission,
	) (ExpressionEmission, error)
	FromStorage(
		Context,
		ast.Node,
		types.Type,
		ExpressionEmission,
	) (ExpressionEmission, error)
	Zero(Context, ast.Node, types.Type) (ExpressionEmission, error)
	Transfer(
		Context,
		ast.Node,
		types.Type,
		types.Type,
		ValueTransferMode,
		ExpressionEmission,
	) (ExpressionEmission, error)
	Assign(
		Context,
		ast.Node,
		types.Type,
		tsgo.Expression,
		ExpressionEmission,
	) (ExpressionEmission, error)
	Equal(
		Context,
		ast.Node,
		types.Type,
		tsgo.Expression,
		tsgo.Expression,
	) (ExpressionEmission, error)
	Hash(
		Context,
		ast.Node,
		types.Type,
		tsgo.Expression,
	) (ExpressionEmission, error)
	BinaryUpdate(
		Context,
		ast.Node,
		ast.Expr,
		types.Type,
		types.Type,
		token.Token,
		tsgo.Expression,
		ExpressionEmission,
	) (ExpressionEmission, bool, error)
	Increment(
		Context,
		ast.Node,
		types.Type,
		token.Token,
		tsgo.Expression,
	) (ExpressionEmission, bool, error)
}

type ValueTransferMode uint8

const (
	ValueTransferInvalid ValueTransferMode = iota
	ValueTransferCopy
	ValueTransferRepresentation
)

func (m ValueTransferMode) Valid() bool {
	return m == ValueTransferCopy ||
		m == ValueTransferRepresentation
}

func (e StoreTargetEmission) ReadValue(
	context Context,
	source ast.Node,
) (ExpressionEmission, error) {
	var value ExpressionEmission
	if e.accessor {
		target, err := e.accessorRead(context)
		if err != nil {
			return ExpressionEmission{}, err
		}
		value = target
	} else {
		var err error
		value, err = NewExpressionEmission(
			e.Before(),
			e.value,
			e.Requests(),
		)
		if err != nil {
			return ExpressionEmission{}, err
		}
	}
	if !e.canonicalStorage {
		return value, nil
	}
	return context.Values().FromStorage(
		context,
		source,
		e.sourceType,
		value,
	)
}

func (e StoreTargetEmission) StoreValue(
	context Context,
	source ast.Node,
	value ExpressionEmission,
) (ExpressionEmission, error) {
	if e.canonicalStorage {
		var err error
		value, err = context.Values().ToStorage(
			context,
			source,
			e.sourceType,
			value,
		)
		if err != nil {
			return ExpressionEmission{}, err
		}
	}
	if e.accessor {
		return e.AccessorStore(context, value)
	}
	if e.canonicalStorage {
		return NewExpressionEmission(
			append(
				e.Before(),
				value.Before()...,
			),
			context.Factory().BinaryExpression(
				nil,
				e.value,
				nil,
				context.Factory().BinaryOperatorToken(
					tsgo.BinaryOperatorEqualsToken,
				),
				value.Value(),
			),
			CombineRequests(e.Requests(), value.Requests()),
		)
	}
	assigned, err := context.Values().Assign(
		context,
		source,
		e.sourceType,
		e.value,
		value,
	)
	if err != nil {
		return ExpressionEmission{}, err
	}
	return NewExpressionEmission(
		append(e.Before(), assigned.Before()...),
		assigned.Value(),
		CombineRequests(e.Requests(), assigned.Requests()),
	)
}

func (e StoreTargetEmission) CaptureAccessorLocation(
	context Context,
) (StoreTargetEmission, error) {
	if !e.accessor {
		return StoreTargetEmission{}, &ResultError{
			Result: "accessor location",
			Reason: "store target is not accessor-backed",
		}
	}
	return e.CaptureLocation(context)
}

func (e StoreTargetEmission) CaptureLocation(
	context Context,
) (StoreTargetEmission, error) {
	captured, before, requests, err := e.PrepareLocation(context)
	if err != nil {
		return StoreTargetEmission{}, err
	}
	captured.before = before
	captured.requests = requests
	return captured, nil
}

func (e StoreTargetEmission) PrepareLocation(
	context Context,
) (
	StoreTargetEmission,
	[]tsgo.Statement,
	[]RootRequest,
	error,
) {
	if e.locationCaptured {
		return StoreTargetEmission{}, nil, nil, &ResultError{
			Result: "store location",
			Reason: "target location is already captured",
		}
	}
	if !e.accessor && !e.property {
		captured := e
		before := captured.Before()
		requests := captured.Requests()
		captured.before = nil
		captured.requests = nil
		captured.locationCaptured = true
		return captured, before, requests, nil
	}
	if e.property {
		return e.preparePropertyLocation(context)
	}
	return e.prepareAccessorLocation(context)
}

func (e StoreTargetEmission) preparePropertyLocation(
	context Context,
) (
	StoreTargetEmission,
	[]tsgo.Statement,
	[]RootRequest,
	error,
) {
	before, value, requests, err := captureStoreOperand(
		context,
		e.propertyReceiver,
	)
	if err != nil {
		return StoreTargetEmission{}, nil, nil, err
	}
	captured, err := NewPropertyStoreTargetEmission(
		context.Factory(),
		DirectExpression(value),
		e.propertyMember,
		e.sourceType,
	)
	if err != nil {
		return StoreTargetEmission{}, nil, nil, err
	}
	captured.copiesValue = e.copiesValue
	captured.canonicalStorage = e.canonicalStorage
	captured.locationCaptured = true
	return captured,
		append(slices.Clone(e.before), before...),
		append(slices.Clone(e.requests), requests...),
		nil
}

func (e StoreTargetEmission) prepareAccessorLocation(
	context Context,
) (
	StoreTargetEmission,
	[]tsgo.Statement,
	[]RootRequest,
	error,
) {
	var operands []ExpressionEmission
	if !e.accessorFunction {
		operands = append(operands, e.accessorReceiver)
	}
	operands = append(operands, e.accessorArguments...)
	values := make([]tsgo.Expression, 0, len(operands))
	var before []tsgo.Statement
	var requests []RootRequest
	for _, operand := range operands {
		operandBefore, value, operandRequests, err := captureStoreOperand(
			context,
			operand,
		)
		if err != nil {
			return StoreTargetEmission{}, nil, nil, err
		}
		before = append(before, operandBefore...)
		values = append(values, value)
		requests = append(requests, operandRequests...)
	}
	argumentStart := 0
	if !e.accessorFunction {
		argumentStart = 1
	}
	arguments := make([]ExpressionEmission, 0, len(values)-argumentStart)
	for _, value := range values[argumentStart:] {
		arguments = append(arguments, DirectExpression(value))
	}
	var captured StoreTargetEmission
	var err error
	if e.accessorFunction {
		captured, err = NewFunctionStoreTargetEmission(
			DirectExpression(e.getterFunction.Value()),
			DirectExpression(e.setterFunction.Value()),
			arguments,
			e.sourceType,
		)
		requests = CombineRequests(
			requests,
			e.getterFunction.Requests(),
			e.setterFunction.Requests(),
		)
	} else {
		captured, err = NewAccessorStoreTargetEmission(
			DirectExpression(values[0]),
			e.getterMember,
			e.setterMember,
			arguments,
			e.sourceType,
		)
	}
	if err != nil {
		return StoreTargetEmission{}, nil, nil, err
	}
	captured.copiesValue = e.copiesValue
	captured.canonicalStorage = e.canonicalStorage
	captured.locationCaptured = true
	return captured,
		append(slices.Clone(e.before), before...),
		append(slices.Clone(e.requests), requests...),
		nil
}

func captureStoreOperand(
	context Context,
	operand ExpressionEmission,
) ([]tsgo.Statement, tsgo.Expression, []RootRequest, error) {
	name, err := context.Names().Temporary(TemporaryStoreOperand)
	if err != nil {
		return nil, nil, nil, err
	}
	before := append(
		operand.Before(),
		constantStatement(context, name, operand.Value()),
	)
	return before,
		context.Factory().Identifier(name),
		operand.Requests(),
		nil
}

func (e StoreTargetEmission) accessorRead(
	context Context,
) (ExpressionEmission, error) {
	if !e.accessor {
		return ExpressionEmission{}, &ResultError{
			Result: "accessor read",
			Reason: "store target is not accessor-backed",
		}
	}
	var operands []ExpressionEmission
	if !e.accessorFunction {
		operands = append(operands, e.accessorReceiver)
	}
	operands = append(operands, e.accessorArguments...)
	values, before, requests, err := arrangeStoreOperands(context, operands)
	if err != nil {
		return ExpressionEmission{}, err
	}
	before = append(e.Before(), before...)
	requests = CombineRequests(e.requests, requests)
	var target tsgo.Expression
	if e.accessorFunction {
		target = functionCall(
			context,
			e.getterFunction.Value(),
			values,
		)
		requests = CombineRequests(
			requests,
			e.getterFunction.Requests(),
		)
	} else {
		target = accessorCall(
			context,
			values[0],
			e.getterMember,
			values[1:],
		)
	}
	return NewExpressionEmission(before, target, requests)
}

func (e StoreTargetEmission) AccessorStore(
	context Context,
	value ExpressionEmission,
) (ExpressionEmission, error) {
	if !e.accessor {
		return ExpressionEmission{}, &ResultError{
			Result: "accessor store",
			Reason: "store target is not accessor-backed",
		}
	}
	var operands []ExpressionEmission
	if !e.accessorFunction {
		operands = append(operands, e.accessorReceiver)
	}
	operands = append(operands, e.accessorArguments...)
	operands = append(operands, value)
	var values []tsgo.Expression
	var before []tsgo.Statement
	var requests []RootRequest
	if e.locationCaptured {
		values = make([]tsgo.Expression, 0, len(operands))
		for _, operand := range operands {
			values = append(values, operand.Value())
		}
		before = value.Before()
		requests = value.Requests()
	} else {
		var err error
		values, before, requests, err = arrangeStoreOperands(
			context,
			operands,
		)
		if err != nil {
			return ExpressionEmission{}, err
		}
	}
	before = append(e.Before(), before...)
	requests = append(slices.Clone(e.requests), requests...)
	var target tsgo.Expression
	if e.accessorFunction {
		target = functionCall(
			context,
			e.setterFunction.Value(),
			values,
		)
		requests = CombineRequests(
			requests,
			e.getterFunction.Requests(),
			e.setterFunction.Requests(),
		)
	} else {
		target = accessorCall(
			context,
			values[0],
			e.setterMember,
			values[1:],
		)
	}
	return NewExpressionEmission(before, target, requests)
}

func arrangeStoreOperands(
	context Context,
	operands []ExpressionEmission,
) ([]tsgo.Expression, []tsgo.Statement, []RootRequest, error) {
	capture := make([]bool, len(operands))
	laterHasPrerequisites := false
	for index := len(operands) - 1; index >= 0; index-- {
		capture[index] = laterHasPrerequisites
		if len(operands[index].Before()) != 0 {
			laterHasPrerequisites = true
		}
	}
	values := make([]tsgo.Expression, 0, len(operands))
	var before []tsgo.Statement
	var requests []RootRequest
	for index, operand := range operands {
		before = append(before, operand.Before()...)
		value := operand.Value()
		if capture[index] {
			name, err := context.Names().Temporary(TemporaryStoreOperand)
			if err != nil {
				return nil, nil, nil, err
			}
			before = append(before, constantStatement(context, name, value))
			value = context.Factory().Identifier(name)
		}
		values = append(values, value)
		requests = append(requests, operand.Requests()...)
	}
	return values, before, requests, nil
}

func accessorCall(
	context Context,
	receiver tsgo.Expression,
	member string,
	arguments []tsgo.Expression,
) tsgo.CallExpression {
	return context.Factory().CallExpression(
		context.Factory().PropertyAccessExpression(
			receiver,
			nil,
			context.Factory().Identifier(member),
			tsgo.NodeFlagsNone,
		),
		nil,
		nil,
		arguments,
		tsgo.NodeFlagsNone,
	)
}

func functionCall(
	context Context,
	function tsgo.Expression,
	arguments []tsgo.Expression,
) tsgo.CallExpression {
	return context.Factory().CallExpression(
		function,
		nil,
		nil,
		arguments,
		tsgo.NodeFlagsNone,
	)
}

func constantStatement(
	context Context,
	name string,
	value tsgo.Expression,
) tsgo.VariableStatement {
	return context.Factory().VariableStatement(
		nil,
		context.Factory().VariableDeclarationList(
			[]tsgo.VariableDeclaration{
				context.Factory().VariableDeclaration(
					context.Factory().Identifier(name),
					nil,
					nil,
					value,
				),
			},
			tsgo.NodeFlagsConst,
		),
	)
}
