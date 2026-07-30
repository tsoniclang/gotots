package returnstatement

import (
	"go/ast"
	"strconv"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func emitControlled(
	context api.Context,
	children api.ChildEmitter,
	source *ast.ReturnStmt,
	control api.ReturnControl,
) (api.StatementEmission, error) {
	direct, err := emitDirect(
		context,
		children,
		source,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	statements, value, err := directReturnParts(context, source, direct)
	if err != nil {
		return api.StatementEmission{}, err
	}
	propagated, err := propagateControlled(
		context,
		source,
		control,
		value,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	statements = append(statements, propagated.Statements()...)
	return api.NewStatementEmission(
		statements,
		api.CombineRequests(
			direct.Requests(),
			propagated.Requests(),
		),
	)
}

func controlledAssignments(
	context api.Context,
	source ast.Node,
	targets []api.StoreTargetEmission,
	value tsgo.Expression,
) ([]tsgo.Statement, []api.RootRequest, error) {
	if len(targets) == 1 {
		return storeNamedResult(
			context,
			source,
			targets[0],
			value,
		)
	}
	if len(targets) == 0 {
		return nil, nil, &api.InvariantError{
			Role:   api.RoleReturnResult,
			Reason: "controlled named return has no result targets",
		}
	}
	name, err := context.Names().Temporary(api.TemporaryMultipleResults)
	if err != nil {
		return nil, nil, err
	}
	statements := []tsgo.Statement{
		context.Factory().VariableStatement(
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
		),
	}
	var requests []api.RootRequest
	for index, target := range targets {
		stores, storeRequests, err := storeNamedResult(
			context,
			source,
			target,
			context.Factory().ElementAccessExpression(
				context.Factory().Identifier(name),
				nil,
				context.Factory().NumericLiteral(
					strconv.Itoa(index),
					tsgo.TokenFlagsNone,
				),
				tsgo.NodeFlagsNone,
			),
		)
		if err != nil {
			return nil, nil, err
		}
		statements = append(statements, stores...)
		requests = append(requests, storeRequests...)
	}
	return statements, requests, nil
}

func storeNamedResult(
	context api.Context,
	source ast.Node,
	target api.StoreTargetEmission,
	value tsgo.Expression,
) ([]tsgo.Statement, []api.RootRequest, error) {
	stored, err := target.StoreValue(
		context.WithRole(api.RoleReturnResult),
		source,
		api.DirectExpression(value),
	)
	if err != nil {
		return nil, nil, err
	}
	statements := stored.Before()
	statements = append(
		statements,
		context.Factory().ExpressionStatement(stored.Value()),
	)
	return statements, stored.Requests(), nil
}

func propagateControlled(
	context api.Context,
	source ast.Node,
	control api.ReturnControl,
	value tsgo.Expression,
) (api.StatementEmission, error) {
	if !control.Valid() {
		return api.StatementEmission{}, &api.InvariantError{
			Role:   api.RoleReturnResult,
			Reason: "return control is invalid",
		}
	}
	var statements []tsgo.Statement
	var requests []api.RootRequest
	switch {
	case value == nil:
		if control.Named() || control.ResultTarget() != "" {
			return api.StatementEmission{},
				api.Unsupported(
					context,
					api.CategoryStatement,
					source,
				)
		}
	case control.Named():
		assignments, assignmentRequests, err := controlledAssignments(
			context,
			source,
			control.NamedTargets(),
			value,
		)
		if err != nil {
			return api.StatementEmission{}, err
		}
		statements = append(statements, assignments...)
		requests = append(requests, assignmentRequests...)
	case control.ResultTarget() != "":
		statements = append(
			statements,
			assignment(
				context,
				context.Factory().Identifier(control.ResultTarget()),
				value,
			),
		)
	default:
		return api.StatementEmission{},
			api.Unsupported(
				context,
				api.CategoryStatement,
				source,
			)
	}
	statements = append(
		statements,
		context.Factory().BreakStatement(
			context.Factory().Identifier(control.Label()),
		),
	)
	return api.NewStatementEmission(statements, requests)
}

func directReturnParts(
	context api.Context,
	source ast.Node,
	direct api.StatementEmission,
) ([]tsgo.Statement, tsgo.Expression, error) {
	statements := direct.Statements()
	if len(statements) == 0 {
		return nil, nil,
			api.Unsupported(context, api.CategoryStatement, source)
	}
	last := len(statements) - 1
	returnStatement, ok := statements[last].(tsgo.ReturnStatement)
	if !ok {
		return nil, nil,
			api.Unsupported(context, api.CategoryStatement, source)
	}
	return statements[:last], returnStatement.Expression(), nil
}

func assignment(
	context api.Context,
	target tsgo.Expression,
	value tsgo.Expression,
) tsgo.ExpressionStatement {
	return context.Factory().ExpressionStatement(
		context.Factory().BinaryExpression(
			nil,
			target,
			nil,
			context.Factory().BinaryOperatorToken(
				tsgo.BinaryOperatorEqualsToken,
			),
			value,
		),
	)
}
