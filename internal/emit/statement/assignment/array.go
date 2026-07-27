package assignment

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	arrayvalue "github.com/tsoniclang/gotots/internal/emit/value/array"
)

func emitArrayStore(
	context api.Context,
	children api.ChildEmitter,
	source *ast.AssignStmt,
	index *ast.IndexExpr,
) (api.StatementEmission, error) {
	array, ok := arrayvalue.Resolve(
		context,
		context.TypesInfo().TypeOf(index.X),
	)
	if !ok {
		return api.StatementEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
	}
	sourceType := context.TypesInfo().TypeOf(source.Rhs[0])
	if sourceType == nil || !types.AssignableTo(sourceType, array.ElementType()) {
		return api.StatementEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
	}
	value, err := children.Expression(
		context.
			WithRole(api.RoleAssignmentValue).
			WithExpectedType(array.ElementType()),
		source.Rhs[0],
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	value, err = context.Values().Copy(
		context.WithRole(api.RoleAssignmentValue),
		source.Rhs[0],
		array.ElementType(),
		value,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	stored, err := array.EmitStore(
		context.WithRole(api.RoleAssignmentTarget),
		children,
		index,
		value,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	statements := stored.Before()
	statements = append(
		statements,
		context.Factory().ExpressionStatement(stored.Value()),
	)
	return api.NewStatementEmission(statements, stored.Requests())
}
