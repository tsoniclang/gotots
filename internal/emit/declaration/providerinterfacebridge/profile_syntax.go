package providerinterfacebridge

import (
	"github.com/tsoniclang/gotots/internal/emit/api"
	interfacecontract "github.com/tsoniclang/gotots/internal/emit/runtime/interfacevalue/contract"
	"github.com/tsoniclang/gotots/internal/emit/typescriptclass"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

const profileGeneratedValueMember = "$go$generated"

func profileContractDeclaration(
	factory tsgo.Factory,
	name string,
	runtimeValue string,
	members []tsgo.TypeElement,
	modifiers []tsgo.ModifierLike,
) tsgo.InterfaceDeclaration {
	return factory.InterfaceDeclaration(
		modifiers,
		factory.Identifier(name),
		nil,
		[]tsgo.HeritageClause{
			factory.HeritageClause(
				tsgo.HeritageClauseTokenKindExtendsKeyword,
				[]tsgo.ExpressionWithTypeArguments{
					factory.ExpressionWithTypeArguments(
						factory.Identifier(runtimeValue),
						nil,
					),
				},
			),
		},
		members,
	)
}

func profileBridgeClass(
	factory tsgo.Factory,
	name string,
	runtimeBase string,
	payloadType string,
	implementedTypes []string,
	members []tsgo.ClassElement,
	modifiers []tsgo.ModifierLike,
) tsgo.ClassDeclaration {
	return typescriptclass.Declaration(factory,
		modifiers,
		factory.Identifier(name),
		nil,
		[]tsgo.HeritageClause{
			factory.HeritageClause(
				tsgo.HeritageClauseTokenKindExtendsKeyword,
				[]tsgo.ExpressionWithTypeArguments{
					factory.ExpressionWithTypeArguments(
						factory.Identifier(runtimeBase),
						[]tsgo.TypeNode{profileNamedType(factory, payloadType)},
					),
				},
			),
			implementsHeritage(factory, implementedTypes),
		},
		members,
	)
}

func profileFromMethod(
	factory tsgo.Factory,
	name string,
	reverseName string,
	providerType string,
	generatedType string,
) tsgo.MethodDeclaration {
	value := factory.Identifier("value")
	return factory.MethodDeclaration(
		[]tsgo.ModifierLike{factory.StaticKeyword()},
		nil,
		factory.Identifier(api.ProviderBridgeFromMember),
		nil,
		nil,
		[]tsgo.ParameterDeclaration{
			parameter(factory, value, profileNullableType(factory, providerType)),
		},
		profileNullableType(factory, generatedType),
		factory.Block(
			[]tsgo.Statement{
				profileUndefinedReturn(factory, value),
				factory.IfStatement(
					profileInstanceOf(factory, value, reverseName),
					factory.Block([]tsgo.Statement{
						factory.ReturnStatement(factory.CallExpression(
							factory.PropertyAccessExpression(
								value,
								nil,
								factory.Identifier(profileGeneratedValueMember),
								tsgo.NodeFlagsNone,
							),
							nil,
							nil,
							nil,
							tsgo.NodeFlagsNone,
						)),
					}, true),
					nil,
				),
				factory.ReturnStatement(factory.ConditionalExpression(
					profileInstanceOf(factory, value, name),
					factory.QuestionToken(),
					value,
					factory.ColonToken(),
					factory.NewExpression(factory.Identifier(name), nil, []tsgo.Expression{value}),
				)),
			},
			true,
		),
	)
}

func profileToMethod(
	factory tsgo.Factory,
	name string,
	reverseName string,
	directName string,
	providerType string,
	generatedType string,
) tsgo.MethodDeclaration {
	value := factory.Identifier("value")
	body := []tsgo.Statement{
		profileUndefinedReturn(factory, value),
		factory.IfStatement(
			profileInstanceOf(factory, value, name),
			factory.Block([]tsgo.Statement{
				factory.ReturnStatement(factory.PropertyAccessExpression(
					value,
					nil,
					factory.Identifier(interfacecontract.PayloadMember),
					tsgo.NodeFlagsNone,
				)),
			}, true),
			nil,
		),
	}
	if directName != "" {
		body = append(body, factory.IfStatement(
			profileInstanceOf(factory, value, directName),
			factory.Block([]tsgo.Statement{
				factory.ReturnStatement(factory.CallExpression(
					factory.PropertyAccessExpression(
						factory.Identifier(directName),
						nil,
						factory.Identifier(api.ProviderBridgeToMember),
						tsgo.NodeFlagsNone,
					),
					nil,
					nil,
					[]tsgo.Expression{value},
					tsgo.NodeFlagsNone,
				)),
			}, true),
			nil,
		))
	}
	body = append(body, factory.ReturnStatement(factory.NewExpression(
		factory.Identifier(reverseName),
		nil,
		[]tsgo.Expression{value},
	)))
	return factory.MethodDeclaration(
		[]tsgo.ModifierLike{factory.StaticKeyword()},
		nil,
		factory.Identifier(api.ProviderBridgeToMember),
		nil,
		nil,
		[]tsgo.ParameterDeclaration{
			parameter(factory, value, profileNullableType(factory, generatedType)),
		},
		profileNullableType(factory, providerType),
		factory.Block(body, true),
	)
}

func profileGeneratedValueMethod(
	factory tsgo.Factory,
	generatedType string,
) tsgo.MethodDeclaration {
	return factory.MethodDeclaration(
		nil,
		nil,
		factory.Identifier(profileGeneratedValueMember),
		nil,
		nil,
		nil,
		profileNamedType(factory, generatedType),
		factory.Block([]tsgo.Statement{
			factory.ReturnStatement(payload(factory)),
		}, true),
	)
}

func profileUndefinedReturn(
	factory tsgo.Factory,
	value tsgo.Expression,
) tsgo.IfStatement {
	return factory.IfStatement(
		factory.BinaryExpression(
			nil,
			value,
			nil,
			factory.BinaryOperatorToken(tsgo.BinaryOperatorEqualsEqualsEqualsToken),
			factory.Identifier("undefined"),
		),
		factory.Block([]tsgo.Statement{
			factory.ReturnStatement(factory.Identifier("undefined")),
		}, true),
		nil,
	)
}

func profileInstanceOf(
	factory tsgo.Factory,
	value tsgo.Expression,
	name string,
) tsgo.BinaryExpression {
	return factory.BinaryExpression(
		nil,
		value,
		nil,
		factory.BinaryOperatorToken(tsgo.BinaryOperatorInstanceOfKeyword),
		factory.Identifier(name),
	)
}

func profileNamedType(factory tsgo.Factory, name string) tsgo.TypeNode {
	return factory.TypeReferenceNode(factory.Identifier(name), nil)
}

func profileNullableType(factory tsgo.Factory, name string) tsgo.TypeNode {
	return factory.UnionTypeNode([]tsgo.TypeNode{
		profileNamedType(factory, name),
		factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindUndefinedKeyword),
	})
}
