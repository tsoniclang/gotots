package interfacevalue

import (
	"go/ast"
	"go/types"
	"strconv"

	"github.com/tsoniclang/gotots/internal/emit/api"
	genericoperation "github.com/tsoniclang/gotots/internal/emit/generic/operation"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func AssertGeneric(
	context api.Context,
	source ast.Node,
	sourceType types.Type,
	targetType types.Type,
	commaOK bool,
	value api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	if source == nil ||
		sourceType == nil ||
		targetType == nil ||
		!api.ContainsGenericTypeParameter(targetType) {
		return api.ExpressionEmission{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "generic interface assertion input is invalid",
		}
	}
	operation := api.GenericOperationInterfaceAssert
	results := []types.Type{targetType}
	if commaOK {
		operation = api.GenericOperationInterfaceAssertOK
		results = append(results, types.Typ[types.Bool])
	}
	target, err := genericoperation.Call(
		context,
		source,
		operation,
		[]types.Type{sourceType},
		results,
		[]tsgo.Expression{value.Value()},
		value.Requests()...,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.NewExpressionEmission(
		value.Before(),
		target.Value(),
		target.Requests(),
	)
}

func GenericAssertionElement(
	context api.Context,
	assertion api.ExpressionEmission,
	index int,
) (api.ExpressionEmission, error) {
	if index < 0 || index > 1 || len(assertion.Before()) != 0 {
		return api.ExpressionEmission{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "generic interface assertion element is invalid",
		}
	}
	return api.DirectExpression(
		context.Factory().ElementAccessExpression(
			assertion.Value(),
			nil,
			context.Factory().NumericLiteral(
				strconv.Itoa(index),
				tsgo.TokenFlagsNone,
			),
			tsgo.NodeFlagsNone,
		),
		assertion.Requests()...,
	), nil
}
