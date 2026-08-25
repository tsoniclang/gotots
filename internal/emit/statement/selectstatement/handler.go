package selectstatement

import (
	"go/ast"
	"strconv"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/concurrency/cooperative"
	concurrencyprofile "github.com/tsoniclang/gotots/internal/emit/concurrency/profile"
	panicruntime "github.com/tsoniclang/gotots/internal/emit/runtime/panic"
	"github.com/tsoniclang/gotots/internal/emit/statement/assignment"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.SelectStmt,
) (api.StatementEmission, error) {
	context, targetLabel := context.TakeStatementLabel()
	if source == nil || source.Body == nil {
		return api.StatementEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
	}
	if err := concurrencyprofile.Admit(
		context,
		api.CategoryStatement,
		source,
	); err != nil {
		return api.StatementEmission{}, err
	}
	target, requests, err := prepareClauses(context, children, source)
	if err != nil {
		return api.StatementEmission{}, err
	}
	runtimeSymbol := api.RuntimeSelect
	if target.hasDefault {
		runtimeSymbol = api.RuntimeSelectReady
	}
	selectReference, err := context.Names().Runtime(
		runtimeSymbol,
		api.ImportPhaseValue,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	selected := api.DirectExpression(
		context.Factory().CallExpression(
			context.Factory().Identifier(selectReference.Name()),
			nil,
			nil,
			[]tsgo.Expression{
				context.Factory().ArrayLiteralExpression(
					target.alternatives,
					false,
				),
			},
			tsgo.NodeFlagsNone,
		),
		selectReference.Requests()...,
	)
	if !target.hasDefault &&
		context.ConcurrencySemantics() == api.ConcurrencySemanticsCooperative {
		selected, err = cooperative.Operation(
			context.WithRole(api.RoleSelectClause),
			source,
			selected,
		)
		if err != nil {
			return api.StatementEmission{}, err
		}
	}
	selectedName, err := context.Names().Temporary(
		api.TemporarySwitchSelection,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	selectedValue := context.Factory().Identifier(selectedName)
	statements := target.before
	statements = append(statements, selected.Before()...)
	statements = append(
		statements,
		constant(context, selectedValue, selected.Value()),
	)
	switchValue := tsgo.Expression(selectedValue)
	if target.hasDefault {
		switchValue = context.Factory().ConditionalExpression(
			context.Factory().BinaryExpression(
				nil,
				selectedValue,
				nil,
				context.Factory().BinaryOperatorToken(
					tsgo.BinaryOperatorEqualsEqualsEqualsToken,
				),
				context.Factory().Identifier("undefined"),
			),
			context.Factory().QuestionToken(),
			context.Factory().PrefixUnaryExpression(
				tsgo.PrefixUnaryExpressionOperatorKindMinusToken,
				context.Factory().NumericLiteral(
					"1",
					tsgo.TokenFlagsNone,
				),
			),
			context.Factory().ColonToken(),
			selectedValue,
		)
	}
	clauses, clauseRequests, err := renderClauses(
		context,
		children,
		target.clauses,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	var targetSwitch tsgo.Statement = context.Factory().SwitchStatement(
		switchValue,
		context.Factory().CaseBlock(clauses),
	)
	if targetLabel != "" {
		targetSwitch = context.Factory().LabeledStatement(
			context.Factory().Identifier(targetLabel),
			targetSwitch,
		)
	}
	statements = append(statements, targetSwitch)
	return api.NewStatementEmission(
		statements,
		api.CombineRequests(
			requests,
			selected.Requests(),
			clauseRequests,
		),
	)
}

func renderClauses(
	context api.Context,
	children api.ChildEmitter,
	clauses []clause,
) ([]tsgo.CaseOrDefaultClause, []api.RootRequest, error) {
	targets := make([]tsgo.CaseOrDefaultClause, 0, len(clauses))
	var requests []api.RootRequest
	for _, sourceClause := range clauses {
		var statements []tsgo.Statement
		if sourceClause.assignment != nil {
			guard, guardRequests, err := receiveGuard(
				context,
				sourceClause.receiveResult,
			)
			if err != nil {
				return nil, nil, err
			}
			statements = append(statements, guard)
			requests = append(requests, guardRequests...)
			value := tupleElement(context, sourceClause.receiveResult, 0)
			ok := tupleElement(context, sourceClause.receiveResult, 1)
			bindings, err := assignment.EmitSelectedReceive(
				context.WithRole(api.RoleSelectReceiveTarget),
				children,
				sourceClause.assignment,
				sourceClause.channel.Element(),
				api.DirectExpression(value),
				api.DirectExpression(ok),
			)
			if err != nil {
				return nil, nil, err
			}
			statements = append(statements, bindings.Statements()...)
			requests = append(requests, bindings.Requests()...)
		}
		for _, sourceStatement := range sourceClause.source.Body {
			emission, err := children.Statement(
				context.
					WithRole(api.RoleSelectBody).
					EnterBreakable(),
				sourceStatement,
			)
			if err != nil {
				return nil, nil, err
			}
			statements = append(statements, emission.Statements()...)
			requests = append(requests, emission.Requests()...)
		}
		statements = append(statements, context.Factory().BreakStatement(nil))
		targets = append(
			targets,
			context.Factory().CaseClause(
				context.Factory().NumericLiteral(
					strconv.Itoa(sourceClause.selection),
					tsgo.TokenFlagsNone,
				),
				[]tsgo.Statement{
					context.Factory().Block(statements, true),
				},
			),
		)
	}
	panicReference, err := context.Names().Runtime(
		api.RuntimePanic,
		api.ImportPhaseValue,
	)
	if err != nil {
		return nil, nil, err
	}
	targets = append(
		targets,
		context.Factory().DefaultClause(
			nil,
			[]tsgo.Statement{context.Factory().ExpressionStatement(
				panicruntime.Call(
					context.Factory(),
					panicReference.Name(),
					context.Factory().StringLiteral(
						"select returned an invalid case",
						tsgo.TokenFlagsNone,
					),
				),
			)},
		),
	)
	requests = append(requests, panicReference.Requests()...)
	return targets, requests, nil
}

func receiveGuard(
	context api.Context,
	result tsgo.Identifier,
) (tsgo.Statement, []api.RootRequest, error) {
	panicReference, err := context.Names().Runtime(
		api.RuntimePanic,
		api.ImportPhaseValue,
	)
	if err != nil {
		return nil, nil, err
	}
	condition := context.Factory().BinaryExpression(
		nil,
		result,
		nil,
		context.Factory().BinaryOperatorToken(
			tsgo.BinaryOperatorEqualsEqualsEqualsToken,
		),
		context.Factory().Identifier("undefined"),
	)
	return context.Factory().IfStatement(
		condition,
		context.Factory().Block(
			[]tsgo.Statement{
				context.Factory().ExpressionStatement(
					panicruntime.Call(
						context.Factory(),
						panicReference.Name(),
						context.Factory().StringLiteral(
							"selected receive has no result",
							tsgo.TokenFlagsNone,
						),
					),
				),
			},
			true,
		),
		nil,
	), panicReference.Requests(), nil
}

func tupleElement(
	context api.Context,
	result tsgo.Expression,
	index int,
) tsgo.ElementAccessExpression {
	return context.Factory().ElementAccessExpression(
		result,
		nil,
		context.Factory().NumericLiteral(
			strconv.Itoa(index),
			tsgo.TokenFlagsNone,
		),
		tsgo.NodeFlagsNone,
	)
}
