package operation

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	cooperativecall "github.com/tsoniclang/gotots/internal/emit/concurrency/cooperative"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Call(
	context api.Context,
	source ast.Node,
	operation api.GenericOperation,
	parameterTypes []types.Type,
	resultTypes []types.Type,
	arguments []tsgo.Expression,
	requests ...api.RootRequest,
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
		requests...,
	)
}

func ConstraintMethod(
	context api.Context,
	source ast.Node,
	method *types.Func,
	parameterTypes []types.Type,
	resultTypes []types.Type,
	arguments []tsgo.Expression,
	requests ...api.RootRequest,
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
		requests...,
	)
}

func call(
	context api.Context,
	source ast.Node,
	selection api.GenericOperationSelection,
	parameterTypes []types.Type,
	resultTypes []types.Type,
	arguments []tsgo.Expression,
	requests ...api.RootRequest,
) (api.ExpressionEmission, error) {
	if len(parameterTypes) != len(arguments) {
		return api.ExpressionEmission{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "generic operation argument arity is inconsistent",
		}
	}
	parameters := make([]*types.Var, 0, len(parameterTypes))
	for _, sourceType := range parameterTypes {
		if sourceType == nil {
			return api.ExpressionEmission{}, &api.InvariantError{
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
			return api.ExpressionEmission{}, &api.InvariantError{
				Role:   context.Role(),
				Reason: "generic operation result type is nil",
			}
		}
		results = append(
			results,
			types.NewVar(token.NoPos, nil, "", sourceType),
		)
	}
	signature := types.NewSignatureType(
		nil,
		nil,
		nil,
		types.NewTuple(parameters...),
		types.NewTuple(results...),
		false,
	)
	method, constraintMethod := selection.Method()
	var reference api.GenericOperationReference
	var err error
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
	target := api.DirectExpression(
		context.Factory().CallExpression(
			context.Factory().Identifier(reference.Name()),
			nil,
			nil,
			arguments,
			tsgo.NodeFlagsNone,
		),
		api.CombineRequests(requests, reference.Requests())...,
	)
	if reference.Contract().Consumer() !=
		api.GenericFunctionOperationConsumer() ||
		reference.Contract().Operation() !=
			api.GenericOperationConstraintMethod {
		return target, nil
	}
	return cooperativecall.GenericOperationCall(
		context,
		source,
		reference.Contract(),
		target,
	)
}
