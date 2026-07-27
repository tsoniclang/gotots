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
	if source.Body == nil {
		return api.StatementEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
	}
	var initializer tsgo.ForInitializer
	var initializerRequests []api.RootRequest
	if source.Init != nil {
		target, err := children.ForInitializer(
			context.WithRole(api.RoleForInitializer),
			source.Init,
		)
		if err != nil {
			return api.StatementEmission{}, err
		}
		initializer = target.Value()
		initializerRequests = target.Requests()
	}
	var condition tsgo.Expression
	var conditionRequests []api.RootRequest
	if source.Cond != nil {
		target, err := children.Condition(
			context.WithRole(api.RoleForCondition),
			source.Cond,
		)
		if err != nil {
			return api.StatementEmission{}, err
		}
		if len(target.Before()) != 0 {
			return api.StatementEmission{},
				api.Unsupported(context, api.CategoryStatement, source)
		}
		condition = target.Value()
		conditionRequests = target.Requests()
	}
	var post tsgo.Expression
	var postRequests []api.RootRequest
	if source.Post != nil {
		target, err := children.ForPost(
			context.WithRole(api.RoleForPost),
			source.Post,
		)
		if err != nil {
			return api.StatementEmission{}, err
		}
		if len(target.Before()) != 0 {
			return api.StatementEmission{},
				api.Unsupported(context, api.CategoryStatement, source)
		}
		post = target.Value()
		postRequests = target.Requests()
	}
	body, err := children.Block(
		context.WithRole(api.RoleForBody).EnterLoop(),
		source.Body,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	return api.DirectStatement(
		context.Factory().ForStatement(
			initializer,
			condition,
			post,
			body.Value(),
		),
		api.CombineRequests(
			initializerRequests,
			conditionRequests,
			postRequests,
			body.Requests(),
		)...,
	), nil
}
