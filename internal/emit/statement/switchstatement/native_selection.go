package switchstatement

import (
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func emitNativeSelection(
	context api.Context,
	tag tagEmission,
	clauses []clauseEmission,
	targetLabel string,
) (api.StatementEmission, error) {
	operationContext, nativeNumeric, err := directProjectionContext(
		context,
		tag,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	tagTarget, err := directTagTarget(context, tag, nativeNumeric)
	if err != nil {
		return api.StatementEmission{}, err
	}
	selectedName, err := context.Names().Temporary(api.TemporarySwitchSelection)
	if err != nil {
		return api.StatementEmission{}, err
	}
	selected := context.Factory().Identifier(selectedName)
	targetClauses := make([]tsgo.CaseOrDefaultClause, 0, len(clauses))
	requests := tagTarget.Requests()
	for clauseIndex, clause := range clauses {
		selection := []tsgo.Statement{
			selectClause(context, selected, clauseIndex),
			context.Factory().BreakStatement(nil),
		}
		if clause.isDefault {
			targetClauses = append(
				targetClauses,
				context.Factory().DefaultClause(nil, selection),
			)
			continue
		}
		for expressionIndex, expression := range clause.expressions {
			value, expressionRequests, err := directCaseTarget(
				context,
				operationContext,
				tag,
				clause,
				expressionIndex,
				expression,
				nativeNumeric,
			)
			if err != nil {
				return api.StatementEmission{}, err
			}
			var statements []tsgo.Statement
			if expressionIndex == len(clause.expressions)-1 {
				statements = selection
			}
			targetClauses = append(
				targetClauses,
				context.Factory().CaseClause(value, statements),
			)
			requests = append(requests, expressionRequests...)
		}
	}
	statements := tagTarget.Before()
	statements = append(
		statements,
		switchVariable(
			context,
			tsgo.NodeFlagsLet,
			selectedName,
			context.Factory().PrefixUnaryExpression(
				tsgo.PrefixUnaryExpressionOperatorKindMinusToken,
				context.Factory().NumericLiteral(
					"1",
					tsgo.TokenFlagsNone,
				),
			),
		),
		context.Factory().SwitchStatement(
			tagTarget.Value(),
			context.Factory().CaseBlock(targetClauses),
		),
	)
	execution, executionRequests := conditionalExecution(
		context,
		clauses,
		selected,
		targetLabel,
	)
	statements = append(statements, execution)
	requests = append(requests, executionRequests...)
	return api.NewStatementEmission(
		[]tsgo.Statement{context.Factory().Block(statements, true)},
		requests,
	)
}
