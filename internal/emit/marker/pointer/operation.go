package pointer

import (
	"github.com/tsoniclang/gotots/internal/contracts/tsoniccore"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Operation(
	context api.Context,
	symbol tsoniccore.Symbol,
	typeArguments []api.TypeEmission,
	arguments []api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	reference, err := context.Names().TsonicCore(symbol)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	before := make([]tsgo.Statement, 0)
	types := make([]tsgo.TypeNode, 0, len(typeArguments))
	values := make([]tsgo.Expression, 0, len(arguments))
	requests := reference.Requests()
	for _, typeArgument := range typeArguments {
		types = append(types, typeArgument.Value())
		requests = api.CombineRequests(requests, typeArgument.Requests())
	}
	for _, argument := range arguments {
		before = append(before, argument.Before()...)
		values = append(values, argument.Value())
		requests = api.CombineRequests(requests, argument.Requests())
	}
	return api.NewExpressionEmission(
		before,
		context.Factory().CallExpression(
			reference.Expression(context.Factory()),
			nil,
			types,
			values,
			tsgo.NodeFlagsNone,
		),
		requests,
	)
}
