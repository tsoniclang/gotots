package basic

import (
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/stringvalue"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Apply(
	context api.Context,
	sourceType types.Type,
	operator token.Token,
	left api.ExpressionEmission,
	right api.ExpressionEmission,
) (api.ExpressionEmission, bool, error) {
	targetOperator, ok := Operator(context, sourceType, operator)
	if !ok {
		return api.ExpressionEmission{}, false, nil
	}
	basic := types.Unalias(sourceType).(*types.Basic)
	if basic.Info()&types.IsString != 0 {
		var err error
		left, err = stringvalue.Text(context, left)
		if err != nil {
			return api.ExpressionEmission{}, true, err
		}
		right, err = stringvalue.Text(context, right)
		if err != nil {
			return api.ExpressionEmission{}, true, err
		}
	}
	result := api.DirectExpression(
		context.Factory().BinaryExpression(
			nil,
			left.Value(),
			nil,
			targetOperator,
			right.Value(),
		),
		api.CombineRequests(left.Requests(), right.Requests())...,
	)
	if basic.Info()&types.IsString != 0 && operator == token.ADD {
		value, err := stringvalue.FromText(context, result)
		return value, true, err
	}
	return result, true, nil
}

func Operator(
	context api.Context,
	sourceType types.Type,
	operator token.Token,
) (tsgo.BinaryOperatorToken, bool) {
	basic, ok := types.Unalias(sourceType).(*types.Basic)
	if !ok {
		return nil, false
	}
	var target tsgo.BinaryOperator
	switch basic.Kind() {
	case types.String:
		switch operator {
		case token.ADD:
			target = tsgo.BinaryOperatorPlusToken
		case token.EQL:
			target = tsgo.BinaryOperatorEqualsEqualsEqualsToken
		case token.NEQ:
			target = tsgo.BinaryOperatorExclamationEqualsEqualsToken
		case token.LSS:
			target = tsgo.BinaryOperatorLessThanToken
		case token.LEQ:
			target = tsgo.BinaryOperatorLessThanEqualsToken
		case token.GTR:
			target = tsgo.BinaryOperatorGreaterThanToken
		case token.GEQ:
			target = tsgo.BinaryOperatorGreaterThanEqualsToken
		default:
			return nil, false
		}
	case types.Bool:
		switch operator {
		case token.LAND:
			target = tsgo.BinaryOperatorAmpersandAmpersandToken
		case token.LOR:
			target = tsgo.BinaryOperatorBarBarToken
		case token.EQL:
			target = tsgo.BinaryOperatorEqualsEqualsEqualsToken
		case token.NEQ:
			target = tsgo.BinaryOperatorExclamationEqualsEqualsToken
		default:
			return nil, false
		}
	default:
		return nil, false
	}
	return context.Factory().BinaryOperatorToken(target), true
}
