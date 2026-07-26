package branch

import (
	"go/ast"
	"go/token"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func Emit(
	context api.Context,
	source *ast.BranchStmt,
) (api.StatementEmission, error) {
	if source.Label != nil {
		return api.StatementEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
	}
	switch source.Tok {
	case token.BREAK:
		if !context.CanBreak() {
			return api.StatementEmission{},
				api.Unsupported(context, api.CategoryStatement, source)
		}
		return api.DirectStatement(context.Factory().BreakStatement(nil)), nil
	case token.CONTINUE:
		if !context.CanContinue() {
			return api.StatementEmission{},
				api.Unsupported(context, api.CategoryStatement, source)
		}
		return api.DirectStatement(context.Factory().ContinueStatement(nil)), nil
	default:
		return api.StatementEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
	}
}
