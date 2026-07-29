package typeswitchstatement

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	interfacevalue "github.com/tsoniclang/gotots/internal/emit/value/interfacevalue"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func emitClauses(
	context api.Context,
	children api.ChildEmitter,
	source *ast.TypeSwitchStmt,
	selected guard,
	value tsgo.Expression,
) ([]tsgo.CaseOrDefaultClause, []api.RootRequest, error) {
	targets := make(
		[]tsgo.CaseOrDefaultClause,
		0,
		len(source.Body.List),
	)
	var requests []api.RootRequest
	defaultSeen := false
	for _, node := range source.Body.List {
		clause, ok := node.(*ast.CaseClause)
		if !ok {
			return nil, nil, api.Unsupported(
				context.WithRole(api.RoleTypeSwitchClause),
				api.CategoryStatement,
				node,
			)
		}
		isDefault := len(clause.List) == 0
		if isDefault {
			if defaultSeen {
				return nil, nil, api.Unsupported(
					context.WithRole(api.RoleTypeSwitchClause),
					api.CategoryStatement,
					clause,
				)
			}
			defaultSeen = true
		}
		typesInCase, caseRequests, err := emitCaseTypes(
			context,
			clause,
			selected,
			value,
		)
		if err != nil {
			return nil, nil, err
		}
		body, bodyRequests, err := emitCaseBody(
			context,
			children,
			clause,
			selected,
			typesInCase,
			value,
		)
		if err != nil {
			return nil, nil, err
		}
		requests = append(requests, caseRequests...)
		requests = append(requests, bodyRequests...)
		if isDefault {
			targets = append(
				targets,
				context.Factory().DefaultClause(
					nil,
					[]tsgo.Statement{context.Factory().Block(body, true)},
				),
			)
			continue
		}
		for index, sourceType := range typesInCase {
			var statements []tsgo.Statement
			if index == len(typesInCase)-1 {
				statements = []tsgo.Statement{
					context.Factory().Block(body, true),
				}
			}
			targets = append(
				targets,
				context.Factory().CaseClause(
					sourceType.test.Value(),
					statements,
				),
			)
		}
	}
	return targets, requests, nil
}

func emitCaseTypes(
	context api.Context,
	clause *ast.CaseClause,
	selected guard,
	value tsgo.Expression,
) ([]caseType, []api.RootRequest, error) {
	targets := make([]caseType, 0, len(clause.List))
	var requests []api.RootRequest
	for _, sourceType := range clause.List {
		if isNilCase(context, sourceType) {
			targets = append(targets, caseType{
				nilCase: true,
				test: api.DirectExpression(
					context.Factory().BinaryExpression(
						nil,
						value,
						nil,
						context.Factory().BinaryOperatorToken(
							tsgo.BinaryOperatorEqualsEqualsEqualsToken,
						),
						context.Factory().VoidExpression(
							context.Factory().NumericLiteral(
								"0",
								tsgo.TokenFlagsNone,
							),
						),
					),
				),
			})
			continue
		}
		targetType := context.TypesInfo().TypeOf(sourceType)
		if targetType == nil ||
			!types.AssertableTo(selected.source, targetType) {
			return nil, nil, api.Unsupported(
				context.WithRole(api.RoleTypeSwitchCaseType),
				api.CategoryType,
				sourceType,
			)
		}
		test, err := interfacevalue.Test(
			context.WithRole(api.RoleTypeSwitchCaseType),
			targetType,
			value,
		)
		if err != nil {
			return nil, nil, err
		}
		targets = append(targets, caseType{
			sourceType: targetType,
			test:       test,
		})
		requests = append(requests, test.Requests()...)
	}
	return targets, requests, nil
}

func isNilCase(context api.Context, source ast.Expr) bool {
	identifier, ok := source.(*ast.Ident)
	return ok &&
		context.TypesInfo().Uses[identifier] == types.Universe.Lookup("nil")
}
