package ifstatement

import (
	"go/ast"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.IfStmt,
) (tsgo.IfStatement, error) {
	if source.Init != nil || source.Body == nil {
		return nil, api.Unsupported(context, api.CategoryStatement, source)
	}
	condition, err := children.Condition(
		context.WithRole(api.RoleIfCondition),
		source.Cond,
	)
	if err != nil {
		return nil, err
	}
	thenBlock, err := children.Block(
		context.WithRole(api.RoleIfThen),
		source.Body,
	)
	if err != nil {
		return nil, err
	}
	var elseStatement tsgo.Statement
	switch alternate := source.Else.(type) {
	case nil:
	case *ast.BlockStmt:
		elseStatement, err = children.Block(
			context.WithRole(api.RoleIfElse),
			alternate,
		)
	default:
		return nil, api.Unsupported(context, api.CategoryStatement, source)
	}
	if err != nil {
		return nil, err
	}
	return context.Factory().IfStatement(
		condition,
		thenBlock,
		elseStatement,
	), nil
}
