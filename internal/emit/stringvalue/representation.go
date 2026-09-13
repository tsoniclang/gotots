package stringvalue

import (
	"github.com/tsoniclang/gotots/internal/emit/api"
	runtimestring "github.com/tsoniclang/gotots/internal/emit/runtime/stringvalue"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func FromText(context api.Context, value api.ExpressionEmission) (api.ExpressionEmission, error) {
	return Construct(context, runtimestring.FromTextMember, value)
}

func Construct(context api.Context, member string, arguments ...api.ExpressionEmission) (api.ExpressionEmission, error) {
	reference, err := context.Names().Runtime(api.RuntimeStringValue, api.ImportPhaseValue)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	var before []tsgo.Statement
	var values []tsgo.Expression
	requests := reference.Requests()
	for _, argument := range arguments {
		before = append(before, argument.Before()...)
		values = append(values, argument.Value())
		requests = api.CombineRequests(requests, argument.Requests())
	}
	factory := context.Factory()
	return api.NewExpressionEmission(before, factory.CallExpression(factory.PropertyAccessExpression(reference.Expression(factory), nil,
		factory.Identifier(member), tsgo.NodeFlagsNone), nil, nil, values, tsgo.NodeFlagsNone), requests)
}

func Text(context api.Context, value api.ExpressionEmission) (api.ExpressionEmission, error) {
	return Member(context, runtimestring.TextMember, value)
}

func Member(context api.Context, name string, value api.ExpressionEmission, arguments ...tsgo.Expression) (api.ExpressionEmission, error) {
	factory := context.Factory()
	return api.NewExpressionEmission(value.Before(), factory.CallExpression(factory.PropertyAccessExpression(value.Value(), nil,
		factory.Identifier(name), tsgo.NodeFlagsNone), nil, nil, arguments, tsgo.NodeFlagsNone), value.Requests())
}
