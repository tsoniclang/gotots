package operation

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
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
	reference, err := context.GenericOperation(
		source,
		operation,
		signature,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.DirectExpression(
		context.Factory().CallExpression(
			context.Factory().Identifier(reference.Name()),
			nil,
			nil,
			arguments,
			tsgo.NodeFlagsNone,
		),
		api.CombineRequests(requests, reference.Requests())...,
	), nil
}
