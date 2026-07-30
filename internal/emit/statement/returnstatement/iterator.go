package returnstatement

import (
	"go/ast"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func emitIteratorReturn(
	context api.Context,
	children api.ChildEmitter,
	source *ast.ReturnStmt,
	control api.IteratorRangeControl,
) (api.StatementEmission, error) {
	direct, err := emitDirect(context, children, source)
	if err != nil {
		return api.StatementEmission{}, err
	}
	statements, value, err := directReturnParts(context, source, direct)
	if err != nil {
		return api.StatementEmission{}, err
	}
	propagated, err := propagateIterator(context, source, control, value)
	if err != nil {
		return api.StatementEmission{}, err
	}
	statements = append(statements, propagated.Statements()...)
	return api.NewStatementEmission(
		statements,
		api.CombineRequests(
			direct.Requests(),
			propagated.Requests(),
		),
	)
}

func Propagate(
	context api.Context,
	source ast.Node,
	value tsgo.Expression,
) (api.StatementEmission, error) {
	if control, selected := context.IteratorRangeControl(); selected {
		return propagateIterator(context, source, control, value)
	}
	if control, selected := context.ReturnControl(); selected {
		return propagateControlled(context, source, control, value)
	}
	return api.DirectStatement(
		context.Factory().ReturnStatement(value),
	), nil
}

func propagateIterator(
	context api.Context,
	source ast.Node,
	control api.IteratorRangeControl,
	value tsgo.Expression,
) (api.StatementEmission, error) {
	if !control.Valid() ||
		(value == nil) != (control.ResultName() == "") {
		return api.StatementEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
	}
	var statements []tsgo.Statement
	if value != nil {
		statements = append(
			statements,
			context.Factory().ExpressionStatement(
				context.Factory().BinaryExpression(
					nil,
					context.Factory().Identifier(control.ResultName()),
					nil,
					context.Factory().BinaryOperatorToken(
						tsgo.BinaryOperatorEqualsToken,
					),
					value,
				),
			),
		)
	}
	statements = append(
		statements,
		context.Factory().ExpressionStatement(
			context.Factory().BinaryExpression(
				nil,
				context.Factory().Identifier(control.StateName()),
				nil,
				context.Factory().BinaryOperatorToken(
					tsgo.BinaryOperatorEqualsToken,
				),
				context.Factory().NumericLiteral(
					api.IteratorRangeStateReturned.Literal(),
					tsgo.TokenFlagsNone,
				),
			),
		),
		context.Factory().ReturnStatement(context.Factory().FalseLiteral()),
	)
	return api.NewStatementEmission(statements, nil)
}
