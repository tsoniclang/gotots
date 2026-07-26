package mapstore

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/value/maprepresentation"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func EmitAssignment(
	context api.Context,
	children api.ChildEmitter,
	source *ast.AssignStmt,
) (api.StatementEmission, bool, error) {
	if source.Tok != token.ASSIGN ||
		len(source.Lhs) != 1 ||
		len(source.Rhs) != 1 {
		return api.StatementEmission{}, false, nil
	}
	index, ok := source.Lhs[0].(*ast.IndexExpr)
	if !ok {
		return api.StatementEmission{}, false, nil
	}
	mapType, ok := maprepresentation.Source(
		context,
		context.TypesInfo().TypeOf(index.X),
	)
	if !ok {
		return api.StatementEmission{}, false, nil
	}
	if !types.AssignableTo(
		context.TypesInfo().TypeOf(index.Index),
		mapType.Key(),
	) || !types.AssignableTo(
		context.TypesInfo().TypeOf(source.Rhs[0]),
		mapType.Elem(),
	) {
		return api.StatementEmission{}, true,
			api.Unsupported(context, api.CategoryStatement, source)
	}
	receiver, err := children.Expression(
		context.
			WithRole(api.RoleMapReceiver).
			WithExpectedType(mapType),
		index.X,
	)
	if err != nil {
		return api.StatementEmission{}, true, err
	}
	key, err := children.Expression(
		context.
			WithRole(api.RoleMapKey).
			WithExpectedType(mapType.Key()),
		index.Index,
	)
	if err != nil {
		return api.StatementEmission{}, true, err
	}
	value, err := children.Expression(
		context.
			WithRole(api.RoleMapValue).
			WithExpectedType(mapType.Elem()),
		source.Rhs[0],
	)
	if err != nil {
		return api.StatementEmission{}, true, err
	}
	value, err = context.Values().Copy(
		context.WithRole(api.RoleMapValue),
		source.Rhs[0],
		mapType.Elem(),
		value,
	)
	if err != nil {
		return api.StatementEmission{}, true, err
	}
	values, before, requests, err := maprepresentation.ArrangeOperands(
		context,
		[]api.ExpressionEmission{receiver, key, value},
	)
	if err != nil {
		return api.StatementEmission{}, true, err
	}
	before = append(
		before,
		context.Factory().ExpressionStatement(
			context.Factory().CallExpression(
				context.Factory().PropertyAccessExpression(
					values[0],
					nil,
					context.Factory().Identifier("store"),
					tsgo.NodeFlagsNone,
				),
				nil,
				nil,
				[]tsgo.Expression{values[1], values[2]},
				tsgo.NodeFlagsNone,
			),
		),
	)
	target, err := api.NewStatementEmission(before, requests)
	return target, true, err
}
