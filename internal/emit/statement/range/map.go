package rangestatement

import (
	"go/ast"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/statement/assignment"
	"github.com/tsoniclang/gotots/internal/emit/value/maprepresentation"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func emitMap(
	context api.Context,
	children api.ChildEmitter,
	source *ast.RangeStmt,
	targetLabel string,
) (api.StatementEmission, error) {
	sourceType := context.TypesInfo().TypeOf(source.X)
	model, ok := maprepresentation.Source(context, sourceType)
	if !ok {
		return api.StatementEmission{},
			api.Unsupported(context, api.CategoryExpression, source.X)
	}
	operand, err := children.Expression(
		context.
			WithRole(api.RoleRangeExpression).
			WithExpectedType(sourceType),
		source.X,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	return emitMapValue(
		context,
		children,
		source,
		model,
		operand,
		targetLabel,
	)
}

func emitMapValue(
	context api.Context,
	children api.ChildEmitter,
	source *ast.RangeStmt,
	model maprepresentation.Model,
	operand api.ExpressionEmission,
	targetLabel string,
) (api.StatementEmission, error) {
	operand, err := model.ReadReceiver(context, source.X, operand)
	if err != nil {
		return api.StatementEmission{}, err
	}
	receiver, before, requests, err := capture(
		context,
		api.TemporaryRangeOperand,
		operand,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	keysValue, err := maprepresentation.RangeKeys(
		context.WithRole(api.RoleRangeExpression),
		source.X,
		model,
		receiver,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	keys, keysBefore, keysRequests, err := capture(
		context,
		api.TemporaryRangeKeys,
		keysValue,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	before = append(before, keysBefore...)
	requests = append(requests, keysRequests...)
	index, err := rangeIndex(context)
	if err != nil {
		return api.StatementEmission{}, err
	}
	keyName, err := context.Names().Temporary(api.TemporaryRangeValue)
	if err != nil {
		return api.StatementEmission{}, err
	}
	keyValue := context.Factory().Identifier(keyName)
	entryName, err := context.Names().Temporary(api.TemporaryRangeValue)
	if err != nil {
		return api.StatementEmission{}, err
	}
	entry := context.Factory().Identifier(entryName)
	lookup, err := maprepresentation.RangeLookupOK(
		context.WithRole(api.RoleRangeValue),
		model,
		receiver,
		keyValue,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	var key assignment.RangeIterationValue
	if source.Key != nil && nonBlank(source.Key) {
		key, err = iteration(
			model.Key(),
			api.DirectExpression(keyValue),
		)
		if err != nil {
			return api.StatementEmission{}, err
		}
	}
	var value assignment.RangeIterationValue
	if source.Value != nil && nonBlank(source.Value) {
		value, err = assignment.NewFreshRangeIterationValue(
			model.Element(),
			api.DirectExpression(tupleElement(context, entry, "0")),
		)
		if err != nil {
			return api.StatementEmission{}, err
		}
	}
	bindings, err := rangeBindings(
		context,
		children,
		source,
		key,
		value,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	sourceBody, err := children.Block(
		rangeBodyContext(context, targetLabel),
		source.Body,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	loopBody := []tsgo.Statement{
		variable(
			context,
			tsgo.NodeFlagsConst,
			keyValue,
			context.Factory().NonNullExpression(
				context.Factory().ElementAccessExpression(
					keys,
					nil,
					index,
					tsgo.NodeFlagsNone,
				),
				tsgo.NodeFlagsNone,
			),
		),
	}
	loopBody = append(loopBody, lookup.Before()...)
	loopBody = append(
		loopBody,
		variable(
			context,
			tsgo.NodeFlagsConst,
			entry,
			lookup.Value(),
		),
		context.Factory().IfStatement(
			context.Factory().PrefixUnaryExpression(
				tsgo.PrefixUnaryExpressionOperatorKindExclamationToken,
				tupleElement(context, entry, "1"),
			),
			context.Factory().Block(
				[]tsgo.Statement{
					context.Factory().ContinueStatement(nil),
				},
				true,
			),
			nil,
		),
	)
	loopBody = append(loopBody, bindings.Statements()...)
	loopBody = append(loopBody, sourceBody.Value().Statements()...)
	loop := context.Factory().ForStatement(
		context.Factory().VariableDeclarationList(
			[]tsgo.VariableDeclaration{
				context.Factory().VariableDeclaration(
					index,
					nil,
					nil,
					context.Factory().NumericLiteral(
						"0",
						tsgo.TokenFlagsNone,
					),
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
			property(context, keys, "length"),
		),
		context.Factory().PostfixUnaryExpression(
			index,
			tsgo.PostfixUnaryExpressionOperatorKindPlusPlusToken,
		),
		context.Factory().Block(loopBody, true),
	)
	before = append(before, labelTarget(context, targetLabel, loop))
	return api.NewStatementEmission(
		before,
		api.CombineRequests(
			requests,
			lookup.Requests(),
			bindings.Requests(),
			sourceBody.Requests(),
		),
	)
}
