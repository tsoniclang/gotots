package returnstatement

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
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
	if resultCount == 1 {
		return emitSingle(context, children, source, results.At(0).Type())
	}
	if resultCount > 1 {
		return emitMultiple(context, children, source, results)
	}
	return api.StatementEmission{},
		api.Unsupported(context, api.CategoryStatement, source)
}

func emitSingle(
	context api.Context,
	children api.ChildEmitter,
	source *ast.ReturnStmt,
	resultType types.Type,
) (api.StatementEmission, error) {
	if len(source.Results) != 1 {
		return api.StatementEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
	}
	result, err := children.Expression(
		context.
			WithRole(api.RoleReturnResult).
			WithExpectedType(resultType),
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

func emitMultiple(
	context api.Context,
	children api.ChildEmitter,
	source *ast.ReturnStmt,
	results *types.Tuple,
) (api.StatementEmission, error) {
	if len(source.Results) == 1 {
		sourceType, ok := context.TypesInfo().TypeOf(source.Results[0]).(*types.Tuple)
		if !ok || !types.Identical(sourceType, results) {
			return api.StatementEmission{},
				api.Unsupported(context, api.CategoryStatement, source)
		}
		result, err := children.Expression(
			context.
				WithRole(api.RoleReturnResult).
				WithExpectedResults(results),
			source.Results[0],
		)
		if err != nil {
			return api.StatementEmission{}, err
		}
		statements := result.Before()
		statements = append(statements, context.Factory().ReturnStatement(result.Value()))
		return api.NewStatementEmission(statements, result.Requests())
	}
	if len(source.Results) != results.Len() {
		return api.StatementEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
	}

	values := make([]tsgo.Expression, 0, results.Len())
	var requests []api.PlacementRequest
	for index, sourceResult := range source.Results {
		sourceType := context.TypesInfo().TypeOf(sourceResult)
		if sourceType == nil || !types.AssignableTo(sourceType, results.At(index).Type()) {
			return api.StatementEmission{},
				api.Unsupported(context, api.CategoryStatement, source)
		}
		result, err := children.Expression(
			context.
				WithRole(api.RoleReturnResult).
				WithExpectedType(results.At(index).Type()),
			sourceResult,
		)
		if err != nil {
			return api.StatementEmission{}, err
		}
		result, err = context.Values().Copy(
			context.WithRole(api.RoleReturnResult),
			sourceResult,
			results.At(index).Type(),
			result,
		)
		if err != nil {
			return api.StatementEmission{}, err
		}
		if len(result.Before()) != 0 {
			return api.StatementEmission{},
				api.Unsupported(context, api.CategoryStatement, source)
		}
		values = append(values, result.Value())
		requests = append(requests, result.Requests()...)
	}
	return api.DirectStatement(
		context.Factory().ReturnStatement(
			context.Factory().ArrayLiteralExpression(values, false),
		),
		requests...,
	), nil
}
