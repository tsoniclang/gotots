package interfacevalue

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	interfaceadapter "github.com/tsoniclang/gotots/internal/emit/declaration/interfaceadapter"
	interfacetype "github.com/tsoniclang/gotots/internal/emit/type/interfacevalue"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Test(
	context api.Context,
	targetType types.Type,
	value tsgo.Expression,
) (api.ExpressionEmission, error) {
	if _, ok := interfacetype.Resolve(targetType); ok {
		contract, err := context.Names().InterfaceContract(targetType)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		return api.DirectExpression(
			context.Factory().CallExpression(
				context.Factory().Identifier(contract.GuardName()),
				nil,
				nil,
				[]tsgo.Expression{value},
				tsgo.NodeFlagsNone,
			),
			contract.Requests()...,
		), nil
	}
	adapter, err := context.Names().InterfaceAdapter(targetType)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.DirectExpression(
		context.Factory().CallExpression(
			context.Factory().PropertyAccessExpression(
				context.Factory().Identifier(adapter.Name()),
				nil,
				context.Factory().Identifier(interfaceadapter.GuardMember),
				tsgo.NodeFlagsNone,
			),
			nil,
			nil,
			[]tsgo.Expression{value},
			tsgo.NodeFlagsNone,
		),
		adapter.Requests()...,
	), nil
}

func Extract(
	context api.Context,
	source ast.Node,
	targetType types.Type,
	value tsgo.Expression,
) (api.ExpressionEmission, error) {
	if _, ok := interfacetype.Resolve(targetType); ok {
		contract, err := context.Names().InterfaceContract(targetType)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		return api.DirectExpression(value, contract.Requests()...), nil
	}
	adapter, err := context.Names().InterfaceAdapter(targetType)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	payload := context.Factory().PropertyAccessExpression(
		value,
		nil,
		context.Factory().Identifier(interfaceadapter.ValueMember),
		tsgo.NodeFlagsNone,
	)
	target, err := context.Values().Copy(
		context,
		source,
		targetType,
		api.DirectExpression(payload),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if len(target.Before()) != 0 {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	return api.NewExpressionEmission(
		nil,
		target.Value(),
		api.CombineRequests(target.Requests(), adapter.Requests()),
	)
}
