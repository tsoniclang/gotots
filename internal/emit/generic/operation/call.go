package operation

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	expressionoperands "github.com/tsoniclang/gotots/internal/emit/expression/operands"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Call(
	context api.Context,
	source ast.Node,
	operation api.GenericOperation,
	parameterTypes []types.Type,
	resultTypes []types.Type,
	arguments []api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	selection, err := api.SelectGenericOperation(operation)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return call(
		context,
		source,
		selection,
		parameterTypes,
		resultTypes,
		arguments,
	)
}

func Reference(
	context api.Context,
	source ast.Node,
	operation api.GenericOperation,
	parameterTypes []types.Type,
	resultTypes []types.Type,
) (api.GenericOperationReference, error) {
	selection, err := api.SelectGenericOperation(operation)
	if err != nil {
		return api.GenericOperationReference{}, err
	}
	signature, err := operationSignature(
		context,
		parameterTypes,
		resultTypes,
	)
	if err != nil {
		return api.GenericOperationReference{}, err
	}
	return context.GenericOperation(source, selection.Operation(), signature)
}

func ConstraintMethod(
	context api.Context,
	source ast.Node,
	method *types.Func,
	parameterTypes []types.Type,
	resultTypes []types.Type,
	arguments []api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	selection, err := api.SelectGenericConstraintMethod(method)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return call(
		context,
		source,
		selection,
		parameterTypes,
		resultTypes,
		arguments,
	)
}

func call(
	context api.Context,
	source ast.Node,
	selection api.GenericOperationSelection,
	parameterTypes []types.Type,
	resultTypes []types.Type,
	arguments []api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	if len(parameterTypes) != len(arguments) {
		return api.ExpressionEmission{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "generic operation argument arity is inconsistent",
		}
	}
	signature, err := operationSignature(
		context,
		parameterTypes,
		resultTypes,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	ordered := make([]expressionoperands.Item, 0, len(arguments))
	for _, argument := range arguments {
		ordered = append(ordered, expressionoperands.Present(argument))
	}
	sequence, err := expressionoperands.Preserve(
		context,
		api.TemporaryCallArgument,
		ordered...,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	method, constraintMethod := selection.Method()
	var reference api.GenericOperationReference
	if constraintMethod {
		reference, err = context.GenericConstraintMethod(
			source,
			method,
			signature,
		)
	} else {
		reference, err = context.GenericOperation(
			source,
			selection.Operation(),
			signature,
		)
	}
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	target, err := api.NewExpressionEmission(
		sequence.Before(),
		context.Factory().CallExpression(
			context.Factory().Identifier(reference.Name()),
			nil,
			nil,
			sequence.Values(),
			tsgo.NodeFlagsNone,
		),
		api.CombineRequests(sequence.Requests(), reference.Requests()),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return target, nil
}

func operationSignature(
	context api.Context,
	parameterTypes []types.Type,
	resultTypes []types.Type,
) (*types.Signature, error) {
	parameters := make([]*types.Var, 0, len(parameterTypes))
	for _, sourceType := range parameterTypes {
		if sourceType == nil {
			return nil, &api.InvariantError{
				Role:   context.Role(),
				Reason: "generic operation parameter type is nil",
			}
		}
		parameters = append(
			parameters,
			types.NewVar(token.NoPos, nil, "", sourceType),
		)
	}
	results := make([]*types.Var, 0, len(resultTypes))
	for _, sourceType := range resultTypes {
		if sourceType == nil {
			return nil, &api.InvariantError{
				Role:   context.Role(),
				Reason: "generic operation result type is nil",
			}
		}
		results = append(
			results,
			types.NewVar(token.NoPos, nil, "", sourceType),
		)
	}
	return types.NewSignatureType(
		nil,
		nil,
		nil,
		types.NewTuple(parameters...),
		types.NewTuple(results...),
		false,
	), nil
}
