package panicruntime

import "github.com/tsoniclang/gotots/internal/target/tsgo"

func recovery(
	factory tsgo.Factory,
	className string,
	panicName string,
	valueName string,
) tsgo.ClassDeclaration {
	panicType := factory.TypeReferenceNode(factory.Identifier(panicName), nil)
	valueType := factory.TypeReferenceNode(factory.Identifier(valueName), nil)
	pending := factory.Identifier("pending")
	return factory.ClassDeclaration(
		[]tsgo.ModifierLike{factory.ExportKeyword()},
		factory.Identifier(className),
		nil,
		nil,
		[]tsgo.ClassElement{
			factory.ConstructorDeclaration(
				nil,
				nil,
				[]tsgo.ParameterDeclaration{
					parameterProperty(
						factory,
						"pending",
						factory.UnionTypeNode([]tsgo.TypeNode{
							panicType,
							factory.KeywordTypeNode(
								tsgo.KeywordTypeSyntaxKindUndefinedKeyword,
							),
						}),
						factory.PrivateKeyword(),
					),
				},
				nil,
				factory.Block(nil, true),
			),
			factory.MethodDeclaration(
				nil,
				nil,
				factory.Identifier(TakeName),
				nil,
				nil,
				nil,
				factory.UnionTypeNode([]tsgo.TypeNode{
					valueType,
					factory.KeywordTypeNode(
						tsgo.KeywordTypeSyntaxKindUndefinedKeyword,
					),
				}),
				factory.Block(
					[]tsgo.Statement{
						factory.VariableStatement(
							nil,
							factory.VariableDeclarationList(
								[]tsgo.VariableDeclaration{
									factory.VariableDeclaration(
										pending,
										nil,
										nil,
										factory.PropertyAccessExpression(
											factory.ThisExpression(),
											nil,
											pending,
											tsgo.NodeFlagsNone,
										),
									),
								},
								tsgo.NodeFlagsConst,
							),
						),
						factory.IfStatement(
							factory.BinaryExpression(
								nil,
								pending,
								nil,
								factory.BinaryOperatorToken(
									tsgo.BinaryOperatorEqualsEqualsEqualsToken,
								),
								factory.Identifier("undefined"),
							),
							factory.Block(
								[]tsgo.Statement{
									factory.ReturnStatement(
										factory.Identifier("undefined"),
									),
								},
								true,
							),
							nil,
						),
						factory.ExpressionStatement(
							factory.BinaryExpression(
								nil,
								factory.PropertyAccessExpression(
									factory.ThisExpression(),
									nil,
									pending,
									tsgo.NodeFlagsNone,
								),
								nil,
								factory.BinaryOperatorToken(
									tsgo.BinaryOperatorEqualsToken,
								),
								factory.Identifier("undefined"),
							),
						),
						factory.ReturnStatement(
							factory.PropertyAccessExpression(
								pending,
								nil,
								factory.Identifier("value"),
								tsgo.NodeFlagsNone,
							),
						),
					},
					true,
				),
			),
			factory.MethodDeclaration(
				nil,
				nil,
				factory.Identifier(RecoveredName),
				nil,
				nil,
				nil,
				factory.KeywordTypeNode(
					tsgo.KeywordTypeSyntaxKindBooleanKeyword,
				),
				factory.Block(
					[]tsgo.Statement{
						factory.ReturnStatement(
							factory.BinaryExpression(
								nil,
								factory.PropertyAccessExpression(
									factory.ThisExpression(),
									nil,
									pending,
									tsgo.NodeFlagsNone,
								),
								nil,
								factory.BinaryOperatorToken(
									tsgo.BinaryOperatorEqualsEqualsEqualsToken,
								),
								factory.Identifier("undefined"),
							),
						),
					},
					true,
				),
			),
		},
	)
}
