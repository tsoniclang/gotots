package packagevariable

import (
	"strconv"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/typescriptclass"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func StateDeclarations(
	factory tsgo.Factory,
	fields []tsgo.PropertyDeclaration,
) ([]tsgo.Statement, error) {
	if len(fields) == 0 {
		return nil, &api.InvariantError{
			Role:   api.RoleFileDeclaration,
			Reason: "package state has no fields",
		}
	}
	members := make([]tsgo.ClassElement, 0, len(fields)+1)
	constructorParameters := make([]tsgo.ParameterDeclaration, 0, len(fields))
	initializerParameters := make([]tsgo.ParameterDeclaration, 0, len(fields))
	assignments := make([]tsgo.Statement, 0, len(fields))
	arguments := make([]tsgo.Expression, 0, len(fields))
	seen := make(map[string]bool, len(fields))
	for index, field := range fields {
		if field == nil || field.Type() == nil || len(field.Modifiers()) != 0 ||
			field.PostfixToken() != nil || field.Initializer() != nil {
			return nil, &api.InvariantError{
				Role:   api.RolePackageVariableType,
				Reason: "package state requires a typed constructor-initialized field",
			}
		}
		name, ok := field.Name().(tsgo.Identifier)
		if !ok || seen[name.Text()] {
			return nil, &api.InvariantError{
				Role:   api.RolePackageVariableType,
				Reason: "package state field has no unique canonical identifier",
			}
		}
		seen[name.Text()] = true
		parameterName := "$initial" + strconv.Itoa(index)
		constructorParameters = append(constructorParameters, factory.ParameterDeclaration(
			nil, nil, factory.Identifier(parameterName), nil, field.Type(), nil,
		))
		initializerParameters = append(initializerParameters, factory.ParameterDeclaration(
			nil, nil, factory.Identifier(parameterName), nil, field.Type(), nil,
		))
		assignments = append(assignments, factory.ExpressionStatement(factory.BinaryExpression(
			nil,
			factory.PropertyAccessExpression(factory.ThisExpression(), nil, factory.Identifier(name.Text()), tsgo.NodeFlagsNone),
			nil,
			factory.BinaryOperatorToken(tsgo.BinaryOperatorEqualsToken),
			factory.Identifier(parameterName),
		)))
		arguments = append(arguments, factory.Identifier(parameterName))
		members = append(members, field)
	}
	members = append(members, factory.ConstructorDeclaration(
		nil, nil, constructorParameters, nil, factory.Block(assignments, true),
	))
	class := typescriptclass.Declaration(factory,
		[]tsgo.ModifierLike{factory.ExportKeyword()},
		factory.Identifier(StateClassName), nil, nil, members,
	)
	state := factory.VariableStatement(
		[]tsgo.ModifierLike{factory.ExportKeyword()},
		factory.VariableDeclarationList([]tsgo.VariableDeclaration{factory.VariableDeclaration(
			factory.Identifier(StateValueName), nil,
			factory.TypeReferenceNode(factory.Identifier(StateClassName), nil), nil,
		)}, tsgo.NodeFlagsLet),
	)
	initializer := factory.FunctionDeclaration(
		[]tsgo.ModifierLike{factory.ExportKeyword()}, nil,
		factory.Identifier(StateInitializerName), nil, initializerParameters,
		factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindVoidKeyword),
		factory.Block([]tsgo.Statement{factory.ExpressionStatement(factory.BinaryExpression(
			nil, factory.Identifier(StateValueName), nil,
			factory.BinaryOperatorToken(tsgo.BinaryOperatorEqualsToken),
			factory.NewExpression(factory.Identifier(StateClassName), nil, arguments),
		))}, true),
	)
	return []tsgo.Statement{class, state, initializer}, nil
}
