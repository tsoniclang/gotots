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
	if len(source.Results) == 0 {
		return api.DirectStatement(
			context.Factory().BreakStatement(
				context.Factory().Identifier(control.Label()),
			),
		), nil
	}
	direct, err := Emit(
		context.WithoutReturnControl(),
		children,
		source,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	statements := direct.Statements()
	if len(statements) == 0 {
		return api.StatementEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
	}
	last := len(statements) - 1
	returnStatement, ok := statements[last].(tsgo.ReturnStatement)
	if !ok || returnStatement.Expression() == nil {
		return api.StatementEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
	}
	statements = statements[:last]
	targets := control.NamedTargets()
	if control.Named() {
		assignments, assignmentRequests, err := controlledAssignments(
			context,
			source,
			targets,
			returnStatement.Expression(),
		)
		if err != nil {
			return api.StatementEmission{}, err
		}
		statements = append(statements, assignments...)
		direct, err = api.NewStatementEmission(
			direct.Statements(),
			api.CombineRequests(
				direct.Requests(),
				assignmentRequests,
			),
		)
		if err != nil {
			return api.StatementEmission{}, err
		}
	} else {
		if control.ResultTarget() == "" {
			return api.StatementEmission{},
				api.Unsupported(context, api.CategoryStatement, source)
		}
		statements = append(
			statements,
			assignment(
				context,
				context.Factory().Identifier(control.ResultTarget()),
				returnStatement.Expression(),
			),
		)
	}
	statements = append(
		statements,
		context.Factory().BreakStatement(
			context.Factory().Identifier(control.Label()),
		),
	)
	return api.NewStatementEmission(statements, direct.Requests())
}

func controlledAssignments(
	context api.Context,
	source *ast.ReturnStmt,
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
	source *ast.ReturnStmt,
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
