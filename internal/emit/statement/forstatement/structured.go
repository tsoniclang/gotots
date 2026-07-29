package forstatement

import (
	"go/ast"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func emitStructured(
	context api.Context,
	source *ast.ForStmt,
	initializer api.StatementEmission,
	condition api.ExpressionEmission,
	post api.StatementEmission,
	body api.BlockEmission,
	targetLabel string,
) (api.StatementEmission, error) {
	statements := initializer.Statements()
	var targetCondition tsgo.Expression
	if source.Cond != nil {
		targetCondition = condition.Value()
		if len(condition.Before()) != 0 {
			name, declaration, err := clauseFunction(
				context,
				api.TemporaryForCondition,
				append(
					condition.Before(),
					context.Factory().ReturnStatement(condition.Value()),
				),
			)
			if err != nil {
				return api.StatementEmission{}, err
			}
			statements = append(statements, declaration)
			targetCondition = callClauseFunction(context, name)
		}
	}
	var postExpression tsgo.Expression
	if source.Post != nil {
		if direct, ok := directPostExpression(post); ok {
			postExpression = direct
		} else {
			postStatements := post.Statements()
			name, declaration, err := clauseFunction(
				context,
				api.TemporaryForPost,
				postStatements,
			)
			if err != nil {
				return api.StatementEmission{}, err
			}
			statements = append(statements, declaration)
			postExpression = callClauseFunction(context, name)
		}
	}
	loop := context.Factory().ForStatement(
		nil,
		targetCondition,
		postExpression,
		body.Value(),
	)
	statements = append(statements, labeled(context, targetLabel, loop))
	return api.DirectStatement(
		context.Factory().Block(statements, true),
		api.CombineRequests(
			initializer.Requests(),
			condition.Requests(),
			post.Requests(),
			body.Requests(),
		)...,
	), nil
}

func clauseFunction(
	context api.Context,
	kind api.TemporaryKind,
	body []tsgo.Statement,
) (string, tsgo.VariableStatement, error) {
	name, err := context.Names().Temporary(kind)
	if err != nil {
		return "", nil, err
	}
	return name, context.Factory().VariableStatement(
		nil,
		context.Factory().VariableDeclarationList(
			[]tsgo.VariableDeclaration{
				context.Factory().VariableDeclaration(
					context.Factory().Identifier(name),
					nil,
					nil,
					context.Factory().ArrowFunction(
						nil,
						nil,
						nil,
						nil,
						context.Factory().EqualsGreaterThanToken(),
						context.Factory().Block(body, true),
					),
				),
			},
			tsgo.NodeFlagsConst,
		),
	), nil
}

func callClauseFunction(
	context api.Context,
	name string,
) tsgo.Expression {
	return context.Factory().CallExpression(
		context.Factory().Identifier(name),
		nil,
		nil,
		nil,
		tsgo.NodeFlagsNone,
	)
}
