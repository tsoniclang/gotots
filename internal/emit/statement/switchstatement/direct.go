package switchstatement

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	constantvalue "github.com/tsoniclang/gotots/internal/emit/constant"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func directEligible(
	context api.Context,
	tag tagEmission,
	clauses []clauseEmission,
) bool {
	equalityType := tag.sourceType
	if tag.wrapped {
		equalityType = tag.model.Underlying()
	}
	if !directSwitchType(tag.sourceType, tag.model, tag.wrapped) ||
		context.Values().RequiresCustomEquality(context, equalityType) {
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

func emitDirect(
	context api.Context,
	tag tagEmission,
	clauses []clauseEmission,
	targetLabel string,
) (api.StatementEmission, error) {
	targetClauses := make(
		[]tsgo.CaseOrDefaultClause,
		0,
		len(clauses),
	)
	var requests []api.RootRequest
	for _, clause := range clauses {
		body, bodyRequests := directBody(context, clause)
		requests = append(requests, bodyRequests...)
		if clause.isDefault {
			targetClauses = append(
				targetClauses,
				context.Factory().DefaultClause(
					nil,
					[]tsgo.Statement{context.Factory().Block(body, true)},
				),
			)
			continue
		}
		for index, expression := range clause.expressions {
			value := expression.Value()
			expressionRequests := expression.Requests()
			if tag.wrapped {
				facts, constant := context.TypesInfo().TypeAndValue(
					clause.source.List[index],
				)
				if constant && facts.Value != nil {
					direct, err := constantvalue.EmitValue(
						context.
							WithRole(api.RoleSwitchCaseExpression).
							WithExpectedType(tag.model.Underlying()),
						clause.source.List[index],
						tag.model.Underlying(),
						facts.Value,
					)
					if err != nil {
						return api.StatementEmission{}, err
					}
					if len(direct.Before()) != 0 {
						return api.StatementEmission{},
							api.Unsupported(
								context.WithRole(
									api.RoleSwitchCaseExpression,
								),
								api.CategoryExpression,
								clause.source.List[index],
							)
					}
					value = direct.Value()
					requests = append(requests, direct.Requests()...)
				} else {
					projected, err := tag.model.Project(
						context.WithRole(api.RoleSwitchCaseExpression),
						api.DirectExpression(value, expressionRequests...),
					)
					if err != nil {
						return api.StatementEmission{}, err
					}
					if len(projected.Before()) != 0 {
						return api.StatementEmission{}, api.Unsupported(
							context.WithRole(api.RoleSwitchCaseExpression),
							api.CategoryExpression,
							clause.source.List[index],
						)
					}
					value = projected.Value()
					expressionRequests = projected.Requests()
				}
			}
			var statements []tsgo.Statement
			if index == len(clause.expressions)-1 {
				statements = []tsgo.Statement{
					context.Factory().Block(body, true),
				}
			}
			targetClauses = append(
				targetClauses,
				context.Factory().CaseClause(value, statements),
			)
			requests = append(requests, expressionRequests...)
		}
	}
	tagTarget := tag.target
	if tag.wrapped {
		var err error
		tagTarget, err = tag.model.Project(
			context.WithRole(api.RoleSwitchTag),
			tagTarget,
		)
		if err != nil {
			return api.StatementEmission{}, err
		}
	}
	target := tsgo.Statement(context.Factory().SwitchStatement(
		tagTarget.Value(),
		context.Factory().CaseBlock(targetClauses),
	))
	target = labeledTarget(context, targetLabel, target)
	statements := tagTarget.Before()
	statements = append(statements, target)
	return api.NewStatementEmission(
		statements,
		api.CombineRequests(requests, tagTarget.Requests()),
	)
}

func directBody(
	context api.Context,
	clause clauseEmission,
) ([]tsgo.Statement, []api.RootRequest) {
	var statements []tsgo.Statement
	var requests []api.RootRequest
	for _, emission := range clause.body {
		statements = append(statements, emission.Statements()...)
		requests = append(requests, emission.Requests()...)
	}
	if !clause.fallsThrough {
		statements = append(statements, context.Factory().BreakStatement(nil))
	}
	return statements, requests
}

func directSwitchModel(
	sourceType types.Type,
) (definedtype.Model, bool) {
	model, ok := definedtype.ResolveBasic(sourceType)
	return model, ok
}

func directSwitchType(
	sourceType types.Type,
	model definedtype.Model,
	wrapped bool,
) bool {
	if wrapped {
		sourceType = model.Underlying()
	}
	basic, ok := types.Unalias(sourceType).(*types.Basic)
	if !ok {
		return false
	}
	return basic.Info()&(types.IsBoolean|
		types.IsInteger|
		types.IsFloat|
		types.IsString) != 0
}

func labeledTarget(
	context api.Context,
	name string,
	target tsgo.Statement,
) tsgo.Statement {
	if name == "" {
		return target
	}
	return context.Factory().LabeledStatement(
		context.Factory().Identifier(name),
		target,
	)
}
