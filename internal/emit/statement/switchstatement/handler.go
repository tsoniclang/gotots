package switchstatement

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	constantvalue "github.com/tsoniclang/gotots/internal/emit/constant"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.SwitchStmt,
) (api.StatementEmission, error) {
	if source.Body == nil {
		return api.StatementEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
	}

	var initializer api.StatementEmission
	var err error
	if source.Init != nil {
		initializer, err = children.ScopedInitializer(
			context.WithRole(api.RoleSwitchInitializer),
			source.Init,
		)
		if err != nil {
			return api.StatementEmission{}, err
		}
	}
	expressionless := source.Tag == nil
	tagType := types.Type(types.Typ[types.Bool])
	tag := api.DirectExpression(context.Factory().TrueLiteral())
	var tagModel definedtype.Model
	wrappedTag := false
	if !expressionless {
		tagType = context.TypesInfo().TypeOf(source.Tag)
		tagModel, wrappedTag = directSwitchModel(tagType)
		if !directSwitchType(tagType, tagModel, wrappedTag) {
			return api.StatementEmission{},
				api.Unsupported(
					context.WithRole(api.RoleSwitchTag),
					api.CategoryExpression,
					source.Tag,
				)
		}
		tag, err = children.Expression(
			context.
				WithRole(api.RoleSwitchTag).
				WithExpectedType(tagType),
			source.Tag,
		)
		if err != nil {
			return api.StatementEmission{}, err
		}
		if wrappedTag {
			tag, err = api.NewExpressionEmission(
				tag.Before(),
				tagModel.Unwrap(context.Factory(), tag.Value()),
				tag.Requests(),
			)
			if err != nil {
				return api.StatementEmission{}, err
			}
		}
	}

	clauses := make([]tsgo.CaseOrDefaultClause, 0, len(source.Body.List))
	requests := api.CombineRequests(initializer.Requests(), tag.Requests())
	defaultSeen := false
	for _, sourceClause := range source.Body.List {
		clause, ok := sourceClause.(*ast.CaseClause)
		if !ok {
			return api.StatementEmission{},
				api.Unsupported(
					context.WithRole(api.RoleSwitchClause),
					api.CategoryStatement,
					sourceClause,
				)
		}
		targetClauses, clauseRequests, isDefault, err := emitClause(
			context,
			children,
			clause,
			tagType,
			expressionless,
			tagModel,
			wrappedTag,
		)
		if err != nil {
			return api.StatementEmission{}, err
		}
		if isDefault {
			if defaultSeen {
				return api.StatementEmission{},
					api.Unsupported(
						context.WithRole(api.RoleSwitchClause),
						api.CategoryStatement,
						clause,
					)
			}
			defaultSeen = true
		}
		clauses = append(clauses, targetClauses...)
		requests = append(requests, clauseRequests...)
	}

	target := context.Factory().SwitchStatement(
		tag.Value(),
		context.Factory().CaseBlock(clauses),
	)
	statements := tag.Before()
	statements = append(statements, target)
	if source.Init != nil {
		scoped := initializer.Statements()
		scoped = append(scoped, statements...)
		return api.DirectStatement(
			context.Factory().Block(scoped, true),
			requests...,
		), nil
	}
	return api.NewStatementEmission(statements, requests)
}

func emitClause(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CaseClause,
	tagType types.Type,
	expressionless bool,
	tagModel definedtype.Model,
	wrappedTag bool,
) ([]tsgo.CaseOrDefaultClause, []api.RootRequest, bool, error) {
	if len(source.List) == 0 {
		body, requests, err := emitClauseBody(context, children, source.Body)
		if err != nil {
			return nil, nil, false, err
		}
		body = append(body, context.Factory().BreakStatement(nil))
		return []tsgo.CaseOrDefaultClause{
			context.Factory().DefaultClause(
				nil,
				[]tsgo.Statement{context.Factory().Block(body, true)},
			),
		}, requests, true, nil
	}

	expressions := make([]tsgo.Expression, 0, len(source.List))
	var requests []api.RootRequest
	for _, caseExpression := range source.List {
		caseType := context.TypesInfo().TypeOf(caseExpression)
		if caseType == nil || !types.AssignableTo(caseType, tagType) {
			return nil, nil, false,
				api.Unsupported(
					context.WithRole(api.RoleSwitchCaseExpression),
					api.CategoryExpression,
					caseExpression,
				)
		}
		caseContext := context.
			WithRole(api.RoleSwitchCaseExpression).
			WithExpectedType(tagType)
		var target api.ExpressionEmission
		var err error
		facts, hasFacts := context.TypesInfo().Types[caseExpression]
		if wrappedTag && hasFacts && facts.Value != nil {
			target, err = constantvalue.EmitValue(
				caseContext,
				caseExpression,
				tagModel.Underlying(),
				facts.Value,
			)
		} else if expressionless {
			target, err = children.Condition(caseContext, caseExpression)
		} else {
			target, err = children.Expression(caseContext, caseExpression)
		}
		if err != nil {
			return nil, nil, false, err
		}
		if wrappedTag && (!hasFacts || facts.Value == nil) {
			target, err = api.NewExpressionEmission(
				target.Before(),
				tagModel.Unwrap(context.Factory(), target.Value()),
				target.Requests(),
			)
			if err != nil {
				return nil, nil, false, err
			}
		}
		if len(target.Before()) != 0 {
			return nil, nil, false,
				api.Unsupported(
					context.WithRole(api.RoleSwitchCaseExpression),
					api.CategoryExpression,
					caseExpression,
				)
		}
		expressions = append(expressions, target.Value())
		requests = append(requests, target.Requests()...)
	}
	body, bodyRequests, err := emitClauseBody(context, children, source.Body)
	if err != nil {
		return nil, nil, false, err
	}
	requests = append(requests, bodyRequests...)
	body = append(body, context.Factory().BreakStatement(nil))
	targetBody := []tsgo.Statement{context.Factory().Block(body, true)}

	targets := make([]tsgo.CaseOrDefaultClause, 0, len(expressions))
	for index, expression := range expressions {
		statements := []tsgo.Statement(nil)
		if index == len(expressions)-1 {
			statements = targetBody
		}
		targets = append(
			targets,
			context.Factory().CaseClause(expression, statements),
		)
	}
	return targets, requests, false, nil
}

func directSwitchModel(
	sourceType types.Type,
) (definedtype.Model, bool) {
	model, ok := definedtype.Resolve(sourceType)
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

func emitClauseBody(
	context api.Context,
	children api.ChildEmitter,
	source []ast.Stmt,
) ([]tsgo.Statement, []api.RootRequest, error) {
	var statements []tsgo.Statement
	var requests []api.RootRequest
	for _, sourceStatement := range source {
		target, err := children.Statement(
			context.
				WithRole(api.RoleSwitchCaseStatement).
				EnterBreakable(),
			sourceStatement,
		)
		if err != nil {
			return nil, nil, err
		}
		statements = append(statements, target.Statements()...)
		requests = append(requests, target.Requests()...)
	}
	return statements, requests, nil
}
