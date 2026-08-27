package channelbuiltin

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	channeloperation "github.com/tsoniclang/gotots/internal/emit/concurrency/channel"
	runtimechannel "github.com/tsoniclang/gotots/internal/emit/runtime/channel"
	integervalue "github.com/tsoniclang/gotots/internal/emit/value/integer"
	integeroperand "github.com/tsoniclang/gotots/internal/emit/value/integer/operand"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
	builtin *types.Builtin,
	discarded bool,
) (api.ExpressionEmission, bool, error) {
	if source == nil || builtin == nil {
		return api.ExpressionEmission{}, false, nil
	}
	switch types.Object(builtin) {
	case types.Universe.Lookup("make"):
		if _, ok := channeloperation.Resolve(
			context.TypesInfo().TypeOf(source),
		); !ok {
			return api.ExpressionEmission{}, false, nil
		}
		target, err := emitMake(context, children, source, discarded)
		return target, true, err
	case types.Universe.Lookup("close"):
		if !channelArgument(context, source) {
			return api.ExpressionEmission{}, false, nil
		}
		target, err := emitClose(context, children, source, discarded)
		return target, true, err
	case types.Universe.Lookup("len"):
		if !channelArgument(context, source) {
			return api.ExpressionEmission{}, false, nil
		}
		target, err := emitMeasure(
			context,
			children,
			source,
			discarded,
			runtimechannel.MemberLength,
		)
		return target, true, err
	case types.Universe.Lookup("cap"):
		if !channelArgument(context, source) {
			return api.ExpressionEmission{}, false, nil
		}
		target, err := emitMeasure(
			context,
			children,
			source,
			discarded,
			runtimechannel.MemberCapacity,
		)
		return target, true, err
	default:
		return api.ExpressionEmission{}, false, nil
	}
}

func channelArgument(
	context api.Context,
	source *ast.CallExpr,
) bool {
	if source == nil || len(source.Args) != 1 {
		return false
	}
	_, ok := channeloperation.Resolve(
		context.TypesInfo().TypeOf(source.Args[0]),
	)
	return ok
}

