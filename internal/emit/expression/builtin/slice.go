package builtin

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	runtimeslice "github.com/tsoniclang/gotots/internal/emit/runtime/slice"
	basictype "github.com/tsoniclang/gotots/internal/emit/type/basic"
	slicevalue "github.com/tsoniclang/gotots/internal/emit/value/slice"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func emitMake(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
	discarded bool,
) (api.ExpressionEmission, error) {
	result, err := resultType(context, source, discarded)
	if err != nil || discarded || len(source.Args) < 2 || len(source.Args) > 3 {
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	sourceType := context.TypesInfo().TypeOf(source.Args[0])
	_, elementType, ok := scalarSlice(context, sourceType)
	if !ok || !types.Identical(sourceType, result) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	length, err := integerArgument(context, children, source.Args[1])
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	capacity := length
	if len(source.Args) == 3 {
		capacity, err = integerArgument(context, children, source.Args[2])
		if err != nil {
			return api.ExpressionEmission{}, err
		}
	} else {
		capacity = api.DirectExpression(context.Factory().NullLiteral())
	}
	values := []api.ExpressionEmission{length, capacity}
	zero, err := context.Values().Zero(
		context.WithRole(api.RoleSliceElement),
		source,
		elementType,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	aggregate := context.Values().RequiresStructuralCopy(context, elementType)
	if aggregate {
		if len(zero.Before()) != 0 {
			return api.ExpressionEmission{},
				api.Unsupported(context, api.CategoryExpression, source)
		}
		arguments, before, requests, err := arrangeValues(
			context,
			[]api.ExpressionEmission{length, capacity},
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		arguments = append(
			arguments,
			slicevalue.ValueFactory(context, nil, zero.Value()),
		)
		target, err := aggregateSliceCall(
			context,
			children,
			source,
			elementType,
			api.RuntimeSliceMakeWith,
			arguments,
			before,
			api.CombineRequests(requests, zero.Requests()),
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		return wrapDefinedSlice(context, result, target)
	}
	values = append(values, zero)
	arguments, before, requests, err := arrangeValues(context, values)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	target, err := runtimeStaticCall(
		context,
		children,
		source,
		elementType,
		runtimeslice.MemberMake,
		arguments,
		before,
		requests,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return wrapDefinedSlice(context, result, target)
}

func emitMeasure(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
	discarded bool,
	member runtimeslice.Member,
) (api.ExpressionEmission, error) {
	if _, err := resultType(context, source, discarded); err != nil {
		return api.ExpressionEmission{}, err
	}
	if discarded || len(source.Args) != 1 {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	sourceType := context.TypesInfo().TypeOf(source.Args[0])
	if _, _, ok := scalarSlice(context, sourceType); !ok {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	value, err := children.Expression(
		context.
			WithRole(api.RoleCallArgument).
			WithExpectedType(sourceType),
		source.Args[0],
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	value, err = projectDefinedSlice(context, sourceType, value)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	target := tsgo.Expression(context.Factory().PropertyAccessExpression(
		value.Value(),
		nil,
		context.Factory().Identifier(runtimeslice.MemberName(member)),
		tsgo.NodeFlagsNone,
	))
	if context.IntegerRepresentation() == api.IntegerRepresentationBigInt {
		target = bigInt(context, target)
	}
	return api.NewExpressionEmission(
		value.Before(),
		target,
		value.Requests(),
	)
}

func emitAppend(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
	discarded bool,
) (api.ExpressionEmission, error) {
	result, err := resultType(context, source, discarded)
	if err != nil ||
		discarded ||
		source.Ellipsis.IsValid() ||
		len(source.Args) < 1 {
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	_, elementType, ok := scalarSlice(context, result)
	if !ok || !types.Identical(context.TypesInfo().TypeOf(source.Args[0]), result) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	values := make([]api.ExpressionEmission, 0, len(source.Args))
	receiver, err := children.Expression(
		context.
			WithRole(api.RoleSliceReceiver).
			WithExpectedType(result),
		source.Args[0],
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	receiver, err = projectDefinedSlice(context, result, receiver)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	values = append(values, receiver)
	for _, argument := range source.Args[1:] {
		argumentType := context.TypesInfo().TypeOf(argument)
		if argumentType == nil || !types.AssignableTo(argumentType, elementType) {
			return api.ExpressionEmission{},
				api.Unsupported(
					context.WithRole(api.RoleSliceElement),
					api.CategoryExpression,
					argument,
				)
		}
		target, err := children.Expression(
			context.
				WithRole(api.RoleSliceElement).
				WithExpectedType(elementType),
			argument,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		target, err = context.Values().Copy(
			context.WithRole(api.RoleSliceElement),
			argument,
			elementType,
			target,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		values = append(values, target)
	}
	ordered, before, requests, err := arrangeValues(context, values)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
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
		arguments := make([]tsgo.Expression, 0, len(ordered)+2)
		arguments = append(arguments, ordered[0], zero, copyValue)
		arguments = append(arguments, ordered[1:]...)
		target, err := aggregateSliceCall(
			context,
			children,
			source,
			elementType,
			api.RuntimeSliceAppendWith,
			arguments,
			before,
			api.CombineRequests(requests, zeroRequests, copyRequests),
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		return wrapDefinedSlice(context, result, target)
	}
	target, err := api.NewExpressionEmission(
		before,
		context.Factory().CallExpression(
			context.Factory().PropertyAccessExpression(
				ordered[0],
				nil,
				context.Factory().Identifier(
					runtimeslice.MemberName(runtimeslice.MemberAppend),
				),
				tsgo.NodeFlagsNone,
			),
			nil,
			nil,
			ordered[1:],
			tsgo.NodeFlagsNone,
		),
		requests,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return wrapDefinedSlice(context, result, target)
}

func emitCopy(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
	discarded bool,
) (api.ExpressionEmission, error) {
	if _, err := resultType(context, source, discarded); err != nil {
		return api.ExpressionEmission{}, err
	}
	if len(source.Args) != 2 {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	targetType := context.TypesInfo().TypeOf(source.Args[0])
	sourceType := context.TypesInfo().TypeOf(source.Args[1])
	_, elementType, targetOK := scalarSlice(context, targetType)
	_, sourceElement, sourceOK := scalarSlice(context, sourceType)
	if !targetOK || !sourceOK || !types.Identical(elementType, sourceElement) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	values := make([]api.ExpressionEmission, 0, 2)
	for index, argument := range source.Args {
		expected := targetType
		if index == 1 {
			expected = sourceType
		}
		value, err := children.Expression(
			context.
				WithRole(api.RoleCallArgument).
				WithExpectedType(expected),
			argument,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		value, err = projectDefinedSlice(context, expected, value)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		values = append(values, value)
	}
	arguments, before, requests, err := arrangeValues(context, values)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	var result api.ExpressionEmission
	if context.Values().RequiresStructuralCopy(context, elementType) {
		copyValue, copyRequests, err := slicevalue.AggregateCopyFactory(
			context,
			source,
			elementType,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		arguments = append(arguments, copyValue)
		result, err = aggregateSliceCall(
			context,
			children,
			source,
			elementType,
			api.RuntimeSliceCopyWith,
			arguments,
			before,
			api.CombineRequests(requests, copyRequests),
		)
	} else {
		result, err = runtimeStaticCall(
			context,
			children,
			source,
			elementType,
			runtimeslice.MemberCopy,
			arguments,
			before,
			requests,
		)
	}
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if context.IntegerRepresentation() != api.IntegerRepresentationBigInt {
		return result, nil
	}
	return api.NewExpressionEmission(
		result.Before(),
		bigInt(context, result.Value()),
		result.Requests(),
	)
}

func integerArgument(
	context api.Context,
	children api.ChildEmitter,
	source ast.Expr,
) (api.ExpressionEmission, error) {
	sourceType := context.TypesInfo().TypeOf(source)
	if !basictype.SupportsInteger(context.TypesSizes(), sourceType) {
		return api.ExpressionEmission{},
			api.Unsupported(
				context.WithRole(api.RoleCallArgument),
				api.CategoryExpression,
				source,
			)
	}
	return children.Expression(
		context.
			WithRole(api.RoleCallArgument).
			WithExpectedType(sourceType),
		source,
	)
}

func runtimeStaticCall(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	elementType types.Type,
	method runtimeslice.Member,
	arguments []tsgo.Expression,
	before []tsgo.Statement,
	requests []api.RootRequest,
) (api.ExpressionEmission, error) {
	element, err := children.RepresentedType(
		context.WithRole(api.RoleSliceElementType),
		source,
		elementType,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	runtime, err := context.Names().Runtime(
		api.RuntimeSlice,
		api.ImportPhaseValue,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.NewExpressionEmission(
		before,
		context.Factory().CallExpression(
			context.Factory().PropertyAccessExpression(
				context.Factory().Identifier(runtime.Name()),
				nil,
				context.Factory().Identifier(runtimeslice.MemberName(method)),
				tsgo.NodeFlagsNone,
			),
			nil,
			[]tsgo.TypeNode{element.Value()},
			arguments,
			tsgo.NodeFlagsNone,
		),
		api.CombineRequests(
			requests,
			element.Requests(),
			runtime.Requests(),
		),
	)
}

func arrangeValues(
	context api.Context,
	values []api.ExpressionEmission,
) (
	[]tsgo.Expression,
	[]tsgo.Statement,
	[]api.RootRequest,
	error,
) {
	capture := false
	for _, value := range values {
		capture = capture || len(value.Before()) != 0
	}
	targets := make([]tsgo.Expression, 0, len(values))
	var before []tsgo.Statement
	var requests []api.RootRequest
	for _, value := range values {
		requests = append(requests, value.Requests()...)
		if !capture {
			targets = append(targets, value.Value())
			continue
		}
		name, err := context.Names().Temporary(api.TemporaryCallArgument)
		if err != nil {
			return nil, nil, nil, err
		}
		before = append(before, value.Before()...)
		before = append(before, variable(context, name, value.Value()))
		targets = append(targets, context.Factory().Identifier(name))
	}
	return targets, before, requests, nil
}

func variable(
	context api.Context,
	name string,
	value tsgo.Expression,
) tsgo.VariableStatement {
	return context.Factory().VariableStatement(
		nil,
		context.Factory().VariableDeclarationList(
			[]tsgo.VariableDeclaration{
				context.Factory().VariableDeclaration(
					context.Factory().Identifier(name),
					nil,
					nil,
					value,
				),
			},
			tsgo.NodeFlagsConst,
		),
	)
}

func bigInt(context api.Context, value tsgo.Expression) tsgo.CallExpression {
	return context.Factory().CallExpression(
		context.Factory().Identifier("BigInt"),
		nil,
		nil,
		[]tsgo.Expression{value},
		tsgo.NodeFlagsNone,
	)
}
