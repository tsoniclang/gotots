package returnstatement

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	arrayvalue "github.com/tsoniclang/gotots/internal/emit/value/array"
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
	if resultCount > 0 && len(source.Results) == 0 {
		return emitNamed(context, source, results)
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

func emitNamed(
	context api.Context,
	source *ast.ReturnStmt,
	results *types.Tuple,
) (api.StatementEmission, error) {
	values := make([]tsgo.Expression, 0, results.Len())
	var before []tsgo.Statement
	var requests []api.RootRequest
	for index := range results.Len() {
		result := results.At(index)
		if result.Name() == "" {
			return api.StatementEmission{},
				api.Unsupported(context, api.CategoryStatement, source)
		}
		value, selected := context.AddressableStorage().Read(
			context,
			result,
		)
		if !selected {
			reference, err := context.Names().Reference(result)
			if err != nil {
				return api.StatementEmission{}, err
			}
			value = api.DirectExpression(
				context.Factory().Identifier(reference.Name()),
				reference.Requests()...,
			)
		}
		value, err := context.Values().Copy(
			context.WithRole(api.RoleReturnResult),
			source,
			result.Type(),
			value,
		)
		if err != nil {
			return api.StatementEmission{}, err
		}
		before = append(before, value.Before()...)
		values = append(values, value.Value())
		requests = append(requests, value.Requests()...)
	}
	var result tsgo.Expression
	if len(values) == 1 {
		result = values[0]
	} else {
		result = context.Factory().ArrayLiteralExpression(values, false)
	}
	before = append(before, context.Factory().ReturnStatement(result))
	return api.NewStatementEmission(before, requests)
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
	if _, ok := arrayvalue.Resolve(context, resultType); ok {
		result, err = context.Values().Copy(
			context.WithRole(api.RoleReturnResult),
			source.Results[0],
			resultType,
			result,
		)
		if err != nil {
			return api.StatementEmission{}, err
		}
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
	var requests []api.RootRequest
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
