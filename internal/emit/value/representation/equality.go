package representation

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	genericoperation "github.com/tsoniclang/gotots/internal/emit/generic/operation"
	pointerruntime "github.com/tsoniclang/gotots/internal/emit/runtime/pointer"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
	interfacetype "github.com/tsoniclang/gotots/internal/emit/type/interfacevalue"
	arrayvalue "github.com/tsoniclang/gotots/internal/emit/value/array"
	complexvalue "github.com/tsoniclang/gotots/internal/emit/value/complex"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func (owner Owner) Equal(
	context api.Context,
	source ast.Node,
	sourceType types.Type,
	left tsgo.Expression,
	right tsgo.Expression,
) (api.ExpressionEmission, error) {
	if parameter, ok := api.GenericTypeParameter(sourceType); ok {
		return genericoperation.Call(
			context,
			source,
			api.GenericOperationEqual,
			[]types.Type{parameter, parameter},
			[]types.Type{types.Typ[types.Bool]},
			[]tsgo.Expression{left, right},
		)
	}
	if _, ok := interfacetype.Resolve(sourceType); ok {
		reference, err := context.Names().Runtime(
			api.RuntimeInterfaceEqual,
			api.ImportPhaseValue,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		return api.DirectExpression(
			context.Factory().CallExpression(
				context.Factory().Identifier(reference.Name()),
				nil,
				nil,
				[]tsgo.Expression{left, right},
				tsgo.NodeFlagsNone,
			),
			reference.Requests()...,
		), nil
	}
	if panicNilRuntimeValue(context, sourceType) {
		return panicNilEqual(context), nil
	}
	if defined, ok := definedtype.Resolve(sourceType); ok {
		if defined.Family() == definedtype.FamilyCallable {
			return api.DirectExpression(
				context.Factory().BinaryExpression(
					nil,
					left,
					nil,
					context.Factory().BinaryOperatorToken(
						tsgo.BinaryOperatorEqualsEqualsEqualsToken,
					),
					right,
				),
			), nil
		}
		leftValue := api.DirectExpression(left)
		rightValue := api.DirectExpression(right)
		var err error
		if defined.NilCapable() {
			leftValue, err = defined.Project(context, leftValue)
			if err == nil {
				rightValue, err = defined.Project(context, rightValue)
			}
		} else {
			leftValue = api.DirectExpression(
				defined.Unwrap(context.Factory(), left),
			)
			rightValue = api.DirectExpression(
				defined.Unwrap(context.Factory(), right),
			)
		}
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		result, err := (Owner{}).Equal(
			context.WithRole(api.RoleDefinedValue),
			source,
			defined.Underlying(),
			leftValue.Value(),
			rightValue.Value(),
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		return api.NewExpressionEmission(
			result.Before(),
			result.Value(),
			api.CombineRequests(
				leftValue.Requests(),
				rightValue.Requests(),
				result.Requests(),
			),
		)
	}
	if carrier, ok := complexvalue.Describe(sourceType); ok {
		symbol, valid := complexvalue.EqualSymbol(carrier)
		if !valid {
			return api.ExpressionEmission{}, &api.InvariantError{
				Role:   context.Role(),
				Reason: "complex equality has no runtime symbol",
			}
		}
		return complexvalue.Call(
			context,
			symbol,
			[]tsgo.Expression{left, right},
		)
	}
	if array, ok := arrayvalue.Resolve(context, sourceType); ok {
		return array.Equal(context, source, left, right)
	}
	if structType, ok := isAnonymousStruct(sourceType); ok {
		return anonymousStructEqual(context, source, structType, left, right)
	}
	if _, ok := primitive(context, sourceType); ok {
		return api.DirectExpression(context.Factory().BinaryExpression(
			nil,
			left,
			nil,
			context.Factory().BinaryOperatorToken(
				tsgo.BinaryOperatorEqualsEqualsEqualsToken,
			),
			right,
		)), nil
	}
	if pointerValue(sourceType) {
		reference, err := context.Names().Runtime(
			api.RuntimePointer,
			api.ImportPhaseValue,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		return api.DirectExpression(
			context.Factory().CallExpression(
				context.Factory().PropertyAccessExpression(
					context.Factory().Identifier(reference.Name()),
					nil,
					context.Factory().Identifier(pointerruntime.EqualName),
					tsgo.NodeFlagsNone,
				),
				nil,
				nil,
				[]tsgo.Expression{left, right},
				tsgo.NodeFlagsNone,
			),
			reference.Requests()...,
		), nil
	}
	if channelValue(sourceType) {
		return api.DirectExpression(
			context.Factory().BinaryExpression(
				nil,
				left,
				nil,
				context.Factory().BinaryOperatorToken(
					tsgo.BinaryOperatorEqualsEqualsEqualsToken,
				),
				right,
			),
		), nil
	}
	if callableValue(sourceType) {
		return api.DirectExpression(
			context.Factory().BinaryExpression(
				nil,
				left,
				nil,
				context.Factory().BinaryOperatorToken(
					tsgo.BinaryOperatorEqualsEqualsEqualsToken,
				),
				right,
			),
		), nil
	}
	_, _, ok := namedStruct(sourceType)
	if !ok {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	return owner.namedStructOperation(
		context,
		source,
		sourceType,
		api.NamedStructOperationEqual,
		[]tsgo.Expression{left, right},
	)
}
