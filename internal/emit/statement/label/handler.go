package label

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.LabeledStmt,
) (api.StatementEmission, error) {
	if source == nil || source.Label == nil || source.Stmt == nil {
		return api.StatementEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
	}
	label, ok := context.TypesInfo().DefOf(source.Label).(*types.Label)
	if !ok {
		return api.StatementEmission{},
			api.Unsupported(context, api.CategoryStatement, source.Label)
	}
	name, err := context.Names().Declare(label)
	if err != nil {
		return api.StatementEmission{}, err
	}
	breakable, continuable := targetCapabilities(source.Stmt)
	target, err := api.NewControlLabel(name, breakable, continuable)
	if err != nil {
		return api.StatementEmission{}, err
	}
	childContext := context.
		WithRole(api.RoleLabelTarget).
		WithControlLabel(label, target)
	if breakable {
		childContext = childContext.WithStatementLabel(name)
	}
	emission, err := children.Statement(
		childContext,
		source.Stmt,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	if breakable {
		return emission, nil
	}
	statements := emission.Statements()
	if len(statements) == 0 {
		statements = []tsgo.Statement{context.Factory().EmptyStatement()}
	}
	last := len(statements) - 1
	statements[last] = context.Factory().LabeledStatement(
		context.Factory().Identifier(name),
		statements[last],
	)
	return api.NewStatementEmission(statements, emission.Requests())
}

func targetCapabilities(source ast.Stmt) (bool, bool) {
	switch source.(type) {
	case *ast.ForStmt, *ast.RangeStmt:
		return true, true
	case *ast.SwitchStmt, *ast.TypeSwitchStmt, *ast.SelectStmt:
		return true, false
	default:
		return false, false
	}
}
