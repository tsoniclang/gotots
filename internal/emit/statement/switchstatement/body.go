package switchstatement

import (
	"go/ast"
	"go/token"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func emitClauseBody(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CaseClause,
	allowFallthrough bool,
	targetLabel string,
	fallthroughLowering bool,
) ([]api.StatementEmission, bool, error) {
	if source == nil {
		return nil, false,
			api.Unsupported(context, api.CategoryStatement, source)
	}
	fallthroughIndex := finalFallthrough(source.Body)
	if fallthroughIndex >= 0 && !allowFallthrough {
		return nil, false, api.Unsupported(
			context.WithRole(api.RoleSwitchCaseStatement),
			api.CategoryStatement,
			source.Body[fallthroughIndex],
		)
	}
	statements := make([]ast.Stmt, 0, len(source.Body))
	for index, sourceStatement := range source.Body {
		if index == fallthroughIndex {
			continue
		}
		statements = append(statements, sourceStatement)
	}
	bodyContext := context.WithRole(api.RoleSwitchCaseStatement)
	if context.CallableControl().Goto() || fallthroughLowering {
		bodyContext = context.
			WithRole(api.RoleSwitchCaseStatement).
			EnterBreakableTarget(targetLabel)
	} else {
		bodyContext = bodyContext.EnterBreakable()
	}
	target, err := children.Statements(bodyContext, source, statements)
	if err != nil {
		return nil, false, err
	}
	return []api.StatementEmission{target}, fallthroughIndex >= 0, nil
}

func requiresFallthroughLowering(source *ast.SwitchStmt) bool {
	if source == nil || source.Body == nil {
		return false
	}
	for _, node := range source.Body.List {
		clause, ok := node.(*ast.CaseClause)
		if ok && finalFallthrough(clause.Body) >= 0 {
			return true
		}
	}
	return false
}

func finalFallthrough(source []ast.Stmt) int {
	for index := len(source) - 1; index >= 0; index-- {
		if _, empty := source[index].(*ast.EmptyStmt); empty {
			continue
		}
		branch, ok := source[index].(*ast.BranchStmt)
		if ok && branch.Tok == token.FALLTHROUGH && branch.Label == nil {
			return index
		}
		return -1
	}
	return -1
}
