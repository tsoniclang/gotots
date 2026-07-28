package api

import (
	"slices"

	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

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
	operands := []ExpressionEmission{e.accessorReceiver}
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
	receiver := DirectExpression(values[0])
	arguments := make([]ExpressionEmission, 0, len(values)-1)
	for _, value := range values[1:] {
		arguments = append(arguments, DirectExpression(value))
	}
	captured, err := NewAccessorStoreTargetEmission(
		receiver,
		e.getterMember,
		e.setterMember,
		arguments,
		e.sourceType,
	)
	if err != nil {
		return StoreTargetEmission{}, nil, nil, err
	}
	captured.copiesValue = e.copiesValue
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

func (e StoreTargetEmission) AccessorRead(
	context Context,
) (tsgo.Expression, error) {
	if !e.accessor {
		return nil, &ResultError{
			Result: "accessor read",
			Reason: "store target is not accessor-backed",
		}
	}
	arguments := make([]tsgo.Expression, 0, len(e.accessorArguments))
	for _, argument := range e.accessorArguments {
		arguments = append(arguments, argument.Value())
	}
	return accessorCall(
		context,
		e.accessorReceiver.Value(),
		e.getterMember,
		arguments,
	), nil
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
	operands := []ExpressionEmission{e.accessorReceiver}
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
	return NewExpressionEmission(
		before,
		accessorCall(
			context,
			values[0],
			e.setterMember,
			values[1:],
		),
		requests,
	)
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
