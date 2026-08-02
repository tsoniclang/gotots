package branch

import (
	"go/ast"
	"go/token"
	"go/types"
	"strconv"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Emit(
	context api.Context,
	source *ast.BranchStmt,
) (api.StatementEmission, error) {
	if source.Label != nil {
		label, ok := context.TypesInfo().UseOf(source.Label).(*types.Label)
		if !ok {
			return api.StatementEmission{},
				api.Unsupported(context, api.CategoryStatement, source)
		}
		if source.Tok == token.GOTO {
			if target, selected := context.GotoTarget(label); selected {
				return emitGoto(context, target)
			}
			if !context.CallableControl().Goto() {
				request, err := context.GotoControlRequest(
					label,
					source.Label.Pos(),
				)
				if err != nil {
					return api.StatementEmission{}, err
				}
				return api.NewStatementEmission(
					nil,
					[]api.RootRequest{request},
				)
			}
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
				context.Factory().BreakStatement(
					implicitTarget(context, context.BreakTarget()),
				),
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
				context.Factory().ContinueStatement(
					implicitTarget(context, context.ContinueTarget()),
				),
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
	default:
		return api.StatementEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
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

func implicitTarget(
	context api.Context,
	name string,
) tsgo.Identifier {
	if name == "" {
		return nil
	}
	return context.Factory().Identifier(name)
}

func emitGoto(
	context api.Context,
	target api.GotoTarget,
) (api.StatementEmission, error) {
	label := context.Factory().Identifier(target.Label())
	switch target.Kind() {
	case api.GotoTargetBreak:
		return api.DirectStatement(
			context.Factory().BreakStatement(label),
		), nil
	case api.GotoTargetContinue:
		return api.DirectStatement(
			context.Factory().ContinueStatement(label),
		), nil
	case api.GotoTargetState:
		return api.NewStatementEmission(
			[]tsgo.Statement{
				context.Factory().ExpressionStatement(
					context.Factory().BinaryExpression(
						nil,
						context.Factory().Identifier(
							target.StateName(),
						),
						nil,
						context.Factory().BinaryOperatorToken(
							tsgo.BinaryOperatorEqualsToken,
						),
						context.Factory().NumericLiteral(
							strconv.Itoa(target.State()),
							tsgo.TokenFlagsNone,
						),
					),
				),
				context.Factory().ContinueStatement(label),
			},
			nil,
		)
	default:
		return api.StatementEmission{}, &api.InvariantError{
			Role:   api.RoleLabelTarget,
			Reason: "goto target kind is invalid",
		}
	}
}
