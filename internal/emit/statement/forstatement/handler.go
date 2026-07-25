package forstatement

import (
	"go/ast"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.ForStmt,
) (api.StatementEmission, error) {
	if source.Init == nil || source.Cond == nil || source.Post == nil || source.Body == nil {
		return api.StatementEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
	}
	initializer, err := children.ForInitializer(
		context.WithRole(api.RoleForInitializer),
		source.Init,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	condition, err := children.Condition(
		context.WithRole(api.RoleForCondition),
		source.Cond,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	post, err := children.ForPost(
		context.WithRole(api.RoleForPost),
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
	if len(condition.Before()) != 0 || len(post.Before()) != 0 {
		return api.StatementEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
	}
	return api.DirectStatement(
		context.Factory().ForStatement(
			initializer.Value(),
			condition.Value(),
			post.Value(),
			body.Value(),
		),
		api.CombineRequests(
			initializer.Requests(),
			condition.Requests(),
			post.Requests(),
			body.Requests(),
		)...,
	), nil
}
