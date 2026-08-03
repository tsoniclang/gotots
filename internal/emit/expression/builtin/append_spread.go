package builtin

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	genericoperation "github.com/tsoniclang/gotots/internal/emit/generic/operation"
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
	receiverType := context.TypesInfo().TypeOf(source.Args[0])
	spreadType := context.TypesInfo().TypeOf(source.Args[1])
	_, _, stringSpread, concrete := appendSpreadTypes(
		context,
		result,
		receiverType,
		spreadType,
	)
	if !concrete &&
		(api.ContainsGenericTypeParameter(result) ||
			api.ContainsGenericTypeParameter(receiverType) ||
			api.ContainsGenericTypeParameter(spreadType)) {
		return emitGenericAppendSpread(
			context,
			children,
			source,
			result,
			receiverType,
			spreadType,
		)
	}
	if !concrete {
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
		context.WithRole(api.RoleSliceReceiver).WithExpectedType(receiverType),
		source.Args[0],
	)
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
	target, handled, err := ApplyAppendSpread(
		context,
		source,
		result,
		receiverType,
		spreadType,
		receiver,
		spread,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if !handled {
		return api.ExpressionEmission{}, api.Unsupported(
			context,
			api.CategoryExpression,
			source,
		)
	}
	return target, nil
}

func emitGenericAppendSpread(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
	resultType types.Type,
	receiverType types.Type,
	spreadType types.Type,
) (api.ExpressionEmission, error) {
	if resultType == nil || receiverType == nil || spreadType == nil {
		return api.ExpressionEmission{}, api.Unsupported(
			context,
			api.CategoryExpression,
			source,
		)
	}
	receiver, err := children.Expression(
		context.WithRole(api.RoleSliceReceiver).WithExpectedType(receiverType),
		source.Args[0],
	)
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
	return genericoperation.Call(
		context,
		source,
		api.GenericOperationAppendSpread,
		[]types.Type{receiverType, spreadType},
		[]types.Type{resultType},
		[]api.ExpressionEmission{receiver, spread},
	)
}

func ApplyAppendSpread(
	context api.Context,
	source ast.Node,
	resultType types.Type,
	receiverType types.Type,
	spreadType types.Type,
	receiver api.ExpressionEmission,
	spread api.ExpressionEmission,
) (api.ExpressionEmission, bool, error) {
	elementType, sliceSpread, stringSpread, ok := appendSpreadTypes(
		context,
		resultType,
		receiverType,
		spreadType,
	)
	if !ok {
		return api.ExpressionEmission{}, false, nil
	}
	var err error
	receiver, err = projectDefinedSlice(context, resultType, receiver)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	if sliceSpread {
		spread, err = projectDefinedSlice(context, spreadType, spread)
	} else {
		spread, err = projectDefinedString(context, spreadType, spread)
	}
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	operands, before, requests, err := arrangeValues(
		context,
		[]api.ExpressionEmission{receiver, spread},
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	targetElement, err := context.ContainerStorage().ContainerStorageType(
		context.WithRole(api.RoleSliceElementType),
		source,
		elementType,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
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
			return api.ExpressionEmission{}, true, err
		}
		wrapped, err := wrapDefinedSlice(context, resultType, emission)
		return wrapped, true, err
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
			return api.ExpressionEmission{}, true, err
		}
		wrapped, err := wrapDefinedSlice(context, resultType, emission)
		return wrapped, true, err
	} else {
		zero, err := context.Values().Zero(
			context.WithRole(api.RoleSliceElement),
			source,
			elementType,
		)
		if err != nil {
			return api.ExpressionEmission{}, true, err
		}
		zero, err = context.ContainerStorage().ToContainerStorage(
			context.WithRole(api.RoleSliceElement),
			source,
			elementType,
			zero,
		)
		if err != nil {
			return api.ExpressionEmission{}, true, err
		}
		runtime, err := context.Names().Runtime(
			api.RuntimeSliceAppendSlice,
			api.ImportPhaseValue,
		)
		if err != nil {
			return api.ExpressionEmission{}, true, err
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
		return api.ExpressionEmission{}, true, err
	}
	wrapped, err := wrapDefinedSlice(
		context,
		resultType,
		emission,
	)
	return wrapped, true, err
}

func appendSpreadTypes(
	context api.Context,
	resultType types.Type,
	receiverType types.Type,
	spreadType types.Type,
) (types.Type, bool, bool, bool) {
	_, elementType, resultOK := scalarSlice(context, resultType)
	_, spreadElementType, spreadOK := scalarSlice(context, spreadType)
	stringSpread := resultOK && appendStringSpread(elementType, spreadType)
	sliceSpread := resultOK && spreadOK &&
		types.Identical(elementType, spreadElementType) &&
		types.AssignableTo(spreadType, types.NewSlice(elementType))
	return elementType,
		sliceSpread,
		stringSpread,
		resultOK &&
			types.Identical(receiverType, resultType) &&
			(sliceSpread || stringSpread)
}

func appendStringSpread(elementType, spreadType types.Type) bool {
	element, ok := types.Unalias(elementType).(*types.Basic)
	spread, spreadOK := types.Unalias(spreadType).Underlying().(*types.Basic)
	return ok &&
		element.Kind() == types.Uint8 &&
		spreadOK &&
		spread.Info()&types.IsString != 0
}
