package forstatement

import (
	"go/ast"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.ForStmt,
) (api.StatementEmission, error) {
	context, targetLabel := context.TakeStatementLabel()
	if source == nil || source.Body == nil {
		return api.StatementEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
	}
	initializer, err := emitClauseStatement(
		context.WithRole(api.RoleForInitializer),
		children,
		source.Init,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	condition, err := emitCondition(context, children, source)
	if err != nil {
		return api.StatementEmission{}, err
	}
	post, err := emitClauseStatement(
		context.WithRole(api.RoleForPost),
		children,
		source.Post,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	body, err := children.Block(
		context.WithRole(api.RoleForBody).EnterLoop(),
		source.Body,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	directInitializer, initializerDirect := targetInitializer(initializer)
	directPost, postDirect := directPostExpression(post)
	structured := source.Init != nil && !initializerDirect ||
		source.Post != nil && !postDirect ||
		len(condition.Before()) != 0
	if !structured {
		target := labeled(
			context,
			targetLabel,
			context.Factory().ForStatement(
				directInitializer,
				condition.Value(),
				directPost,
				body.Value(),
			),
		)
		return api.DirectStatement(
			target,
			api.CombineRequests(
				initializer.Requests(),
				condition.Requests(),
				post.Requests(),
				body.Requests(),
			)...,
		), nil
	}
	return emitStructured(
		context,
		source,
		initializer,
		condition,
		post,
		body,
		targetLabel,
	)
}

func emitClauseStatement(
	context api.Context,
	children api.ChildEmitter,
	source ast.Stmt,
) (api.StatementEmission, error) {
	if source == nil {
		return api.NewStatementEmission(nil, nil)
	}
	return children.ScopedInitializer(context, source)
}

func emitCondition(
	context api.Context,
	children api.ChildEmitter,
	source *ast.ForStmt,
) (api.ExpressionEmission, error) {
	if source.Cond == nil {
		return api.ExpressionEmission{}, nil
	}
	return children.Condition(
		context.WithRole(api.RoleForCondition),
		source.Cond,
	)
}

func targetInitializer(
	source api.StatementEmission,
) (tsgo.ForInitializer, bool) {
	statements := source.Statements()
	if len(statements) == 0 {
		return nil, true
	}
	if len(statements) != 1 {
		return nil, false
	}
	switch statement := statements[0].(type) {
	case tsgo.ExpressionStatement:
		target, ok := statement.Expression().(tsgo.ForInitializer)
		return target, ok
	case tsgo.VariableStatement:
		return statement.DeclarationList(), true
	default:
		return nil, false
	}
}

func directPostExpression(
	source api.StatementEmission,
) (tsgo.Expression, bool) {
	statements := source.Statements()
	if len(statements) == 0 {
		return nil, true
	}
	if len(statements) != 1 {
		return nil, false
	}
	statement, ok := statements[0].(tsgo.ExpressionStatement)
	if !ok {
		return nil, false
	}
	return statement.Expression(), true
}

func labeled(
	context api.Context,
	name string,
	statement tsgo.Statement,
) tsgo.Statement {
	if name == "" {
		return statement
	}
	return context.Factory().LabeledStatement(
		context.Factory().Identifier(name),
		statement,
	)
}
