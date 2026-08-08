package maprepresentation

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func (v staticSpecializationValues) Hash(
	context api.Context,
	_ ast.Node,
	sourceType types.Type,
	value tsgo.Expression,
) (api.ExpressionEmission, error) {
	if sourceType != v.key {
		panic("unexpected specialization hash type")
	}
	abi, err := api.NewScalarABIFromSizes(
		context.IntegerRepresentation(),
		context.TypesSizes(),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	one, err := api.IntegerLiteral(
		context.Factory(),
		abi,
		api.PrimitiveInt32,
		"1",
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	hash := tsgo.Expression(context.Factory().BinaryExpression(
		nil,
		staticField(context, value, "x"),
		nil,
		context.Factory().BinaryOperatorToken(
			tsgo.BinaryOperatorAmpersandToken,
		),
		one,
	))
	return api.DirectExpression(hash), nil
}
