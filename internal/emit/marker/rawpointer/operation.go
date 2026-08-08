package rawpointer

import (
	"github.com/tsoniclang/gotots/internal/contracts/tsoniccore"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Operation(
	context api.Context,
	symbol tsoniccore.Symbol,
	arguments ...api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	reference, err := context.Names().TsonicCore(symbol)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	before := make([]tsgo.Statement, 0)
	values := make([]tsgo.Expression, 0, len(arguments))
	requests := reference.Requests()
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
			nil,
			values,
			tsgo.NodeFlagsNone,
		),
		requests,
	)
}
