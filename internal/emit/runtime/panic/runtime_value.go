package panicruntime

import (
	interfacecontract "github.com/tsoniclang/gotots/internal/emit/runtime/interfacevalue/contract"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func runtimePanicValue(
	factory tsgo.Factory,
	className string,
	valueName string,
	errorTokenName string,
	runtimeErrorTokenName string,
) tsgo.ClassDeclaration {
	valueType := factory.TypeReferenceNode(factory.Identifier(valueName), nil)
	return factory.ClassDeclaration(
		[]tsgo.ModifierLike{factory.ExportKeyword()},
		factory.Identifier(className),
		nil,
		[]tsgo.HeritageClause{
			factory.HeritageClause(
				tsgo.HeritageClauseTokenKindImplementsKeyword,
				[]tsgo.ExpressionWithTypeArguments{
					factory.ExpressionWithTypeArguments(
						factory.Identifier(valueName),
						nil,
					),
				},
			),
		},
		[]tsgo.ClassElement{
			factory.ConstructorDeclaration(
				nil,
				nil,
				[]tsgo.ParameterDeclaration{
					parameterProperty(
						factory,
						"message",
						factory.KeywordTypeNode(
							tsgo.KeywordTypeSyntaxKindStringKeyword,
						),
						factory.PublicKeyword(),
						factory.ReadonlyKeyword(),
					),
				},
				nil,
				factory.Block(nil, true),
			),
			factory.PropertyDeclaration(
				[]tsgo.ModifierLike{factory.ReadonlyKeyword()},
				factory.Identifier(interfacecontract.DynamicTypeMember),
				nil,
				factory.KeywordTypeNode(
					tsgo.KeywordTypeSyntaxKindObjectKeyword,
				),
				factory.Identifier(className),
			),
			factory.PropertyDeclaration(
				[]tsgo.ModifierLike{factory.ReadonlyKeyword()},
				factory.Identifier(interfacecontract.MethodsMember),
				nil,
				readonlyObjectSet(factory),
				factory.NewExpression(
					factory.Identifier("Set"),
					[]tsgo.TypeNode{
						factory.KeywordTypeNode(
							tsgo.KeywordTypeSyntaxKindObjectKeyword,
						),
					},
					[]tsgo.Expression{
						factory.ArrayLiteralExpression(
							[]tsgo.Expression{
								factory.Identifier(errorTokenName),
								factory.Identifier(runtimeErrorTokenName),
							},
							false,
						),
					},
				),
			),
			implementsMethod(factory),
			equalMethod(factory, valueType),
			hashMethod(factory),
			formatStringProperty(factory),
			formatMethod(factory),
			errorMethod(factory),
			runtimeErrorMethod(factory),
		},
	)
}

func formatStringProperty(factory tsgo.Factory) tsgo.PropertyDeclaration {
	return factory.PropertyDeclaration(
		[]tsgo.ModifierLike{factory.ReadonlyKeyword()},
		factory.Identifier(interfacecontract.FormatStringMember),
		nil,
		factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindBooleanKeyword),
		factory.FalseLiteral(),
	)
}

func formatMethod(factory tsgo.Factory) tsgo.MethodDeclaration {
	verb := factory.Identifier("verb")
	message := factory.PropertyAccessExpression(
		factory.ThisExpression(),
		nil,
		factory.Identifier("message"),
		tsgo.NodeFlagsNone,
	)
	return factory.MethodDeclaration(
		nil,
		nil,
		factory.Identifier(interfacecontract.FormatMember),
		nil,
		nil,
		[]tsgo.ParameterDeclaration{
			parameter(
				factory,
				"verb",
				factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindStringKeyword),
			),
			parameter(
				factory,
				"_flags",
				factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindStringKeyword),
			),
			parameter(
				factory,
				"_precision",
				factory.UnionTypeNode([]tsgo.TypeNode{
					factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindNumberKeyword),
					factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindUndefinedKeyword),
				}),
			),
		},
		factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindStringKeyword),
		factory.Block(
			[]tsgo.Statement{
				factory.IfStatement(
					factory.BinaryExpression(
						nil,
						verb,
						nil,
						factory.BinaryOperatorToken(
							tsgo.BinaryOperatorEqualsEqualsEqualsToken,
						),
						factory.StringLiteral("T", tsgo.TokenFlagsNone),
					),
					factory.Block(
						[]tsgo.Statement{factory.ReturnStatement(
							factory.StringLiteral("runtime.errorString", tsgo.TokenFlagsNone),
						)},
						true,
					),
					nil,
				),
				factory.ReturnStatement(message),
			},
			true,
		),
	)
}

func implementsMethod(factory tsgo.Factory) tsgo.MethodDeclaration {
	contract := factory.Identifier("contract")
	token := factory.Identifier("token")
	has := factory.CallExpression(
		factory.PropertyAccessExpression(
			factory.PropertyAccessExpression(
				factory.ThisExpression(),
				nil,
				factory.Identifier(interfacecontract.MethodsMember),
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
		factory.Identifier(interfacecontract.ImplementsMember),
		nil,
		nil,
		[]tsgo.ParameterDeclaration{
			parameter(factory, "contract", readonlyObjectArray(factory)),
		},
		factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindBooleanKeyword),
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

func errorMethod(factory tsgo.Factory) tsgo.MethodDeclaration {
	return factory.MethodDeclaration(
		nil,
		nil,
		factory.Identifier("Error"),
		nil,
		nil,
		nil,
		factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindStringKeyword),
		factory.Block(
			[]tsgo.Statement{
				factory.ReturnStatement(
					factory.PropertyAccessExpression(
						factory.ThisExpression(),
						nil,
						factory.Identifier("message"),
						tsgo.NodeFlagsNone,
					),
				),
			},
			true,
		),
	)
}

func runtimeErrorMethod(factory tsgo.Factory) tsgo.MethodDeclaration {
	return factory.MethodDeclaration(
		nil,
		nil,
		factory.Identifier("RuntimeError"),
		nil,
		nil,
		nil,
		factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindVoidKeyword),
		factory.Block(nil, true),
	)
}

func equalMethod(
	factory tsgo.Factory,
	valueType tsgo.TypeNode,
) tsgo.MethodDeclaration {
	return factory.MethodDeclaration(
		nil,
		nil,
		factory.Identifier(interfacecontract.EqualMember),
		nil,
		nil,
		[]tsgo.ParameterDeclaration{parameter(factory, "other", valueType)},
		factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindBooleanKeyword),
		factory.Block(
			[]tsgo.Statement{
				factory.ReturnStatement(
					factory.BinaryExpression(
						nil,
						factory.ThisExpression(),
						nil,
						factory.BinaryOperatorToken(
							tsgo.BinaryOperatorEqualsEqualsEqualsToken,
						),
						factory.Identifier("other"),
					),
				),
			},
			true,
		),
	)
}

func hashMethod(factory tsgo.Factory) tsgo.MethodDeclaration {
	return factory.MethodDeclaration(
		nil,
		nil,
		factory.Identifier(interfacecontract.HashMember),
		nil,
		nil,
		nil,
		factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindNumberKeyword),
		factory.Block(
			[]tsgo.Statement{
				factory.ReturnStatement(
					factory.NumericLiteral("0", tsgo.TokenFlagsNone),
				),
			},
			true,
		),
	)
}
