package array

import (
	"go/ast"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func (a RuntimeArray) Zero(
	context api.Context,
	source ast.Node,
) (api.ExpressionEmission, error) {
	elementZero, err := context.Values().Zero(
		context,
		source,
		a.ElementType(),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if len(elementZero.Before()) != 0 {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	typeArguments, typeRequests, err := a.targetTypeArguments(context)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	target, requests, err := a.callStatic(
		context,
		"zero",
		typeArguments,
		a.lengthLiteral(context),
		elementZero.Value(),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.DirectExpression(
		target,
		api.CombineRequests(
			elementZero.Requests(),
			typeRequests,
			requests,
		)...,
	), nil
}

func (a RuntimeArray) Copy(
	context api.Context,
	fresh bool,
	value api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	if fresh {
		return api.NewExpressionEmission(
			value.Before(),
			value.Value(),
			value.Requests(),
		)
	}
	return api.NewExpressionEmission(
		value.Before(),
		callMember(context, value.Value(), "copy"),
		value.Requests(),
	)
}

func (a RuntimeArray) Equal(
	context api.Context,
	left tsgo.Expression,
	right tsgo.Expression,
) api.ExpressionEmission {
	return api.DirectExpression(callMember(context, left, "equal", right))
}
