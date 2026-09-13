package array

import (
	"go/ast"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func (a RuntimeArray) PointerToValue(context api.Context, children api.ChildEmitter, source ast.Node, value api.ExpressionEmission) (api.ExpressionEmission, error) {
	stored, err := a.storage(context, value)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	reference, err := context.Names().Runtime(api.RuntimeArraySlice, api.ImportPhaseValue)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	slice, err := api.NewExpressionEmission(stored.Before(), context.Factory().CallExpression(
		reference.Expression(context.Factory()), nil, nil, []tsgo.Expression{
			stored.Value(),
			context.Factory().NumericLiteral("0", tsgo.TokenFlagsNone),
			context.Factory().NullLiteral(),
			context.Factory().NullLiteral(),
		}, tsgo.NodeFlagsNone,
	), api.CombineRequests(stored.Requests(), reference.Requests()))
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return a.PointerFromSlice(context, children, source, slice)
}
