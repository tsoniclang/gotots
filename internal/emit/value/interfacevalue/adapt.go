package interfacevalue

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	interfacetype "github.com/tsoniclang/gotots/internal/emit/type/interfacevalue"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func AdaptExpected(
	context api.Context,
	source ast.Expr,
	target api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	expected := context.ExpectedType()
	if _, ok := interfacetype.Resolve(expected); !ok {
		return target, nil
	}
	actual := context.TypesInfo().TypeOf(source)
	if actual == nil || !types.AssignableTo(actual, expected) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	if _, ok := interfacetype.Resolve(actual); ok {
		return target, nil
	}
	if basic, ok := types.Unalias(actual).(*types.Basic); ok &&
		basic.Kind() == types.UntypedNil {
		return api.DirectExpression(
			context.Factory().VoidExpression(
				context.Factory().NumericLiteral(
					"0",
					tsgo.TokenFlagsNone,
				),
			),
			target.Requests()...,
		), nil
	}
	actual = DynamicType(actual)
	copied, err := context.Values().Copy(
		context.WithRole(api.RoleConversionOperand),
		source,
		actual,
		target,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	adapter, err := context.Names().InterfaceAdapter(actual)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.NewExpressionEmission(
		copied.Before(),
		context.Factory().NewExpression(
			context.Factory().Identifier(adapter.Name()),
			nil,
			[]tsgo.Expression{copied.Value()},
		),
		api.CombineRequests(
			copied.Requests(),
			adapter.Requests(),
		),
	)
}

func OperandContext(
	context api.Context,
	source ast.Expr,
) api.Context {
	if _, ok := interfacetype.Resolve(context.ExpectedType()); !ok {
		return context
	}
	actual := context.TypesInfo().TypeOf(source)
	if actual == nil {
		return context
	}
	if _, ok := interfacetype.Resolve(actual); ok {
		return context
	}
	if basic, ok := types.Unalias(actual).(*types.Basic); ok &&
		basic.Kind() == types.UntypedNil {
		return context
	}
	return context.WithExpectedType(DynamicType(actual))
}

func DynamicType(sourceType types.Type) types.Type {
	if basic, ok := types.Unalias(sourceType).(*types.Basic); ok &&
		basic.Info()&types.IsUntyped != 0 &&
		basic.Kind() != types.UntypedNil {
		return types.Default(sourceType)
	}
	return sourceType
}

func Convert(
	context api.Context,
	source ast.Node,
	sourceType types.Type,
	targetType types.Type,
	value api.ExpressionEmission,
) (api.ExpressionEmission, bool, error) {
	if _, ok := interfacetype.Resolve(targetType); !ok {
		return api.ExpressionEmission{}, false, nil
	}
	if sourceType == nil || !types.ConvertibleTo(sourceType, targetType) {
		return api.ExpressionEmission{}, true,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	if _, ok := interfacetype.Resolve(sourceType); ok {
		return value, true, nil
	}
	if basic, ok := types.Unalias(sourceType).(*types.Basic); ok &&
		basic.Kind() == types.UntypedNil {
		return api.DirectExpression(
			context.Factory().VoidExpression(
				context.Factory().NumericLiteral(
					"0",
					tsgo.TokenFlagsNone,
				),
			),
			value.Requests()...,
		), true, nil
	}
	sourceType = DynamicType(sourceType)
	copied, err := context.Values().Copy(
		context.WithRole(api.RoleConversionOperand),
		source,
		sourceType,
		value,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	adapter, err := context.Names().InterfaceAdapter(sourceType)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	target, err := api.NewExpressionEmission(
		copied.Before(),
		context.Factory().NewExpression(
			context.Factory().Identifier(adapter.Name()),
			nil,
			[]tsgo.Expression{copied.Value()},
		),
		api.CombineRequests(
			copied.Requests(),
			adapter.Requests(),
		),
	)
	return target, true, err
}
