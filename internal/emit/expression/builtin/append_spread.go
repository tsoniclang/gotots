package builtin

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
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
	stringSpread := appendStringSpread(elementType, spreadType)
	sliceSpread := spreadOK &&
		types.Identical(elementType, spreadElementType) &&
		types.AssignableTo(spreadType, types.NewSlice(elementType))
	if !resultOK ||
		!types.Identical(receiverType, result) ||
		(!sliceSpread && !stringSpread) {
		return api.ExpressionEmission{}, api.Unsupported(
			context,
			api.CategoryExpression,
			source,
		)
	}
	spreadExpected := spreadType
	if stringSpread {
		var expectedOK bool
		spreadExpected, expectedOK = stringArgumentExpectedType(spreadType)
		if !expectedOK {
			return api.ExpressionEmission{}, api.Unsupported(
				context.WithRole(api.RoleCallArgument),
				api.CategoryExpression,
				source.Args[1],
			)
		}
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
		context.WithRole(api.RoleCallArgument).WithExpectedType(spreadExpected),
		source.Args[1],
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if sliceSpread {
		spread, err = projectDefinedSlice(context, spreadType, spread)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
	} else {
		spread, err = projectDefinedString(context, spreadType, spread)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
	}
	operands, before, requests, err := arrangeValues(
		context,
		[]api.ExpressionEmission{receiver, spread},
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	targetElement, err := context.ContainerStorage().ContainerStorageType(
		context.WithRole(api.RoleSliceElementType),
		source,
		elementType,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	requests = api.CombineRequests(requests, targetElement.Requests())
	if stringSpread {
		emission, err := slicevalue.AppendString(
			context,
			targetElement,
			source,
			elementType,
			operands,
			before,
			requests,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		return wrapDefinedSlice(context, result, emission)
	}
	var target tsgo.Expression
	if context.Values().RequiresStructuralCopy(context, elementType) {
		emission, err := slicevalue.AppendSpreadAggregate(
			context,
			targetElement,
			source,
			elementType,
			operands,
			before,
			requests,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		return wrapDefinedSlice(context, result, emission)
	} else {
		zero, err := context.Values().Zero(
			context.WithRole(api.RoleSliceElement),
			source,
			elementType,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		zero, err = context.ContainerStorage().ToContainerStorage(
			context.WithRole(api.RoleSliceElement),
			source,
			elementType,
			zero,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		runtime, err := context.Names().Runtime(
			api.RuntimeSliceAppendSlice,
			api.ImportPhaseValue,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		before = append(before, zero.Before()...)
		target = context.Factory().CallExpression(
			context.Factory().Identifier(runtime.Name()),
			nil,
			[]tsgo.TypeNode{targetElement.Value()},
			[]tsgo.Expression{
				operands[0],
				operands[1],
				zero.Value(),
			},
			tsgo.NodeFlagsNone,
		)
		requests = api.CombineRequests(
			requests,
			zero.Requests(),
			runtime.Requests(),
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

func appendStringSpread(elementType, spreadType types.Type) bool {
	element, ok := types.Unalias(elementType).(*types.Basic)
	spread, spreadOK := types.Unalias(spreadType).Underlying().(*types.Basic)
	return ok &&
		element.Kind() == types.Uint8 &&
		spreadOK &&
		spread.Info()&types.IsString != 0
}
