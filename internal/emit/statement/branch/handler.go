package branch

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func Emit(
	context api.Context,
	source *ast.BranchStmt,
) (api.StatementEmission, error) {
	if source.Label != nil {
		label, ok := context.TypesInfo().Uses[source.Label].(*types.Label)
		if !ok {
			return api.StatementEmission{},
				api.Unsupported(context, api.CategoryStatement, source)
		}
		target, ok := context.ControlLabel(label)
		if !ok {
			return api.StatementEmission{},
				api.Unsupported(context, api.CategoryStatement, source)
		}
		targetLabel := context.Factory().Identifier(target.Name())
		switch source.Tok {
		case token.BREAK:
			if target.Breakable() {
				return api.DirectStatement(
					context.Factory().BreakStatement(targetLabel),
				), nil
			}
		case token.CONTINUE:
			if target.Continuable() {
				return api.DirectStatement(
					context.Factory().ContinueStatement(targetLabel),
				), nil
			}
		}
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
