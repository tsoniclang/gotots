package index

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	expressionoperands "github.com/tsoniclang/gotots/internal/emit/expression/operands"
	mapruntime "github.com/tsoniclang/gotots/internal/emit/runtime/map"
	runtimeslice "github.com/tsoniclang/gotots/internal/emit/runtime/slice"
	basictype "github.com/tsoniclang/gotots/internal/emit/type/basic"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
	arrayvalue "github.com/tsoniclang/gotots/internal/emit/value/array"
	integeroperand "github.com/tsoniclang/gotots/internal/emit/value/integer/operand"
	"github.com/tsoniclang/gotots/internal/emit/value/maprepresentation"
	slicevalue "github.com/tsoniclang/gotots/internal/emit/value/slice"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Apply(
	context api.Context,
	source ast.Node,
	operandType types.Type,
	indexType types.Type,
	resultType types.Type,
	operand api.ExpressionEmission,
	index api.ExpressionEmission,
) (api.ExpressionEmission, bool, error) {
	if mapType, ok := maprepresentation.Source(context, operandType); ok {
		if !types.Identical(resultType, mapType.Element()) ||
			!types.AssignableTo(indexType, mapType.Key()) {
			return api.ExpressionEmission{}, false, nil
		}
		var err error
		operand, err = mapType.ReadReceiver(context, source, operand)
		if err != nil {
			return api.ExpressionEmission{}, true, err
		}
		ordered, err := expressionoperands.Preserve(
			context,
			api.TemporarySliceOperand,
			expressionoperands.Present(operand),
			expressionoperands.Present(index),
		)
		if err != nil {
			return api.ExpressionEmission{}, true, err
		}
		member, err := mapruntime.Name(mapruntime.MemberLookup)
		if err != nil {
			return api.ExpressionEmission{}, true, err
		}
		values := ordered.Values()
		target, err := api.NewExpressionEmission(
			ordered.Before(),
			context.Factory().CallExpression(
				context.Factory().PropertyAccessExpression(
					values[0],
					nil,
					context.Factory().Identifier(member),
					tsgo.NodeFlagsNone,
				),
				nil,
				nil,
				[]tsgo.Expression{values[1]},
				tsgo.NodeFlagsNone,
			),
			ordered.Requests(),
		)
		return target, true, err
	}
	if array, ok := arrayvalue.Resolve(context, operandType); ok {
		if !types.Identical(resultType, array.ElementType()) ||
			!integeroperand.Supports(context.TypesSizes(), indexType) {
			return api.ExpressionEmission{}, false, nil
		}
		target, err := array.ApplyIndex(context, operand, index)
		return target, true, err
	}
	if basictype.SupportsString(operandType) &&
		integeroperand.Supports(context.TypesSizes(), indexType) &&
		isByte(resultType) {
		ordered, err := expressionoperands.Preserve(
			context,
			api.TemporarySliceOperand,
			expressionoperands.Present(operand),
			expressionoperands.Present(index),
		)
		if err != nil {
			return api.ExpressionEmission{}, true, err
		}
		reference, err := context.Names().Runtime(
			api.RuntimeStringIndex,
			api.ImportPhaseValue,
		)
		if err != nil {
			return api.ExpressionEmission{}, true, err
		}
		values := ordered.Values()
		target := tsgo.Expression(context.Factory().CallExpression(
			context.Factory().Identifier(reference.Name()),
			nil,
			nil,
			values,
			tsgo.NodeFlagsNone,
		))
		if context.IntegerRepresentation() == api.IntegerRepresentationBigInt {
			target = context.Factory().CallExpression(
				context.Factory().Identifier("BigInt"),
				nil,
				nil,
				[]tsgo.Expression{target},
				tsgo.NodeFlagsNone,
			)
		}
		result, err := api.NewExpressionEmission(
			ordered.Before(),
			target,
			api.CombineRequests(
				ordered.Requests(),
				reference.Requests(),
			),
		)
		return result, true, err
	}
	_, elementType, represented := slicevalue.Resolve(operandType)
	defined, definedOK := definedtype.ResolveSlice(operandType)
	if definedOK {
		sliceType, _ := defined.Slice()
		elementType = sliceType.Elem()
		represented = true
	}
	if !represented ||
		!types.Identical(resultType, elementType) ||
		!integeroperand.Supports(context.TypesSizes(), indexType) {
		return api.ExpressionEmission{}, false, nil
	}
	var err error
	if definedOK {
		operand, err = defined.Project(context, operand)
		if err != nil {
			return api.ExpressionEmission{}, true, err
		}
	}
	ordered, err := expressionoperands.Preserve(
		context,
		api.TemporarySliceOperand,
		expressionoperands.Present(operand),
		expressionoperands.Present(index),
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	values := ordered.Values()
	target, err := api.NewExpressionEmission(
		ordered.Before(),
		context.Factory().CallExpression(
			context.Factory().PropertyAccessExpression(
				values[0],
				nil,
				context.Factory().Identifier(
					runtimeslice.MemberName(runtimeslice.MemberGet),
				),
				tsgo.NodeFlagsNone,
			),
			nil,
			nil,
			[]tsgo.Expression{values[1]},
			tsgo.NodeFlagsNone,
		),
		ordered.Requests(),
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	target, err = context.ContainerStorage().FromContainerStorage(
		context.WithRole(api.RoleSliceElement),
		source,
		elementType,
		target,
	)
	return target, true, err
}
