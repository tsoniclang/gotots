package interfaceadapter

import (
	"github.com/tsoniclang/gotots/internal/emit/runtime/interfacevalue"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func methodSetDeclaration(
	factory tsgo.Factory,
	name string,
	tokens []tsgo.Expression,
) tsgo.VariableStatement {
	return factory.VariableStatement(
		nil,
		factory.VariableDeclarationList(
			[]tsgo.VariableDeclaration{
				factory.VariableDeclaration(
					factory.Identifier(name+"$methods"),
					nil,
					readonlySetType(factory),
					factory.NewExpression(
						factory.Identifier("Set"),
						[]tsgo.TypeNode{
							factory.KeywordTypeNode(
								tsgo.KeywordTypeSyntaxKindObjectKeyword,
							),
						},
						[]tsgo.Expression{
							factory.ArrayLiteralExpression(tokens, false),
						},
					),
				),
			},
			tsgo.NodeFlagsConst,
		),
	)
}

func constructor(
	factory tsgo.Factory,
	payload tsgo.TypeNode,
) tsgo.ConstructorDeclaration {
	return factory.ConstructorDeclaration(
		nil,
		nil,
		[]tsgo.ParameterDeclaration{
			factory.ParameterDeclaration(
				[]tsgo.ModifierLike{
					factory.PublicKeyword(),
					factory.ReadonlyKeyword(),
				},
				nil,
				factory.Identifier(ValueMember),
				nil,
				payload,
				nil,
			),
		},
		nil,
		factory.Block(nil, true),
	)
}

func dynamicTypeProperty(
	factory tsgo.Factory,
	dynamicTypeName string,
) tsgo.PropertyDeclaration {
	return factory.PropertyDeclaration(
		[]tsgo.ModifierLike{factory.ReadonlyKeyword()},
		factory.Identifier(interfacevalue.DynamicTypeMember),
		nil,
		factory.KeywordTypeNode(
			tsgo.KeywordTypeSyntaxKindObjectKeyword,
		),
		factory.Identifier(dynamicTypeName),
	)
}

func guardMethod(
	factory tsgo.Factory,
	name string,
	runtimeValueName string,
	dynamicTypeName string,
) tsgo.MethodDeclaration {
	value := factory.Identifier("value")
	return factory.MethodDeclaration(
		[]tsgo.ModifierLike{factory.StaticKeyword()},
		nil,
		factory.Identifier(GuardMember),
		nil,
		nil,
		[]tsgo.ParameterDeclaration{
			parameter(
				factory,
				"value",
				factory.UnionTypeNode([]tsgo.TypeNode{
					factory.TypeReferenceNode(
						factory.Identifier(runtimeValueName),
						nil,
					),
					factory.KeywordTypeNode(
						tsgo.KeywordTypeSyntaxKindUndefinedKeyword,
					),
				}),
			),
		},
		factory.TypePredicateNode(
			nil,
			value,
			factory.TypeReferenceNode(factory.Identifier(name), nil),
		),
		factory.Block(
			[]tsgo.Statement{
				factory.ReturnStatement(
					factory.BinaryExpression(
						nil,
						factory.BinaryExpression(
							nil,
							value,
							nil,
							factory.BinaryOperatorToken(
								tsgo.BinaryOperatorExclamationEqualsEqualsToken,
							),
							factory.Identifier("undefined"),
						),
						nil,
						factory.BinaryOperatorToken(
							tsgo.BinaryOperatorAmpersandAmpersandToken,
						),
						factory.BinaryExpression(
							nil,
							factory.PropertyAccessExpression(
								value,
								nil,
								factory.Identifier(
									interfacevalue.DynamicTypeMember,
								),
								tsgo.NodeFlagsNone,
							),
							nil,
							factory.BinaryOperatorToken(
								tsgo.BinaryOperatorEqualsEqualsEqualsToken,
							),
							factory.Identifier(dynamicTypeName),
						),
					),
				),
			},
			true,
		),
	)
}

func methodSetProperty(
	factory tsgo.Factory,
	name string,
) tsgo.PropertyDeclaration {
	return factory.PropertyDeclaration(
		[]tsgo.ModifierLike{factory.ReadonlyKeyword()},
		factory.Identifier(interfacevalue.MethodsMember),
		nil,
		readonlySetType(factory),
		factory.Identifier(name+"$methods"),
	)
}

func implementsMethod(
	factory tsgo.Factory,
) tsgo.MethodDeclaration {
	contract := factory.Identifier("contract")
	token := factory.Identifier("token")
	has := factory.CallExpression(
		factory.PropertyAccessExpression(
			factory.PropertyAccessExpression(
				factory.ThisExpression(),
				nil,
				factory.Identifier(interfacevalue.MethodsMember),
				tsgo.NodeFlagsNone,
			),
			nil,
			factory.Identifier("has"),
			tsgo.NodeFlagsNone,
		),
		nil,
		nil,
		[]tsgo.Expression{token},
		tsgo.NodeFlagsNone,
	)
	return factory.MethodDeclaration(
		nil,
		nil,
		factory.Identifier(interfacevalue.ImplementsMember),
		nil,
		nil,
		[]tsgo.ParameterDeclaration{
			parameter(
				factory,
				"contract",
				readonlyObjectArray(factory),
			),
		},
		factory.KeywordTypeNode(
			tsgo.KeywordTypeSyntaxKindBooleanKeyword,
		),
		factory.Block(
			[]tsgo.Statement{
				factory.ReturnStatement(
					factory.CallExpression(
						factory.PropertyAccessExpression(
							contract,
							nil,
							factory.Identifier("every"),
							tsgo.NodeFlagsNone,
						),
						nil,
						nil,
						[]tsgo.Expression{
							factory.ArrowFunction(
								nil,
								nil,
								[]tsgo.ParameterDeclaration{
									parameter(
										factory,
										"token",
										factory.KeywordTypeNode(
											tsgo.KeywordTypeSyntaxKindObjectKeyword,
										),
									),
								},
								factory.KeywordTypeNode(
									tsgo.KeywordTypeSyntaxKindBooleanKeyword,
								),
								factory.EqualsGreaterThanToken(),
								has,
							),
						},
						tsgo.NodeFlagsNone,
					),
				),
			},
			true,
		),
	)
}

func readonlySetType(factory tsgo.Factory) tsgo.TypeNode {
	return factory.TypeReferenceNode(
		factory.Identifier("ReadonlySet"),
		[]tsgo.TypeNode{
			factory.KeywordTypeNode(
				tsgo.KeywordTypeSyntaxKindObjectKeyword,
			),
		},
	)
}

func readonlyObjectArray(factory tsgo.Factory) tsgo.TypeNode {
	return factory.TypeOperatorNode(
		tsgo.TypeOperatorNodeOperatorKindReadonlyKeyword,
		factory.ArrayTypeNode(
			factory.KeywordTypeNode(
				tsgo.KeywordTypeSyntaxKindObjectKeyword,
			),
		),
	)
}

func parameter(
	factory tsgo.Factory,
	name string,
	typeNode tsgo.TypeNode,
) tsgo.ParameterDeclaration {
	return factory.ParameterDeclaration(
		nil,
		nil,
		factory.Identifier(name),
		nil,
		typeNode,
		nil,
	)
}
