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
) (api.StatementEmission, error) {
	if source.Init != nil || source.Body == nil {
		return api.StatementEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
	}
	condition, err := children.Condition(
		context.WithRole(api.RoleIfCondition),
		source.Cond,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	thenBlock, err := children.Block(
		context.WithRole(api.RoleIfThen),
		source.Body,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	var elseStatement tsgo.Statement
	var elseRequests []api.PlacementRequest
	switch alternate := source.Else.(type) {
	case nil:
	case *ast.BlockStmt:
		var alternateBlock api.BlockEmission
		alternateBlock, err = children.Block(
			context.WithRole(api.RoleIfElse),
			alternate,
		)
		if err == nil {
			elseStatement = alternateBlock.Value()
			elseRequests = alternateBlock.Requests()
		}
	default:
		return api.StatementEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
	}
	if err != nil {
		return api.StatementEmission{}, err
	}
	statements := condition.Before()
	statements = append(
		statements,
		context.Factory().IfStatement(
			condition.Value(),
			thenBlock.Value(),
			elseStatement,
		),
	)
	return api.NewStatementEmission(
		statements,
		api.CombineRequests(
			condition.Requests(),
			thenBlock.Requests(),
			elseRequests,
		),
	)
}
