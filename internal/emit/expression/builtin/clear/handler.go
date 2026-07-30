package clear

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	genericoperation "github.com/tsoniclang/gotots/internal/emit/generic/operation"
	mapruntime "github.com/tsoniclang/gotots/internal/emit/runtime/map"
	mapvalue "github.com/tsoniclang/gotots/internal/emit/value/maprepresentation"
	slicevalue "github.com/tsoniclang/gotots/internal/emit/value/slice"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
	builtin *types.Builtin,
	discarded bool,
) (api.ExpressionEmission, bool, error) {
	if builtin == nil ||
		types.Object(builtin) != types.Universe.Lookup("clear") {
		return api.ExpressionEmission{}, false, nil
	}
	target, err := emit(
		context,
		children,
		source,
		discarded,
		false,
	)
	return target, true, err
}

func EmitDeferred(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
	builtin *types.Builtin,
) (api.ExpressionEmission, bool, error) {
	if builtin == nil ||
		types.Object(builtin) != types.Universe.Lookup("clear") {
		return api.ExpressionEmission{}, false, nil
	}
	target, err := emit(context, children, source, true, true)
	return target, true, err
}

func emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
	discarded bool,
	capture bool,
) (api.ExpressionEmission, error) {
	if source == nil ||
		!discarded ||
		source.Ellipsis.IsValid() ||
		len(source.Args) != 1 {
		return api.ExpressionEmission{}, api.Unsupported(
			context,
			api.CategoryExpression,
			source,
		)
	}
	argument := source.Args[0]
	argumentType := context.TypesInfo().TypeOf(argument)
	if argumentType == nil {
		return api.ExpressionEmission{}, api.Unsupported(
			context.WithRole(api.RoleCallArgument),
			api.CategoryType,
			argument,
		)
	}
	if api.ContainsGenericTypeParameter(argumentType) {
		operand, err := children.Expression(
			context.
				WithRole(api.RoleBuiltinArgument).
				WithExpectedType(argumentType),
			argument,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		if capture {
			operand, err = captureReceiver(context, operand)
			if err != nil {
				return api.ExpressionEmission{}, err
			}
		}
		target, err := genericoperation.Call(
			context,
			source,
			api.GenericOperationClear,
			[]types.Type{argumentType},
			nil,
			[]tsgo.Expression{operand.Value()},
			operand.Requests()...,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		return api.NewExpressionEmission(
			operand.Before(),
			target.Value(),
			target.Requests(),
		)
	}
	if mapType, ok := mapvalue.Source(context, argumentType); ok {
		return emitMap(
			context,
			children,
			source,
			argument,
			mapType,
			capture,
		)
	}
	_, elementType, ok := slicevalue.Source(argumentType)
	if !ok {
		return api.ExpressionEmission{}, api.Unsupported(
			context.WithRole(api.RoleCallArgument),
			api.CategoryType,
			argument,
		)
	}
	return emitSlice(
		context,
		children,
		source,
		argument,
		argumentType,
		elementType,
		capture,
	)
}

func Apply(
	context api.Context,
	source ast.Node,
	argumentType types.Type,
	argument api.ExpressionEmission,
) (api.ExpressionEmission, bool, error) {
	if mapType, ok := mapvalue.Source(context, argumentType); ok {
		target, err := applyMap(context, source, mapType, argument)
		return target, true, err
	}
	_, elementType, ok := slicevalue.Source(argumentType)
	if !ok {
		return api.ExpressionEmission{}, false, nil
	}
	target, err := applySlice(
		context,
		source,
		argumentType,
		elementType,
		argument,
	)
	return target, true, err
}

func emitMap(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
	argument ast.Expr,
	mapType mapvalue.Model,
	capture bool,
) (api.ExpressionEmission, error) {
	receiver, err := children.Expression(
		context.WithRole(api.RoleMapReceiver).WithExpectedType(mapType.Type()),
		argument,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if capture {
		receiver, err = captureReceiver(context, receiver)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
	}
	return applyMap(context, source, mapType, receiver)
}

func applyMap(
	context api.Context,
	source ast.Node,
	mapType mapvalue.Model,
	receiver api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	var err error
	receiver, err = mapType.ReadReceiver(context, source, receiver)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	name, err := mapruntime.Name(mapruntime.MemberClear)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.NewExpressionEmission(
		receiver.Before(),
		methodCall(context, receiver.Value(), name),
		receiver.Requests(),
	)
}

func emitSlice(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
	argument ast.Expr,
	argumentType types.Type,
	elementType types.Type,
	capture bool,
) (api.ExpressionEmission, error) {
	receiver, err := children.Expression(
		context.WithRole(api.RoleSliceReceiver).WithExpectedType(argumentType),
		argument,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if capture {
		receiver, err = captureReceiver(context, receiver)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
	}
	return applySlice(
		context,
		source,
		argumentType,
		elementType,
		receiver,
	)
}

func applySlice(
	context api.Context,
	source ast.Node,
	argumentType types.Type,
	elementType types.Type,
	receiver api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	var err error
	receiver, err = slicevalue.Project(context, argumentType, receiver)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if context.Values().RequiresStructuralCopy(context, elementType) {
		return slicevalue.ClearAggregate(
			context,
			source,
			elementType,
			receiver,
		)
	}
	zero, err := context.Values().Zero(
		context.WithRole(api.RoleSliceElement),
		source,
		elementType,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if len(zero.Before()) != 0 {
		return api.ExpressionEmission{}, api.Unsupported(
			context,
			api.CategoryExpression,
			source,
		)
	}
	runtime, err := context.Names().Runtime(
		api.RuntimeSliceClear,
		api.ImportPhaseValue,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	before := append(receiver.Before(), zero.Before()...)
	return api.NewExpressionEmission(
		before,
		context.Factory().CallExpression(
			context.Factory().Identifier(runtime.Name()),
			nil,
			nil,
			[]tsgo.Expression{receiver.Value(), zero.Value()},
			tsgo.NodeFlagsNone,
		),
		api.CombineRequests(
			receiver.Requests(),
			zero.Requests(),
			runtime.Requests(),
		),
	)
}

func captureReceiver(
	context api.Context,
	receiver api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	name, err := context.Names().Temporary(api.TemporaryCallArgument)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	before := append(
		receiver.Before(),
		context.Factory().VariableStatement(
			nil,
			context.Factory().VariableDeclarationList(
				[]tsgo.VariableDeclaration{
					context.Factory().VariableDeclaration(
						context.Factory().Identifier(name),
						nil,
						nil,
						receiver.Value(),
					),
				},
				tsgo.NodeFlagsConst,
			),
		),
	)
	return api.NewExpressionEmission(
		before,
		context.Factory().Identifier(name),
		receiver.Requests(),
	)
}

func methodCall(
	context api.Context,
	receiver tsgo.Expression,
	name string,
	arguments ...tsgo.Expression,
) tsgo.CallExpression {
	return context.Factory().CallExpression(
		context.Factory().PropertyAccessExpression(
			receiver,
			nil,
			context.Factory().Identifier(name),
			tsgo.NodeFlagsNone,
		),
		nil,
		nil,
		arguments,
		tsgo.NodeFlagsNone,
	)
}
