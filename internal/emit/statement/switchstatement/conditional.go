package switchstatement

import (
	"strconv"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func emitConditional(
	context api.Context,
	tag tagEmission,
	clauses []clauseEmission,
	targetLabel string,
) (api.StatementEmission, error) {
	selectedName, err := context.Names().Temporary(api.TemporarySwitchSelection)
	if err != nil {
		return api.StatementEmission{}, err
	}
	selected := context.Factory().Identifier(selectedName)
	statements := tag.target.Before()
	requests := tag.target.Requests()
	var tagValue tsgo.Expression
	if !tag.expressionless {
		copied, err := context.Values().Transfer(
			context.WithRole(api.RoleSwitchTag),
			tag.source,
			tag.sourceType,
			tag.sourceType,
			api.ValueTransferCopy,
			tag.target,
		)
		if err != nil {
			return api.StatementEmission{}, err
		}
		tagName, err := context.Names().Temporary(api.TemporarySwitchTag)
		if err != nil {
			return api.StatementEmission{}, err
		}
		statements = copied.Before()
		statements = append(
			statements,
			switchVariable(
				context,
				tsgo.NodeFlagsConst,
				tagName,
				copied.Value(),
			),
		)
		requests = copied.Requests()
		tagValue = context.Factory().Identifier(tagName)
	}
	statements = append(
		statements,
		switchVariable(
			context,
			tsgo.NodeFlagsLet,
			selectedName,
			context.Factory().PrefixUnaryExpression(
				tsgo.PrefixUnaryExpressionOperatorKindMinusToken,
				context.Factory().NumericLiteral("1", tsgo.TokenFlagsNone),
			),
		),
	)
	defaultIndex := -1
	for index, clause := range clauses {
		if clause.isDefault {
			defaultIndex = index
			continue
		}
		check, checkRequests, err := conditionalClauseCheck(
			context,
			tag,
			tagValue,
			clause,
			index,
			selected,
		)
		if err != nil {
			return api.StatementEmission{}, err
		}
		statements = append(statements, check)
		requests = append(requests, checkRequests...)
	}
	if defaultIndex >= 0 {
		statements = append(
			statements,
			context.Factory().IfStatement(
				unselected(context, selected),
				context.Factory().Block(
					[]tsgo.Statement{selectClause(
						context,
						selected,
						defaultIndex,
					)},
					true,
				),
				nil,
			),
		)
	}
	execution, executionRequests := conditionalExecution(
		context,
		clauses,
		selected,
	)
	requests = append(requests, executionRequests...)
	statements = append(
		statements,
		labeledTarget(context, targetLabel, execution),
	)
	return api.NewStatementEmission(
		[]tsgo.Statement{context.Factory().Block(statements, true)},
		requests,
	)
}

func conditionalClauseCheck(
	context api.Context,
	tag tagEmission,
	tagValue tsgo.Expression,
	clause clauseEmission,
	clauseIndex int,
	selected tsgo.Identifier,
) (tsgo.Statement, []api.RootRequest, error) {
	matchName, err := context.Names().Temporary(api.TemporarySwitchMatch)
	if err != nil {
		return nil, nil, err
	}
	match := context.Factory().Identifier(matchName)
	body := []tsgo.Statement{
		switchVariable(
			context,
			tsgo.NodeFlagsLet,
			matchName,
			context.Factory().FalseLiteral(),
		),
	}
	var requests []api.RootRequest
	for index, expression := range clause.expressions {
		condition := expression
		if !tag.expressionless {
			equal, err := context.Values().Equal(
				context.WithRole(api.RoleSwitchCaseExpression),
				clause.source.List[index],
				tag.sourceType,
				tagValue,
				expression.Value(),
			)
			if err != nil {
				return nil, nil, err
			}
			condition, err = api.NewExpressionEmission(
				append(expression.Before(), equal.Before()...),
				equal.Value(),
				api.CombineRequests(
					expression.Requests(),
					equal.Requests(),
				),
			)
			if err != nil {
				return nil, nil, err
			}
		}
		evaluate := condition.Before()
		evaluate = append(
			evaluate,
			context.Factory().ExpressionStatement(
				context.Factory().BinaryExpression(
					nil,
					match,
					nil,
					context.Factory().BinaryOperatorToken(
						tsgo.BinaryOperatorEqualsToken,
					),
					condition.Value(),
				),
			),
		)
		body = append(
			body,
			context.Factory().IfStatement(
				context.Factory().PrefixUnaryExpression(
					tsgo.PrefixUnaryExpressionOperatorKindExclamationToken,
					match,
				),
				context.Factory().Block(evaluate, true),
				nil,
			),
		)
		requests = append(requests, condition.Requests()...)
	}
	body = append(
		body,
		context.Factory().IfStatement(
			match,
			context.Factory().Block(
				[]tsgo.Statement{selectClause(
					context,
					selected,
					clauseIndex,
				)},
				true,
			),
			nil,
		),
	)
	return context.Factory().IfStatement(
		unselected(context, selected),
		context.Factory().Block(body, true),
		nil,
	), requests, nil
}

func conditionalExecution(
	context api.Context,
	clauses []clauseEmission,
	selected tsgo.Expression,
) (tsgo.SwitchStatement, []api.RootRequest) {
	targets := make([]tsgo.CaseOrDefaultClause, 0, len(clauses))
	var requests []api.RootRequest
	for index, clause := range clauses {
		body, bodyRequests := directBody(context, clause)
		requests = append(requests, bodyRequests...)
		targets = append(
			targets,
			context.Factory().CaseClause(
				context.Factory().NumericLiteral(
					strconv.Itoa(index),
					tsgo.TokenFlagsNone,
				),
				[]tsgo.Statement{context.Factory().Block(body, true)},
			),
		)
	}
	return context.Factory().SwitchStatement(
		selected,
		context.Factory().CaseBlock(targets),
	), requests
}

func unselected(
	context api.Context,
	selected tsgo.Expression,
) tsgo.Expression {
	return context.Factory().BinaryExpression(
		nil,
		selected,
		nil,
		context.Factory().BinaryOperatorToken(
			tsgo.BinaryOperatorEqualsEqualsEqualsToken,
		),
		context.Factory().PrefixUnaryExpression(
			tsgo.PrefixUnaryExpressionOperatorKindMinusToken,
			context.Factory().NumericLiteral("1", tsgo.TokenFlagsNone),
		),
	)
}

func selectClause(
	context api.Context,
	selected tsgo.Expression,
	index int,
) tsgo.Statement {
	return context.Factory().ExpressionStatement(
		context.Factory().BinaryExpression(
			nil,
			selected,
			nil,
			context.Factory().BinaryOperatorToken(
				tsgo.BinaryOperatorEqualsToken,
			),
			context.Factory().NumericLiteral(
				strconv.Itoa(index),
				tsgo.TokenFlagsNone,
			),
		),
	)
}

func switchVariable(
	context api.Context,
	flags tsgo.NodeFlags,
	name string,
	value tsgo.Expression,
) tsgo.VariableStatement {
	return context.Factory().VariableStatement(
		nil,
		context.Factory().VariableDeclarationList(
			[]tsgo.VariableDeclaration{
				context.Factory().VariableDeclaration(
					context.Factory().Identifier(name),
					nil,
					nil,
					value,
				),
			},
			flags,
		),
	)
}
