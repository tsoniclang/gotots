package providerinterfacebridge

import (
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/runtime/interfacevalue/contract"
	panicruntime "github.com/tsoniclang/gotots/internal/emit/runtime/panic"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func constructor(
	factory tsgo.Factory,
	providerType api.NameReference,
	contractName string,
) tsgo.ConstructorDeclaration {
	value := factory.Identifier("value")
	return factory.ConstructorDeclaration(
		[]tsgo.ModifierLike{factory.PrivateKeyword()},
		nil,
		[]tsgo.ParameterDeclaration{
			parameter(
				factory,
				value,
				factory.TypeReferenceNode(providerType.EntityName(factory), nil),
			),
		},
		nil,
		factory.Block(
			[]tsgo.Statement{
				factory.ExpressionStatement(
					factory.CallExpression(
						factory.SuperExpression(),
						nil,
						nil,
						[]tsgo.Expression{
							value,
							factory.Identifier(contractName),
						},
						tsgo.NodeFlagsNone,
					),
				),
			},
			true,
		),
	)
}

func fromMethod(
	factory tsgo.Factory,
	name string,
	providerType api.NameReference,
	canonicalType string,
) tsgo.MethodDeclaration {
	value := factory.Identifier("value")
	return factory.MethodDeclaration(
		[]tsgo.ModifierLike{factory.StaticKeyword()},
		nil,
		factory.Identifier(api.ProviderBridgeFromMember),
		nil,
		nil,
		[]tsgo.ParameterDeclaration{
			parameter(
				factory,
				value,
				nullableReferenceType(factory, providerType),
			),
		},
		nullableType(factory, canonicalType),
		factory.Block(
			[]tsgo.Statement{
				factory.ReturnStatement(
					factory.ConditionalExpression(
						factory.BinaryExpression(
							nil,
							value,
							nil,
							factory.BinaryOperatorToken(
								tsgo.BinaryOperatorEqualsEqualsEqualsToken,
							),
							factory.Identifier("undefined"),
						),
						factory.QuestionToken(),
						factory.Identifier("undefined"),
						factory.ColonToken(),
						factory.ConditionalExpression(
							factory.BinaryExpression(
								nil,
								value,
								nil,
								factory.BinaryOperatorToken(
									tsgo.BinaryOperatorInstanceOfKeyword,
								),
								factory.Identifier(name),
							),
							factory.QuestionToken(),
							value,
							factory.ColonToken(),
							factory.NewExpression(
								factory.Identifier(name),
								nil,
								[]tsgo.Expression{value},
							),
						),
					),
				),
			},
			true,
		),
	)
}

func toMethod(
	factory tsgo.Factory,
	name string,
	providerType api.NameReference,
	canonicalType string,
	panicName string,
	directProviderUse bool,
) tsgo.MethodDeclaration {
	value := factory.Identifier("value")
	var finalReturn tsgo.Statement
	if directProviderUse {
		finalReturn = factory.ReturnStatement(value)
	} else {
		finalReturn = factory.ReturnStatement(
			panicruntime.Call(
				factory,
				panicName,
				factory.StringLiteral(
					"provider interface received a foreign implementation",
					tsgo.TokenFlagsNone,
				),
			),
		)
	}
	return factory.MethodDeclaration(
		[]tsgo.ModifierLike{factory.StaticKeyword()},
		nil,
		factory.Identifier(api.ProviderBridgeToMember),
		nil,
		nil,
		[]tsgo.ParameterDeclaration{
			parameter(factory, value, nullableType(factory, canonicalType)),
		},
		nullableReferenceType(factory, providerType),
		factory.Block(
			[]tsgo.Statement{
				factory.IfStatement(
					factory.BinaryExpression(
						nil,
						value,
						nil,
						factory.BinaryOperatorToken(
							tsgo.BinaryOperatorEqualsEqualsEqualsToken,
						),
						factory.Identifier("undefined"),
					),
					factory.Block(
						[]tsgo.Statement{
							factory.ReturnStatement(factory.Identifier("undefined")),
						},
						true,
					),
					nil,
				),
				factory.IfStatement(
					factory.BinaryExpression(
						nil,
						value,
						nil,
						factory.BinaryOperatorToken(
							tsgo.BinaryOperatorInstanceOfKeyword,
						),
						factory.Identifier(name),
					),
					factory.Block(
						[]tsgo.Statement{
							factory.ReturnStatement(
								factory.PropertyAccessExpression(
									value,
									nil,
									factory.Identifier(contract.PayloadMember),
									tsgo.NodeFlagsNone,
								),
							),
						},
						true,
					),
					nil,
				),
				finalReturn,
			},
			true,
		),
	)
}

func payload(factory tsgo.Factory) tsgo.PropertyAccessExpression {
	return factory.PropertyAccessExpression(
		factory.ThisExpression(),
		nil,
		factory.Identifier(contract.PayloadMember),
		tsgo.NodeFlagsNone,
	)
}

func nullableType(factory tsgo.Factory, name string) tsgo.TypeNode {
	return factory.UnionTypeNode([]tsgo.TypeNode{
		factory.TypeReferenceNode(factory.Identifier(name), nil),
		factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindUndefinedKeyword),
	})
}

func nullableReferenceType(
	factory tsgo.Factory,
	reference api.NameReference,
) tsgo.TypeNode {
	return factory.UnionTypeNode([]tsgo.TypeNode{
		factory.TypeReferenceNode(reference.EntityName(factory), nil),
		factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindUndefinedKeyword),
	})
}

func parameter(
	factory tsgo.Factory,
	name tsgo.BindingName,
	typeNode tsgo.TypeNode,
) tsgo.ParameterDeclaration {
	return factory.ParameterDeclaration(nil, nil, name, nil, typeNode, nil)
}
