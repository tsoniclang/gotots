package rangestatement

import (
	"go/ast"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/statement/assignment"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func capture(
	context api.Context,
	kind api.TemporaryKind,
	value api.ExpressionEmission,
) (tsgo.Identifier, []tsgo.Statement, []api.RootRequest, error) {
	name, err := context.Names().Temporary(kind)
	if err != nil {
		return nil, nil, nil, err
	}
	identifier := context.Factory().Identifier(name)
	statements := value.Before()
	statements = append(
		statements,
		variable(context, tsgo.NodeFlagsConst, identifier, value.Value()),
	)
	return identifier, statements, value.Requests(), nil
}

func body(
	context api.Context,
	children api.ChildEmitter,
	source *ast.RangeStmt,
	key assignment.RangeIterationValue,
	value assignment.RangeIterationValue,
) (api.BlockEmission, error) {
	var bindings api.StatementEmission
	var err error
	if source.Key != nil {
		bindings, err = assignment.EmitRangeIteration(
			context.WithRole(api.RoleRangeBody),
			children,
			source,
			key,
			value,
		)
		if err != nil {
			return api.BlockEmission{}, err
		}
	}
	sourceBody, err := children.Block(
		context.WithRole(api.RoleRangeBody).EnterLoop(),
		source.Body,
	)
	if err != nil {
		return api.BlockEmission{}, err
	}
	statements := bindings.Statements()
	statements = append(statements, sourceBody.Value().Statements()...)
	return api.DirectBlock(
		context.Factory().Block(statements, true),
		api.CombineRequests(
			bindings.Requests(),
			sourceBody.Requests(),
		)...,
	), nil
}

func numericLoop(
	context api.Context,
	before []tsgo.Statement,
	requests []api.RootRequest,
	index tsgo.Identifier,
	limit tsgo.Expression,
	body tsgo.Block,
	bodyRequests []api.RootRequest,
	indexBigInt bool,
	targetLabel string,
) (api.StatementEmission, error) {
	zero := tsgo.Expression(
		context.Factory().NumericLiteral("0", tsgo.TokenFlagsNone),
	)
	if indexBigInt {
		zero = context.Factory().BigIntLiteral("0n", tsgo.TokenFlagsNone)
	}
	loop := context.Factory().ForStatement(
		context.Factory().VariableDeclarationList(
			[]tsgo.VariableDeclaration{
				context.Factory().VariableDeclaration(
					index,
					nil,
					nil,
					zero,
				),
			},
			tsgo.NodeFlagsLet,
		),
		context.Factory().BinaryExpression(
			nil,
			index,
			nil,
			context.Factory().BinaryOperatorToken(
				tsgo.BinaryOperatorLessThanToken,
			),
			limit,
		),
		context.Factory().PostfixUnaryExpression(
			index,
			tsgo.PostfixUnaryExpressionOperatorKindPlusPlusToken,
		),
		body,
	)
	before = append(before, labelTarget(context, targetLabel, loop))
	return api.NewStatementEmission(
		before,
		api.CombineRequests(requests, bodyRequests),
	)
}

func variable(
	context api.Context,
	flags tsgo.NodeFlags,
	name tsgo.Identifier,
	value tsgo.Expression,
) tsgo.VariableStatement {
	return context.Factory().VariableStatement(
		nil,
		context.Factory().VariableDeclarationList(
			[]tsgo.VariableDeclaration{
				context.Factory().VariableDeclaration(
					name,
					nil,
					nil,
					value,
				),
			},
			flags,
		),
	)
}

func property(
	context api.Context,
	receiver tsgo.Expression,
	name string,
) tsgo.PropertyAccessExpression {
	return context.Factory().PropertyAccessExpression(
		receiver,
		nil,
		context.Factory().Identifier(name),
		tsgo.NodeFlagsNone,
	)
}

func labelTarget(
	context api.Context,
	name string,
	statement tsgo.Statement,
) tsgo.Statement {
	if name == "" {
		return statement
	}
	return context.Factory().LabeledStatement(
		context.Factory().Identifier(name),
		statement,
	)
}
