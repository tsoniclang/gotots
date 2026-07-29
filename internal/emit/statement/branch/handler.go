package branch

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
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
		if context.CanBreak() {
			return api.DirectStatement(
				context.Factory().BreakStatement(nil),
			), nil
		}
		if control, ok := context.IteratorRangeControl(); ok {
			return iteratorBranch(
				context,
				control,
				api.IteratorRangeStateDone,
				false,
			)
		}
	case token.CONTINUE:
		if context.CanContinue() {
			return api.DirectStatement(
				context.Factory().ContinueStatement(nil),
			), nil
		}
		if control, ok := context.IteratorRangeControl(); ok {
			return iteratorBranch(
				context,
				control,
				api.IteratorRangeStateReady,
				true,
			)
		}
	}
	return api.StatementEmission{},
		api.Unsupported(context, api.CategoryStatement, source)
}

func iteratorBranch(
	context api.Context,
	control api.IteratorRangeControl,
	state api.IteratorRangeState,
	result bool,
) (api.StatementEmission, error) {
	if !control.Valid() || state.Literal() == "" {
		return api.StatementEmission{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "iterator-range branch control is invalid",
		}
	}
	resultValue := tsgo.Expression(context.Factory().FalseLiteral())
	if result {
		resultValue = context.Factory().TrueLiteral()
	}
	return api.NewStatementEmission(
		[]tsgo.Statement{
			context.Factory().ExpressionStatement(
				context.Factory().BinaryExpression(
					nil,
					context.Factory().Identifier(control.StateName()),
					nil,
					context.Factory().BinaryOperatorToken(
						tsgo.BinaryOperatorEqualsToken,
					),
					context.Factory().NumericLiteral(
						state.Literal(),
						tsgo.TokenFlagsNone,
					),
				),
			),
			context.Factory().ReturnStatement(resultValue),
		},
		nil,
	)
}
