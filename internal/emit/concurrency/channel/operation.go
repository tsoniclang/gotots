package channel

import (
	"go/ast"

	"github.com/tsoniclang/gotots/internal/emit/api"
	expressionoperands "github.com/tsoniclang/gotots/internal/emit/expression/operands"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func StaticCall(
	context api.Context,
	source ast.Node,
	member string,
	arguments ...api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	if source == nil || member == "" || len(arguments) == 0 {
		return api.ExpressionEmission{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "channel runtime call is invalid",
		}
	}
	items := make([]expressionoperands.Item, 0, len(arguments))
	for _, argument := range arguments {
		items = append(items, expressionoperands.Present(argument))
	}
	ordered, err := expressionoperands.Preserve(
		context,
		api.TemporaryChannelOperand,
		items...,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	runtime, err := context.Names().Runtime(
		api.RuntimeChannel,
		api.ImportPhaseValue,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	call := context.Factory().CallExpression(
		context.Factory().PropertyAccessExpression(
			context.Factory().Identifier(runtime.Name()),
			nil,
			context.Factory().Identifier(member),
			tsgo.NodeFlagsNone,
		),
		nil,
		nil,
		ordered.Values(),
		tsgo.NodeFlagsNone,
	)
	return api.NewExpressionEmission(
		ordered.Before(),
		call,
		api.CombineRequests(
			ordered.Requests(),
			runtime.Requests(),
		),
	)
}

func BlockingCall(
	context api.Context,
	source ast.Node,
	member string,
	arguments ...api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	target, err := StaticCall(context, source, member, arguments...)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return target, nil
}
