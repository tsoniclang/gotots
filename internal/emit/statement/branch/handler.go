package branch

import (
	"go/ast"
	"go/token"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Emit(context api.Context, source *ast.BranchStmt) (tsgo.Statement, error) {
	if source.Label != nil {
		return nil, api.Unsupported(context, api.CategoryStatement, source)
	}
	switch source.Tok {
	case token.BREAK:
		if !context.CanBreak() {
			return nil, api.Unsupported(context, api.CategoryStatement, source)
		}
		return context.Factory().BreakStatement(nil), nil
	case token.CONTINUE:
		if !context.CanContinue() {
			return nil, api.Unsupported(context, api.CategoryStatement, source)
		}
		return context.Factory().ContinueStatement(nil), nil
	default:
		return nil, api.Unsupported(context, api.CategoryStatement, source)
	}
}
