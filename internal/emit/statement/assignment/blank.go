package assignment

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func emitBlankAssignment(
	context api.Context,
	children api.ChildEmitter,
	source *ast.AssignStmt,
) (api.StatementEmission, error) {
	sourceValue := source.Rhs[0]
	sourceType := context.TypesInfo().TypeOf(sourceValue)
	if sourceType == nil {
		return api.StatementEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
	}
	if facts, ok := context.TypesInfo().TypeAndValue(sourceValue); ok &&
		facts.Value != nil {
		return api.NewStatementEmission(nil, nil)
	}
	expectedType := sourceType
	if basic, ok := types.Unalias(sourceType).(*types.Basic); ok &&
		basic.Info()&types.IsUntyped != 0 {
		expectedType = types.Default(sourceType)
	}
	value, err := children.Expression(
		context.
			WithRole(api.RoleAssignmentValue).
			WithExpectedType(expectedType),
		sourceValue,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	statements := value.Before()
	statements = append(
		statements,
		context.Factory().ExpressionStatement(value.Value()),
	)
	return api.NewStatementEmission(statements, value.Requests())
}
