package builtin

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	runtimeslice "github.com/tsoniclang/gotots/internal/emit/runtime/slice"
	slicevalue "github.com/tsoniclang/gotots/internal/emit/value/slice"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func emitAppendSpread(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
	discarded bool,
) (api.ExpressionEmission, error) {
	result, err := resultType(context, source, discarded)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if discarded ||
		source == nil ||
		!source.Ellipsis.IsValid() ||
		len(source.Args) != 2 {
		return api.ExpressionEmission{}, api.Unsupported(
			context,
			api.CategoryExpression,
			source,
		)
	}
	_, elementType, resultOK := scalarSlice(context, result)
	receiverType := context.TypesInfo().TypeOf(source.Args[0])
	spreadType := context.TypesInfo().TypeOf(source.Args[1])
	_, spreadElementType, spreadOK := scalarSlice(context, spreadType)
	if !resultOK ||
		!types.Identical(receiverType, result) ||
		!spreadOK ||
		!types.Identical(elementType, spreadElementType) ||
		!types.AssignableTo(spreadType, result) {
		return api.ExpressionEmission{}, api.Unsupported(
			context,
			api.CategoryExpression,
			source,
		)
	}
	receiver, err := children.Expression(
		context.WithRole(api.RoleSliceReceiver).WithExpectedType(result),
		source.Args[0],
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	receiver, err = projectDefinedSlice(context, result, receiver)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	spread, err := children.Expression(
		context.WithRole(api.RoleCallArgument).WithExpectedType(spreadType),
		source.Args[1],
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	spread, err = projectDefinedSlice(context, spreadType, spread)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	operands, before, requests, err := arrangeValues(
		context,
		[]api.ExpressionEmission{receiver, spread},
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	var target tsgo.Expression
	if context.Values().RequiresStructuralCopy(context, elementType) {
		zero, zeroRequests, err := slicevalue.AggregateZeroFactory(
			context,
			source,
			elementType,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		copyValue, copyRequests, err := slicevalue.AggregateCopyFactory(
			context,
			source,
			elementType,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		runtime, err := context.Names().Runtime(
			api.RuntimeSliceAppendSliceWith,
			api.ImportPhaseValue,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		target = context.Factory().CallExpression(
			context.Factory().Identifier(runtime.Name()),
			nil,
			nil,
			[]tsgo.Expression{operands[0], zero, copyValue, operands[1]},
			tsgo.NodeFlagsNone,
		)
		requests = api.CombineRequests(
			requests,
			zeroRequests,
			copyRequests,
			runtime.Requests(),
		)
	} else {
		target = context.Factory().CallExpression(
			context.Factory().PropertyAccessExpression(
				operands[0],
				nil,
				context.Factory().Identifier(
					runtimeslice.MemberName(runtimeslice.MemberAppendSlice),
				),
				tsgo.NodeFlagsNone,
			),
			nil,
			nil,
			[]tsgo.Expression{operands[1]},
			tsgo.NodeFlagsNone,
		)
	}
	emission, err := api.NewExpressionEmission(before, target, requests)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return wrapDefinedSlice(
		context,
		result,
		emission,
	)
}
