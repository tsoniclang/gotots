package typeswitchstatement

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	interfacevalue "github.com/tsoniclang/gotots/internal/emit/value/interfacevalue"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func emitCaseBody(
	context api.Context,
	children api.ChildEmitter,
	clause *ast.CaseClause,
	selected guard,
	typesInCase []caseType,
	value tsgo.Expression,
	targetLabel string,
) ([]tsgo.Statement, []api.RootRequest, error) {
	var statements []tsgo.Statement
	var requests []api.RootRequest
	if selected.binding != nil && selected.binding.Name != "_" {
		binding, ok := context.TypesInfo().ImplicitOf(clause).(*types.Var)
		if !ok || binding.Name() != selected.binding.Name {
			return nil, nil, api.Unsupported(
				context.WithRole(api.RoleTypeSwitchBinding),
				api.CategoryStatement,
				clause,
			)
		}
		expectedType := selected.sourceType
		if len(typesInCase) == 1 && !typesInCase[0].nilCase {
			expectedType = typesInCase[0].sourceType
		}
		if !types.Identical(binding.Type(), expectedType) {
			return nil, nil, api.Unsupported(
				context.WithRole(api.RoleTypeSwitchBinding),
				api.CategoryStatement,
				clause,
			)
		}
		name, err := context.Names().Declare(binding)
		if err != nil {
			return nil, nil, err
		}
		initial := api.DirectExpression(value)
		if len(typesInCase) == 1 && !typesInCase[0].nilCase {
			if api.ContainsGenericTypeParameter(binding.Type()) {
				assertion, assertionErr := interfacevalue.AssertGeneric(
					context.WithRole(api.RoleTypeSwitchBinding),
					clause.List[0],
					selected.sourceType,
					binding.Type(),
					true,
					api.DirectExpression(value),
				)
				if assertionErr != nil {
					return nil, nil, assertionErr
				}
				initial, err = interfacevalue.GenericAssertionElement(
					context.WithRole(api.RoleTypeSwitchBinding),
					assertion,
					0,
				)
			} else {
				initial, err = interfacevalue.Extract(
					context.WithRole(api.RoleTypeSwitchBinding),
					clause,
					binding.Type(),
					value,
				)
			}
			if err != nil {
				return nil, nil, err
			}
		}
		if len(initial.Before()) != 0 {
			return nil, nil, api.Unsupported(
				context.WithRole(api.RoleTypeSwitchBinding),
				api.CategoryStatement,
				clause,
			)
		}
		targetName := name
		var targetType tsgo.TypeNode
		flags := tsgo.NodeFlagsLet
		represented, typeErr := children.RepresentedType(
			context.WithRole(api.RoleTypeSwitchBinding),
			clause,
			binding.Type(),
		)
		if typeErr != nil {
			return nil, nil, typeErr
		}
		targetType = represented.Value()
		requests = append(requests, represented.Requests()...)
		statements = append(statements, initial.Before()...)
		statements = append(
			statements,
			context.Factory().VariableStatement(
				nil,
				context.Factory().VariableDeclarationList(
					[]tsgo.VariableDeclaration{
						context.Factory().VariableDeclaration(
							context.Factory().Identifier(targetName),
							nil,
							targetType,
							initial.Value(),
						),
					},
					flags,
				),
			),
		)
		requests = append(requests, initial.Requests()...)
	} else if context.TypesInfo().ImplicitOf(clause) != nil {
		return nil, nil, api.Unsupported(
			context.WithRole(api.RoleTypeSwitchBinding),
			api.CategoryStatement,
			clause,
		)
	}
	bodyContext := context.
		WithRole(api.RoleTypeSwitchStatement).
		EnterBreakable()
	if context.CallableControl().Goto() {
		bodyContext = context.
			WithRole(api.RoleTypeSwitchStatement).
			EnterBreakableTarget(targetLabel)
	}
	body, err := children.Statements(bodyContext, clause, clause.Body)
	if err != nil {
		return nil, nil, err
	}
	statements = append(statements, body.Statements()...)
	requests = append(requests, body.Requests()...)
	statements = append(
		statements,
		context.Factory().BreakStatement(nil),
	)
	return statements, requests, nil
}
