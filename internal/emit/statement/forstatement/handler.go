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
) (tsgo.ForStatement, error) {
	if source.Init == nil || source.Cond == nil || source.Post == nil || source.Body == nil {
		return nil, api.Unsupported(context, api.CategoryStatement, source)
	}
	initializer, err := children.ForInitializer(
		context.WithRole(api.RoleForInitializer),
		source.Init,
	)
	if err != nil {
		return nil, err
	}
	condition, err := children.Condition(
		context.WithRole(api.RoleForCondition),
		source.Cond,
	)
	if err != nil {
		return nil, err
	}
	post, err := children.ForPost(
		context.WithRole(api.RoleForPost),
		source.Post,
	)
	if err != nil {
		return nil, err
	}
	body, err := children.Block(
		context.WithRole(api.RoleForBody).EnterLoop(),
		source.Body,
	)
	if err != nil {
		return nil, err
	}
	return context.Factory().ForStatement(
		initializer,
		condition,
		post,
		body,
	), nil
}
