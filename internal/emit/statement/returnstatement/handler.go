package returnstatement

import (
	"go/ast"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.ReturnStmt,
) (api.StatementEmission, error) {
	results := context.FunctionResults()
	resultCount := 0
	if results != nil {
		resultCount = results.Len()
	}
	if resultCount == 0 && len(source.Results) == 0 {
		return api.DirectStatement(context.Factory().ReturnStatement(nil)), nil
	}
	if len(source.Results) != 1 || resultCount != 1 {
		return api.StatementEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
	}
	result, err := children.Expression(
		context.
			WithRole(api.RoleReturnResult).
			WithExpectedType(results.At(0).Type()),
		source.Results[0],
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	statements := result.Before()
	statements = append(
		statements,
		context.Factory().ReturnStatement(result.Value()),
	)
	return api.NewStatementEmission(statements, result.Requests())
}
