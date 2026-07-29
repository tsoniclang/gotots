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
) ([]tsgo.Statement, []api.RootRequest, error) {
	var statements []tsgo.Statement
	var requests []api.RootRequest
	if selected.binding != nil && selected.binding.Name != "_" {
		binding, ok := context.TypesInfo().Implicits[clause].(*types.Var)
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
		targetType, err := children.RepresentedType(
			context.WithRole(api.RoleTypeSwitchBinding),
			clause,
			binding.Type(),
		)
		if err != nil {
			return nil, nil, err
		}
		initial := api.DirectExpression(value)
		if len(typesInCase) == 1 && !typesInCase[0].nilCase {
			initial, err = interfacevalue.Extract(
				context.WithRole(api.RoleTypeSwitchBinding),
				clause,
				binding.Type(),
				value,
			)
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
		statements = append(
			statements,
			context.Factory().VariableStatement(
				nil,
				context.Factory().VariableDeclarationList(
					[]tsgo.VariableDeclaration{
						context.Factory().VariableDeclaration(
							context.Factory().Identifier(name),
							nil,
							targetType.Value(),
							initial.Value(),
						),
					},
					tsgo.NodeFlagsConst,
				),
			),
		)
		requests = append(requests, targetType.Requests()...)
		requests = append(requests, initial.Requests()...)
	} else if context.TypesInfo().Implicits[clause] != nil {
		return nil, nil, api.Unsupported(
			context.WithRole(api.RoleTypeSwitchBinding),
			api.CategoryStatement,
			clause,
		)
	}
	for _, sourceStatement := range clause.Body {
		target, err := children.Statement(
			context.
				WithRole(api.RoleTypeSwitchStatement).
				EnterBreakable(),
			sourceStatement,
		)
		if err != nil {
			return nil, nil, err
		}
		statements = append(statements, target.Statements()...)
		requests = append(requests, target.Requests()...)
	}
	statements = append(
		statements,
		context.Factory().BreakStatement(nil),
	)
	return statements, requests, nil
}
