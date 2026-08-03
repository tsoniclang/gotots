package returnstatement

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/resulttuple"
	arrayvalue "github.com/tsoniclang/gotots/internal/emit/value/array"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.ReturnStmt,
) (api.StatementEmission, error) {
	if control, selected := context.IteratorRangeControl(); selected {
		requests, err := context.IteratorReturnControlRequests()
		if err != nil {
			return api.StatementEmission{}, err
		}
		if !control.Returning() {
			return api.NewStatementEmission(nil, requests)
		}
		result, err := emitIteratorReturn(
			context,
			children,
			source,
			control,
		)
		if err != nil {
			return api.StatementEmission{}, err
		}
		return api.NewStatementEmission(
			result.Statements(),
			api.CombineRequests(result.Requests(), requests),
		)
	}
	if control, selected := context.ReturnControl(); selected {
		return emitControlled(context, children, source, control)
	}
	return emitDirect(context, children, source)
}

func emitDirect(
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
		value, selected, err := context.AddressableStorage().Read(
			context,
			result,
		)
		if err != nil {
			return api.StatementEmission{}, err
		}
		if !selected {
			targetName, err := context.Names().Result(result, index)
			if err != nil {
				return api.StatementEmission{}, err
			}
			value = api.DirectExpression(
				context.Factory().Identifier(targetName),
			)
		}
		value, err = context.Values().Transfer(
			context.WithRole(api.RoleReturnResult),
			source,
			result.Type(),
			result.Type(),
			api.ValueTransferCopy,
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
	sourceType := context.TypesInfo().TypeOf(source.Results[0])
	mode := api.ValueTransferRepresentation
	if _, copied := arrayvalue.Resolve(context, resultType); copied {
		mode = api.ValueTransferCopy
	}
	result, err = context.Values().Transfer(
		context.WithRole(api.RoleReturnResult),
		source.Results[0],
		sourceType,
		resultType,
		mode,
		result,
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
		sourceResults, ok := context.TypesInfo().TypeOf(source.Results[0]).(*types.Tuple)
		if !ok || sourceResults.Len() != results.Len() {
			return api.StatementEmission{},
				api.Unsupported(context, api.CategoryStatement, source)
		}
		for index := range results.Len() {
			if !types.AssignableTo(
				sourceResults.At(index).Type(),
				results.At(index).Type(),
			) {
				return api.StatementEmission{},
					api.Unsupported(context, api.CategoryStatement, source)
			}
		}
		if !types.Identical(sourceResults, results) {
			return emitAdaptedMultiple(
				context,
				children,
				source,
				sourceResults,
				results,
			)
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
	emissions := make([]api.ExpressionEmission, 0, results.Len())
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
		result, err = context.Values().Transfer(
			context.WithRole(api.RoleReturnResult),
			sourceResult,
			sourceType,
			results.At(index).Type(),
			api.ValueTransferCopy,
			result,
		)
		if err != nil {
			return api.StatementEmission{}, err
		}
		emissions = append(emissions, result)
		requests = append(requests, result.Requests()...)
	}
	hasPrerequisites := false
	for _, result := range emissions {
		if len(result.Before()) != 0 {
			hasPrerequisites = true
			break
		}
	}
	var statements []tsgo.Statement
	for _, result := range emissions {
		if !hasPrerequisites {
			values = append(values, result.Value())
			continue
		}
		statements = append(statements, result.Before()...)
		name, err := context.Names().Temporary(api.TemporaryMultipleResults)
		if err != nil {
			return api.StatementEmission{}, err
		}
		statements = append(
			statements,
			capturedResultStatement(context, name, result.Value()),
		)
		values = append(values, context.Factory().Identifier(name))
	}
	statements = append(
		statements,
		context.Factory().ReturnStatement(
			context.Factory().ArrayLiteralExpression(values, false),
		),
	)
	return api.NewStatementEmission(statements, requests)
}

func emitAdaptedMultiple(
	context api.Context,
	children api.ChildEmitter,
	source *ast.ReturnStmt,
	sourceResults *types.Tuple,
	targetResults *types.Tuple,
) (api.StatementEmission, error) {
	capture, err := resulttuple.Emit(
		context,
		children,
		source.Results[0],
		sourceResults,
		api.RoleReturnResult,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	statements := capture.Statements()
	requests := capture.Requests()
	values := make([]tsgo.Expression, 0, targetResults.Len())
	for index := range targetResults.Len() {
		element, err := capture.Element(context, index)
		if err != nil {
			return api.StatementEmission{}, err
		}
		targetType := targetResults.At(index).Type()
		value, err := context.Values().Transfer(
			context.WithRole(api.RoleReturnResult),
			source.Results[0],
			sourceResults.At(index).Type(),
			targetType,
			api.ValueTransferCopy,
			api.DirectExpression(element),
		)
		if err != nil {
			return api.StatementEmission{}, err
		}
		statements = append(statements, value.Before()...)
		values = append(values, value.Value())
		requests = append(requests, value.Requests()...)
	}
	statements = append(
		statements,
		context.Factory().ReturnStatement(
			context.Factory().ArrayLiteralExpression(values, false),
		),
	)
	return api.NewStatementEmission(statements, requests)
}

func capturedResultStatement(
	context api.Context,
	name string,
	value tsgo.Expression,
) tsgo.VariableStatement {
	return context.Factory().VariableStatement(
		nil,
		context.Factory().VariableDeclarationList(
			[]tsgo.VariableDeclaration{
				context.Factory().VariableDeclaration(
					context.Factory().Identifier(name),
					nil,
					nil,
					value,
				),
			},
			tsgo.NodeFlagsConst,
		),
	)
}
