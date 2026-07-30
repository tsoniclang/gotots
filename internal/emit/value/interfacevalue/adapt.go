package interfacevalue

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	genericoperation "github.com/tsoniclang/gotots/internal/emit/generic/operation"
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
	adapted, handled, err := Assign(
		context,
		source,
		actual,
		expected,
		target,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if !handled {
		return api.ExpressionEmission{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "interface expectation was not handled by interface assignment",
		}
	}
	return adapted, nil
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
	if api.ContainsGenericTypeParameter(actual) {
		return context.WithExpectedType(actual)
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
	target, err := adapt(
		context,
		source,
		sourceType,
		targetType,
		value,
	)
	return target, true, err
}

func Assign(
	context api.Context,
	source ast.Node,
	sourceType types.Type,
	targetType types.Type,
	value api.ExpressionEmission,
) (api.ExpressionEmission, bool, error) {
	if _, ok := interfacetype.Resolve(targetType); !ok {
		return api.ExpressionEmission{}, false, nil
	}
	if sourceType == nil || !types.AssignableTo(sourceType, targetType) {
		return api.ExpressionEmission{}, true,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	target, err := adapt(
		context,
		source,
		sourceType,
		targetType,
		value,
	)
	return target, true, err
}

func adapt(
	context api.Context,
	source ast.Node,
	sourceType types.Type,
	targetType types.Type,
	value api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	if api.ContainsGenericTypeParameter(sourceType) {
		adapted, err := genericoperation.Call(
			context,
			source,
			api.GenericOperationInterfaceAdapt,
			[]types.Type{sourceType},
			[]types.Type{targetType},
			[]api.ExpressionEmission{value},
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		return adapted, nil
	}
	if _, ok := interfacetype.Resolve(sourceType); ok {
		demands, err := context.Names().InterfaceContractDemand(
			sourceType,
			targetType,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		return api.NewExpressionEmission(
			value.Before(),
			value.Value(),
			api.CombineRequests(value.Requests(), demands),
		)
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
		), nil
	}
	sourceType = DynamicType(sourceType)
	copied, err := context.Values().Transfer(
		context.WithRole(api.RoleConversionOperand),
		source,
		sourceType,
		sourceType,
		api.ValueTransferCopy,
		value,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	adapter, err := context.Names().InterfaceAdapter(
		sourceType,
		targetType,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
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
	return target, err
}
