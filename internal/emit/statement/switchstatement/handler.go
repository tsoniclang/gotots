package switchstatement

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
)

type tagEmission struct {
	source         ast.Expr
	sourceType     types.Type
	target         api.ExpressionEmission
	model          definedtype.Model
	expressionless bool
	wrapped        bool
}

type clauseEmission struct {
	source       *ast.CaseClause
	expressions  []api.ExpressionEmission
	body         []api.StatementEmission
	isDefault    bool
	fallsThrough bool
}

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.SwitchStmt,
) (api.StatementEmission, error) {
	context, targetLabel := context.TakeStatementLabel()
	if source == nil || source.Body == nil {
		return api.StatementEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
	}
	targetLabel, err := context.SelectControlTarget(targetLabel)
	if err != nil {
		return api.StatementEmission{}, err
	}
	fallthroughLowering := requiresFallthroughLowering(source)
	if fallthroughLowering && targetLabel == "" {
		targetLabel, err = context.Names().Temporary(
			api.TemporaryControlTarget,
		)
		if err != nil {
			return api.StatementEmission{}, err
		}
	}
	initializer, err := emitInitializer(context, children, source)
	if err != nil {
		return api.StatementEmission{}, err
	}
	tag, err := emitTag(context, children, source)
	if err != nil {
		return api.StatementEmission{}, err
	}
	clauses, err := emitClauses(
		context,
		children,
		source,
		tag,
		targetLabel,
		fallthroughLowering,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	var target api.StatementEmission
	if directEligible(context, tag, clauses) {
		target, err = emitDirect(context, tag, clauses, targetLabel)
	} else {
		target, err = emitConditional(context, tag, clauses, targetLabel)
	}
	if err != nil {
		return api.StatementEmission{}, err
	}
	requests := api.CombineRequests(
		initializer.Requests(),
		target.Requests(),
	)
	if source.Init == nil {
		return api.NewStatementEmission(target.Statements(), requests)
	}
	statements := initializer.Statements()
	statements = append(statements, target.Statements()...)
	return api.DirectStatement(
		context.Factory().Block(statements, true),
		requests...,
	), nil
}

func emitInitializer(
	context api.Context,
	children api.ChildEmitter,
	source *ast.SwitchStmt,
) (api.StatementEmission, error) {
	if source.Init == nil {
		return api.NewStatementEmission(nil, nil)
	}
	return children.ScopedInitializer(
		context.WithRole(api.RoleSwitchInitializer),
		source.Init,
	)
}

func emitTag(
	context api.Context,
	children api.ChildEmitter,
	source *ast.SwitchStmt,
) (tagEmission, error) {
	if source.Tag == nil {
		return tagEmission{
			sourceType:     types.Typ[types.Bool],
			target:         api.DirectExpression(context.Factory().TrueLiteral()),
			expressionless: true,
		}, nil
	}
	sourceType := context.TypesInfo().TypeOf(source.Tag)
	if sourceType == nil || !types.Comparable(sourceType) {
		return tagEmission{}, api.Unsupported(
			context.WithRole(api.RoleSwitchTag),
			api.CategoryExpression,
			source.Tag,
		)
	}
	target, err := children.Expression(
		context.
			WithRole(api.RoleSwitchTag).
			WithExpectedType(sourceType),
		source.Tag,
	)
	if err != nil {
		return tagEmission{}, err
	}
	model, wrapped := directSwitchModel(sourceType)
	return tagEmission{
		source:     source.Tag,
		sourceType: sourceType,
		target:     target,
		model:      model,
		wrapped:    wrapped,
	}, nil
}

func emitClauses(
	context api.Context,
	children api.ChildEmitter,
	source *ast.SwitchStmt,
	tag tagEmission,
	targetLabel string,
	fallthroughLowering bool,
) ([]clauseEmission, error) {
	clauses := make([]clauseEmission, 0, len(source.Body.List))
	defaultSeen := false
	for index, node := range source.Body.List {
		sourceClause, ok := node.(*ast.CaseClause)
		if !ok {
			return nil, api.Unsupported(
				context.WithRole(api.RoleSwitchClause),
				api.CategoryStatement,
				node,
			)
		}
		clause := clauseEmission{
			source:    sourceClause,
			isDefault: len(sourceClause.List) == 0,
		}
		if clause.isDefault {
			if defaultSeen {
				return nil, api.Unsupported(
					context.WithRole(api.RoleSwitchClause),
					api.CategoryStatement,
					sourceClause,
				)
			}
			defaultSeen = true
		} else {
			expressions, err := emitCaseExpressions(
				context,
				children,
				sourceClause,
				tag,
			)
			if err != nil {
				return nil, err
			}
			clause.expressions = expressions
		}
		body, fallsThrough, err := emitClauseBody(
			context,
			children,
			sourceClause,
			index+1 < len(source.Body.List),
			targetLabel,
			fallthroughLowering,
		)
		if err != nil {
			return nil, err
		}
		clause.body = body
		clause.fallsThrough = fallsThrough
		clauses = append(clauses, clause)
	}
	return clauses, nil
}

func emitCaseExpressions(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CaseClause,
	tag tagEmission,
) ([]api.ExpressionEmission, error) {
	targets := make([]api.ExpressionEmission, 0, len(source.List))
	for _, sourceExpression := range source.List {
		sourceType := context.TypesInfo().TypeOf(sourceExpression)
		if sourceType == nil || !types.AssignableTo(sourceType, tag.sourceType) {
			return nil, api.Unsupported(
				context.WithRole(api.RoleSwitchCaseExpression),
				api.CategoryExpression,
				sourceExpression,
			)
		}
		caseContext := context.
			WithRole(api.RoleSwitchCaseExpression).
			WithExpectedType(tag.sourceType)
		var target api.ExpressionEmission
		var err error
		if tag.expressionless {
			target, err = children.Condition(caseContext, sourceExpression)
		} else {
			target, err = children.Expression(caseContext, sourceExpression)
		}
		if err != nil {
			return nil, err
		}
		targets = append(targets, target)
	}
	return targets, nil
}
