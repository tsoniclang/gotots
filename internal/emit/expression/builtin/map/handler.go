package mapbuiltin

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	mapruntime "github.com/tsoniclang/gotots/internal/emit/runtime/map"
	"github.com/tsoniclang/gotots/internal/emit/value/maprepresentation"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
	discarded bool,
) (api.ExpressionEmission, bool, error) {
	identifier, ok := source.Fun.(*ast.Ident)
	if !ok {
		return api.ExpressionEmission{}, false, nil
	}
	object := context.TypesInfo().Uses[identifier]
	switch object {
	case types.Universe.Lookup("make"):
		if _, ok := maprepresentation.Source(
			context,
			context.TypesInfo().TypeOf(source),
		); !ok {
			return api.ExpressionEmission{}, false, nil
		}
		target, err := emitMake(context, children, source, discarded)
		return target, true, err
	case types.Universe.Lookup("len"):
		if !mapArgument(context, source, 0) {
			return api.ExpressionEmission{}, false, nil
		}
		target, err := emitLen(context, children, source, discarded)
		return target, true, err
	case types.Universe.Lookup("delete"):
		if !mapArgument(context, source, 0) {
			return api.ExpressionEmission{}, false, nil
		}
		target, err := emitDelete(context, children, source, discarded)
		return target, true, err
	default:
		return api.ExpressionEmission{}, false, nil
	}
}

func mapArgument(
	context api.Context,
	source *ast.CallExpr,
	index int,
) bool {
	if source == nil || index < 0 || index >= len(source.Args) {
		return false
	}
	_, ok := maprepresentation.Source(
		context,
		context.TypesInfo().TypeOf(source.Args[index]),
	)
	return ok
}

func emitMake(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
	discarded bool,
) (api.ExpressionEmission, error) {
	sourceType := context.TypesInfo().TypeOf(source)
	mapType, ok := maprepresentation.Source(context, sourceType)
	if discarded ||
		!ok ||
		source.Ellipsis.IsValid() ||
		len(source.Args) < 1 ||
		len(source.Args) > 2 ||
		context.ExpectedType() == nil ||
		!types.AssignableTo(sourceType, context.ExpectedType()) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	if _, ok := source.Args[0].(*ast.MapType); !ok ||
		!types.Identical(context.TypesInfo().TypeOf(source.Args[0]), sourceType) {
		return api.ExpressionEmission{},
			api.Unsupported(
				context.WithRole(api.RoleCallArgumentType),
				api.CategoryType,
				source.Args[0],
			)
	}
	size := api.DirectExpression(
		context.Factory().NumericLiteral("0", tsgo.TokenFlagsNone),
	)
	var err error
	if len(source.Args) == 2 {
		sizeType := context.TypesInfo().TypeOf(source.Args[1])
		if sizeType == nil {
			return api.ExpressionEmission{},
				api.Unsupported(
					context.WithRole(api.RoleMapSize),
					api.CategoryExpression,
					source.Args[1],
				)
		}
		size, err = children.Expression(
			context.
				WithRole(api.RoleMapSize).
				WithExpectedType(sizeType),
			source.Args[1],
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
	}
	zero, err := context.Values().Zero(
		context.WithRole(api.RoleMapValue),
		source,
		mapType.Elem(),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if len(zero.Before()) != 0 {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	target, requests, err := maprepresentation.Make(
		context,
		source,
		sourceType,
		zero.Value(),
		size.Value(),
		nil,
		size.Requests(),
		zero.Requests(),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.NewExpressionEmission(size.Before(), target, requests)
}

func emitLen(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
	discarded bool,
) (api.ExpressionEmission, error) {
	if discarded ||
		source.Ellipsis.IsValid() ||
		len(source.Args) != 1 ||
		context.ExpectedType() == nil {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	mapType, ok := maprepresentation.Source(
		context,
		context.TypesInfo().TypeOf(source.Args[0]),
	)
	resultType := context.TypesInfo().TypeOf(source)
	if !ok ||
		resultType == nil ||
		!types.AssignableTo(resultType, context.ExpectedType()) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	receiver, err := children.Expression(
		context.
			WithRole(api.RoleMapReceiver).
			WithExpectedType(mapType),
		source.Args[0],
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	lengthName, err := mapruntime.Name(mapruntime.MemberLength)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	target := tsgo.Expression(methodCall(
		context,
		receiver.Value(),
		lengthName,
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
	return api.NewExpressionEmission(
		receiver.Before(),
		target,
		receiver.Requests(),
	)
}

func emitDelete(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
	discarded bool,
) (api.ExpressionEmission, error) {
	if !discarded ||
		source.Ellipsis.IsValid() ||
		len(source.Args) != 2 {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	mapType, ok := maprepresentation.Source(
		context,
		context.TypesInfo().TypeOf(source.Args[0]),
	)
	if !ok ||
		!types.AssignableTo(
			context.TypesInfo().TypeOf(source.Args[1]),
			mapType.Key(),
		) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	receiver, err := children.Expression(
		context.
			WithRole(api.RoleMapReceiver).
			WithExpectedType(mapType),
		source.Args[0],
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	key, err := children.Expression(
		context.
			WithRole(api.RoleMapKey).
			WithExpectedType(mapType.Key()),
		source.Args[1],
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	values, before, requests, err := maprepresentation.ArrangeOperands(
		context,
		[]api.ExpressionEmission{receiver, key},
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	deleteName, err := mapruntime.Name(mapruntime.MemberDelete)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.NewExpressionEmission(
		before,
		methodCall(context, values[0], deleteName, values[1]),
		requests,
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
