package switchstatement

import (
	"go/ast"
	"go/token"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func emitClauseBody(
	context api.Context,
	children api.ChildEmitter,
	source []ast.Stmt,
	allowFallthrough bool,
) ([]api.StatementEmission, bool, error) {
	fallthroughIndex := finalFallthrough(source)
	if fallthroughIndex >= 0 && !allowFallthrough {
		return nil, false, api.Unsupported(
			context.WithRole(api.RoleSwitchCaseStatement),
			api.CategoryStatement,
			source[fallthroughIndex],
		)
	}
	targets := make([]api.StatementEmission, 0, len(source))
	for index, sourceStatement := range source {
		if index == fallthroughIndex {
			continue
		}
		target, err := children.Statement(
			context.
				WithRole(api.RoleSwitchCaseStatement).
				EnterBreakable(),
			sourceStatement,
		)
		if err != nil {
			return nil, false, err
		}
		targets = append(targets, target)
	}
	return targets, fallthroughIndex >= 0, nil
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
