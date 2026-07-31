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
	directPost, postDirect := directPostExpression(post)
	if len(condition.Before()) == 0 && postDirect {
		loop := context.Factory().ForStatement(
			nil,
			condition.Value(),
			directPost,
			body.Value(),
		)
		statements = append(statements, labeled(context, targetLabel, loop))
		return structuredResult(
			context,
			statements,
			initializer,
			condition,
			post,
			body,
		), nil
	}
	loopStatements, declarations, err := loweredLoopStatements(
		context,
		source,
		condition,
		post,
		body,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	statements = append(statements, declarations...)
	loop := context.Factory().ForStatement(
		nil,
		nil,
		nil,
		context.Factory().Block(loopStatements, true),
	)
	statements = append(statements, labeled(context, targetLabel, loop))
	return structuredResult(
		context,
		statements,
		initializer,
		condition,
		post,
		body,
	), nil
}

func loweredLoopStatements(
	context api.Context,
	source *ast.ForStmt,
	condition api.ExpressionEmission,
	post api.StatementEmission,
	body api.BlockEmission,
) ([]tsgo.Statement, []tsgo.Statement, error) {
	var statements []tsgo.Statement
	var declarations []tsgo.Statement
	if source.Post != nil {
		firstName, err := context.Names().Temporary(
			api.TemporaryForFirstIteration,
		)
		if err != nil {
			return nil, nil, err
		}
		declarations = append(
			declarations,
			context.Factory().VariableStatement(
				nil,
				context.Factory().VariableDeclarationList(
					[]tsgo.VariableDeclaration{
						context.Factory().VariableDeclaration(
							context.Factory().Identifier(firstName),
							nil,
							nil,
							context.Factory().TrueLiteral(),
						),
					},
					tsgo.NodeFlagsLet,
				),
			),
		)
		statements = append(
			statements,
			context.Factory().IfStatement(
				context.Factory().Identifier(firstName),
				context.Factory().Block(
					[]tsgo.Statement{
						context.Factory().ExpressionStatement(
							context.Factory().BinaryExpression(
								nil,
								context.Factory().Identifier(firstName),
								nil,
								context.Factory().BinaryOperatorToken(
									tsgo.BinaryOperatorEqualsToken,
								),
								context.Factory().FalseLiteral(),
							),
						),
					},
					true,
				),
				context.Factory().Block(post.Statements(), true),
			),
		)
	}
	if source.Cond != nil {
		statements = append(statements, condition.Before()...)
		statements = append(
			statements,
			context.Factory().IfStatement(
				context.Factory().PrefixUnaryExpression(
					tsgo.PrefixUnaryExpressionOperatorKindExclamationToken,
					condition.Value(),
				),
				context.Factory().Block(
					[]tsgo.Statement{
						context.Factory().BreakStatement(nil),
					},
					true,
				),
				nil,
			),
		)
	}
	statements = append(statements, body.Value())
	return statements, declarations, nil
}

func structuredResult(
	context api.Context,
	statements []tsgo.Statement,
	initializer api.StatementEmission,
	condition api.ExpressionEmission,
	post api.StatementEmission,
	body api.BlockEmission,
) api.StatementEmission {
	return api.DirectStatement(
		context.Factory().Block(statements, true),
		api.CombineRequests(
			initializer.Requests(),
			condition.Requests(),
			post.Requests(),
			body.Requests(),
		)...,
	)
}
