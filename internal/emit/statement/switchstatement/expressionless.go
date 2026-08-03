package switchstatement

import (
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func expressionlessDirectEligible(
	tag tagEmission,
	clauses []clauseEmission,
) bool {
	if !tag.expressionless {
		return false
	}
	for _, clause := range clauses {
		if clause.fallsThrough {
			return false
		}
		for _, expression := range clause.expressions {
			if len(expression.Before()) != 0 {
				return false
			}
		}
	}
	return true
}

func emitExpressionlessDirect(
	context api.Context,
	clauses []clauseEmission,
	targetLabel string,
) (api.StatementEmission, error) {
	var alternate tsgo.Statement
	var defaultClause *clauseEmission
	var requests []api.RootRequest
	for index := range clauses {
		if clauses[index].isDefault {
			defaultClause = &clauses[index]
		}
	}
	if defaultClause != nil {
		body, bodyRequests := expressionlessBody(*defaultClause)
		alternate = context.Factory().Block(body, true)
		requests = append(requests, bodyRequests...)
	}
	for index := len(clauses) - 1; index >= 0; index-- {
		clause := clauses[index]
		if clause.isDefault {
			continue
		}
		condition, conditionRequests := expressionlessCondition(
			context,
			clause,
		)
		body, bodyRequests := expressionlessBody(clause)
		alternate = context.Factory().IfStatement(
			condition,
			context.Factory().Block(body, true),
			alternate,
		)
		requests = append(
			requests,
			api.CombineRequests(conditionRequests, bodyRequests)...,
		)
	}
	var statements []tsgo.Statement
	if alternate != nil {
		statements = []tsgo.Statement{alternate}
	}
	target := tsgo.Statement(context.Factory().Block(statements, true))
	target = labeledTarget(context, targetLabel, target)
	return api.DirectStatement(target, requests...), nil
}

func expressionlessCondition(
	context api.Context,
	clause clauseEmission,
) (tsgo.Expression, []api.RootRequest) {
	var result tsgo.Expression
	var requests []api.RootRequest
	for _, expression := range clause.expressions {
		if result == nil {
			result = expression.Value()
		} else {
			result = context.Factory().BinaryExpression(
				nil,
				result,
				nil,
				context.Factory().BinaryOperatorToken(
					tsgo.BinaryOperatorBarBarToken,
				),
				expression.Value(),
			)
		}
		requests = append(requests, expression.Requests()...)
	}
	return result, requests
}

func expressionlessBody(
	clause clauseEmission,
) ([]tsgo.Statement, []api.RootRequest) {
	var statements []tsgo.Statement
	var requests []api.RootRequest
	for _, emission := range clause.body {
		statements = append(statements, emission.Statements()...)
		requests = append(requests, emission.Requests()...)
	}
	return statements, requests
}