func emitMake(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
	discarded bool,
) (api.ExpressionEmission, error) {
	resultType := context.TypesInfo().TypeOf(source)
	model, ok := channeloperation.Resolve(resultType)
	if !ok ||
		source.Ellipsis.IsValid() ||
		len(source.Args) < 1 ||
		len(source.Args) > 2 ||
		(!discarded &&
			(context.ExpectedType() == nil ||
				!types.AssignableTo(resultType, context.ExpectedType()))) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	typeArgument := context.TypesInfo().TypeOf(source.Args[0])
	if typeArgument == nil || !types.Identical(typeArgument, resultType) {
		return api.ExpressionEmission{},
			api.Unsupported(
				context.WithRole(api.RoleCallArgumentType),
				api.CategoryType,
				source.Args[0],
			)
	}
	capacity, err := zeroCapacity(context)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if len(source.Args) == 2 {
		capacityType := context.TypesInfo().TypeOf(source.Args[1])
		if !integeroperand.Supports(context.TypesSizes(), capacityType) {
			return api.ExpressionEmission{},
				api.Unsupported(
					context.WithRole(api.RoleChannelCapacity),
					api.CategoryExpression,
					source.Args[1],
				)
		}
		capacity, err = integeroperand.Emit(
			context.WithRole(api.RoleChannelCapacity),
			children,
			source.Args[1],
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
	}
	elementType := model.Element()
	element, err := children.RepresentedType(
		context.WithRole(api.RoleChannelElementType),
		source,
		elementType,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	zero, err := context.Values().Zero(
		context.WithRole(api.RoleChannelElement),
		source,
		elementType,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	copyValue, err := context.Values().Transfer(
		context.WithRole(api.RoleChannelElement),
		source.Args[0],
		elementType,
		elementType,
		api.ValueTransferCopy,
		api.DirectExpression(context.Factory().Identifier("value")),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	runtime, err := context.Names().Runtime(
		api.RuntimeChannel,
		api.ImportPhaseValue,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	target := api.DirectExpression(
		context.Factory().CallExpression(
			context.Factory().PropertyAccessExpression(
				context.Factory().Identifier(runtime.Name()),
				nil,
				context.Factory().Identifier(runtimechannel.MakeMember),
				tsgo.NodeFlagsNone,
			),
			nil,
			[]tsgo.TypeNode{element.Value()},
			[]tsgo.Expression{
				capacity.Value(),
				valueFunction(context, nil, element.Value(), zero),
				valueFunction(
					context,
					[]tsgo.ParameterDeclaration{
						context.Factory().ParameterDeclaration(
							nil,
							nil,
							context.Factory().Identifier("value"),
							nil,
							element.Value(),
							nil,
						),
					},
					element.Value(),
					copyValue,
				),
			},
			tsgo.NodeFlagsNone,
		),
		api.CombineRequests(
			capacity.Requests(),
			element.Requests(),
			zero.Requests(),
			copyValue.Requests(),
			runtime.Requests(),
		)...,
	)
	target, err = api.NewExpressionEmission(
		capacity.Before(),
		target.Value(),
		target.Requests(),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return model.Wrap(context, target)
}

func zeroCapacity(context api.Context) (api.ExpressionEmission, error) {
	value, err := integervalue.Literal(context, types.Typ[types.Int], "0")
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.DirectExpression(value), nil
}

func valueFunction(
	context api.Context,
	parameters []tsgo.ParameterDeclaration,
	result tsgo.TypeNode,
	value api.ExpressionEmission,
) tsgo.ArrowFunction {
	statements := value.Before()
	statements = append(
		statements,
		context.Factory().ReturnStatement(value.Value()),
	)
	return context.Factory().ArrowFunction(
		nil,
		nil,
		parameters,
		result,
		context.Factory().EqualsGreaterThanToken(),
		context.Factory().Block(statements, true),
	)
}

func emitClose(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
	discarded bool,
) (api.ExpressionEmission, error) {
	if !discarded ||
		source.Ellipsis.IsValid() ||
		len(source.Args) != 1 {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	sourceType := context.TypesInfo().TypeOf(source.Args[0])
	model, ok := channeloperation.Resolve(sourceType)
	if !ok || model.Direction() == types.RecvOnly {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	channel, err := children.Expression(
		context.
			WithRole(api.RoleChannelOperand).
			WithExpectedType(sourceType),
		source.Args[0],
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	channel, err = model.Project(context, channel)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return channeloperation.StaticCall(
		context,
		source,
		runtimechannel.MemberClose,
		channel,
	)
}

func emitMeasure(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
	discarded bool,
	member string,
) (api.ExpressionEmission, error) {
	resultType := context.TypesInfo().TypeOf(source)
	sourceType := context.TypesInfo().TypeOf(source.Args[0])
	model, ok := channeloperation.Resolve(sourceType)
	if discarded ||
		!ok ||
		context.ExpectedType() == nil ||
		resultType == nil ||
		!types.AssignableTo(resultType, context.ExpectedType()) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	channel, err := children.Expression(
		context.
			WithRole(api.RoleChannelOperand).
			WithExpectedType(sourceType),
		source.Args[0],
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	channel, err = model.Project(context, channel)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	target, err := channeloperation.StaticCall(
		context,
		source,
		member,
		channel,
	)
	if err != nil || !context.ScalarABI().UsesBigInt(resultType) {
		return target, err
	}
	return api.NewExpressionEmission(
		target.Before(),
		context.Factory().CallExpression(
			api.TargetIntrinsicBigInt.Expression(context.Factory()),
			nil,
			nil,
			[]tsgo.Expression{target.Value()},
			tsgo.NodeFlagsNone,
		),
		target.Requests(),
	)
}
