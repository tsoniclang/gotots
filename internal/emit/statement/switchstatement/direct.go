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
	return !containsFallthrough(clauses) &&
		nativeTaggedSwitchEligible(context, tag, clauses)
}

func nativeSelectionEligible(
	context api.Context,
	tag tagEmission,
	clauses []clauseEmission,
) bool {
	return containsFallthrough(clauses) &&
		nativeTaggedSwitchEligible(context, tag, clauses)
}

func nativeTaggedSwitchEligible(
	context api.Context,
	tag tagEmission,
	clauses []clauseEmission,
) bool {
	if tag.expressionless {
		return false
	}
	equalityType := tag.sourceType
	if tag.wrapped {
		equalityType = tag.model.Underlying()
	}
	if !directSwitchType(tag.sourceType, tag.model, tag.wrapped) ||
		context.Values().RequiresCustomEquality(context, equalityType) {
		return false
	}
	for _, clause := range clauses {
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
	operationContext, nativeNumeric, err := directProjectionContext(
		context,
		tag,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
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
			value, expressionRequests, err := directCaseTarget(
				context,
				operationContext,
				tag,
				clause,
				index,
				expression,
				nativeNumeric,
			)
			if err != nil {
				return api.StatementEmission{}, err
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
	tagTarget, err := directTagTarget(context, tag, nativeNumeric)
	if err != nil {
		return api.StatementEmission{}, err
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

func directProjectionContext(
	context api.Context,
	tag tagEmission,
) (api.Context, bool, error) {
	if !tag.wrapped {
		return context, false, nil
	}
	operationContext, err := tag.model.OperationContext(context)
	if err != nil {
		return api.Context{}, false, err
	}
	representation, err := tag.model.Representation(context)
	if err != nil {
		return api.Context{}, false, err
	}
	return operationContext,
		representation.Kind() == api.DefinedValueRepresentationGeneratedNumeric,
		nil
}

func directTagTarget(
	context api.Context,
	tag tagEmission,
	nativeNumeric bool,
) (api.ExpressionEmission, error) {
	if !tag.wrapped || nativeNumeric {
		return tag.target, nil
	}
	return tag.model.Project(
		context.WithRole(api.RoleSwitchTag),
		tag.target,
	)
}

func directCaseTarget(
	context api.Context,
	operationContext api.Context,
	tag tagEmission,
	clause clauseEmission,
	index int,
	expression api.ExpressionEmission,
	nativeNumeric bool,
) (tsgo.Expression, []api.RootRequest, error) {
	value := expression.Value()
	requests := expression.Requests()
	if !tag.wrapped || nativeNumeric {
		return value, requests, nil
	}
	facts, constant := context.TypesInfo().TypeAndValue(clause.source.List[index])
	if constant && facts.Value != nil {
		direct, err := constantvalue.EmitValue(
			operationContext.
				WithRole(api.RoleSwitchCaseExpression).
				WithExpectedType(tag.model.Underlying()),
			clause.source.List[index],
			tag.model.Underlying(),
			facts.Value,
		)
		if err != nil {
			return nil, nil, err
		}
		if len(direct.Before()) != 0 {
			return nil, nil, api.Unsupported(
				context.WithRole(api.RoleSwitchCaseExpression),
				api.CategoryExpression,
				clause.source.List[index],
			)
		}
		return direct.Value(), direct.Requests(), nil
	}
	projected, err := tag.model.Project(
		context.WithRole(api.RoleSwitchCaseExpression),
		api.DirectExpression(value, requests...),
	)
	if err != nil {
		return nil, nil, err
	}
	if len(projected.Before()) != 0 {
		return nil, nil, api.Unsupported(
			context.WithRole(api.RoleSwitchCaseExpression),
			api.CategoryExpression,
			clause.source.List[index],
		)
	}
	return projected.Value(), projected.Requests(), nil
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
