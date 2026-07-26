package mapcomparison

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/value/maprepresentation"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.BinaryExpr,
) (api.ExpressionEmission, bool, error) {
	if source.Op != token.EQL && source.Op != token.NEQ {
		return api.ExpressionEmission{}, false, nil
	}
	mapSource, mapType, ok := operands(context, source)
	if !ok {
		return api.ExpressionEmission{}, false, nil
	}
	if context.ExpectedType() == nil ||
		!types.AssignableTo(
			context.TypesInfo().TypeOf(source),
			context.ExpectedType(),
		) {
		return api.ExpressionEmission{}, true,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	value, err := children.Expression(
		context.
			WithRole(api.RoleMapReceiver).
			WithExpectedType(mapType),
		mapSource,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	target := tsgo.Expression(context.Factory().CallExpression(
		context.Factory().PropertyAccessExpression(
			value.Value(),
			nil,
			context.Factory().Identifier("isNil"),
			tsgo.NodeFlagsNone,
		),
		nil,
		nil,
		nil,
		tsgo.NodeFlagsNone,
	))
	if source.Op == token.NEQ {
		target = context.Factory().PrefixUnaryExpression(
			tsgo.PrefixUnaryExpressionOperatorKindExclamationToken,
			target,
		)
	}
	result, err := api.NewExpressionEmission(
		value.Before(),
		target,
		value.Requests(),
	)
	return result, true, err
}

func operands(
	context api.Context,
	source *ast.BinaryExpr,
) (ast.Expr, *types.Map, bool) {
	if isNil(context, source.X) {
		mapType, ok := maprepresentation.Source(
			context,
			context.TypesInfo().TypeOf(source.Y),
		)
		return source.Y, mapType, ok
	}
	if isNil(context, source.Y) {
		mapType, ok := maprepresentation.Source(
			context,
			context.TypesInfo().TypeOf(source.X),
		)
		return source.X, mapType, ok
	}
	return nil, nil, false
}

func isNil(context api.Context, source ast.Expr) bool {
	identifier, ok := source.(*ast.Ident)
	return ok &&
		context.TypesInfo().Uses[identifier] == types.Universe.Lookup("nil")
}
