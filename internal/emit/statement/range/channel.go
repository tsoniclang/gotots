package rangestatement

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	channelmodel "github.com/tsoniclang/gotots/internal/emit/concurrency/channel"
	runtimechannel "github.com/tsoniclang/gotots/internal/emit/runtime/channel"
	"github.com/tsoniclang/gotots/internal/emit/statement/assignment"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func emitChannel(
	context api.Context,
	children api.ChildEmitter,
	source *ast.RangeStmt,
	model channelmodel.Model,
	targetLabel string,
) (api.StatementEmission, error) {
	if model.Direction() == types.SendOnly ||
		source.Value != nil ||
		(source.Tok == token.ILLEGAL && source.Key != nil) {
		return api.StatementEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
	}
	operand, err := children.Expression(
		context.
			WithRole(api.RoleRangeExpression).
			WithExpectedType(model.Type()),
		source.X,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	operand, err = model.Project(context, operand)
	if err != nil {
		return api.StatementEmission{}, err
	}
	channel, before, requests, err := capture(
		context,
		api.TemporaryRangeOperand,
		operand,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	received, err := channelmodel.BlockingCall(
		context.WithRole(api.RoleRangeExpression),
		source,
		runtimechannel.MemberReceive,
		api.DirectExpression(channel),
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	resultName, err := context.Names().Temporary(api.TemporaryChannelResult)
	if err != nil {
		return api.StatementEmission{}, err
	}
	result := context.Factory().Identifier(resultName)
	element := context.Factory().ElementAccessExpression(
		result,
		nil,
		context.Factory().NumericLiteral("0", tsgo.TokenFlagsNone),
		tsgo.NodeFlagsNone,
	)
	ok := context.Factory().ElementAccessExpression(
		result,
		nil,
		context.Factory().NumericLiteral("1", tsgo.TokenFlagsNone),
		tsgo.NodeFlagsNone,
	)
	var iterationValue assignment.RangeIterationValue
	if source.Key != nil {
		iterationValue, err = assignment.NewFreshRangeIterationValue(
			model.Element(),
			api.DirectExpression(element),
		)
		if err != nil {
			return api.StatementEmission{}, err
		}
	}
	body, err := body(
		context,
		children,
		source,
		iterationValue,
		assignment.RangeIterationValue{},
		targetLabel,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	loopStatements := received.Before()
	loopStatements = append(
		loopStatements,
		variable(
			context,
			tsgo.NodeFlagsConst,
			result,
			received.Value(),
		),
		context.Factory().IfStatement(
			context.Factory().PrefixUnaryExpression(
				tsgo.PrefixUnaryExpressionOperatorKindExclamationToken,
				ok,
			),
			context.Factory().Block(
				[]tsgo.Statement{
					context.Factory().BreakStatement(nil),
				},
				true,
			),
			nil,
		),
	)
	loopStatements = append(loopStatements, body.Value().Statements()...)
	loop := context.Factory().WhileStatement(
		context.Factory().TrueLiteral(),
		context.Factory().Block(loopStatements, true),
	)
	before = append(before, labelTarget(context, targetLabel, loop))
	return api.NewStatementEmission(
		before,
		api.CombineRequests(
			requests,
			received.Requests(),
			body.Requests(),
		),
	)
}
