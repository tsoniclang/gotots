package interfacetype

import (
	"github.com/tsoniclang/gotots/internal/emit/runtime/interfacevalue"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func contractDeclaration(
	factory tsgo.Factory,
	name string,
	modifiers []tsgo.ModifierLike,
	tokens []tsgo.Expression,
) tsgo.VariableStatement {
	return factory.VariableStatement(
		modifiers,
		factory.VariableDeclarationList(
			[]tsgo.VariableDeclaration{
				factory.VariableDeclaration(
					factory.Identifier(name+"$contract"),
					nil,
					readonlyObjectArray(factory),
					factory.CallExpression(
						factory.PropertyAccessExpression(
							factory.Identifier("Object"),
							nil,
							factory.Identifier("freeze"),
							tsgo.NodeFlagsNone,
						),
						nil,
						nil,
						[]tsgo.Expression{
							factory.ArrayLiteralExpression(tokens, false),
						},
						tsgo.NodeFlagsNone,
					),
				),
			},
			tsgo.NodeFlagsConst,
		),
	)
}

func guardDeclaration(
	factory tsgo.Factory,
	name string,
	runtimeValueName string,
	modifiers []tsgo.ModifierLike,
) tsgo.FunctionDeclaration {
	value := factory.Identifier("value")
	return factory.FunctionDeclaration(
		modifiers,
		nil,
		factory.Identifier(name+"$is"),
		nil,
		[]tsgo.ParameterDeclaration{
			factory.ParameterDeclaration(
				nil,
				nil,
				value,
				nil,
				factory.UnionTypeNode([]tsgo.TypeNode{
					factory.TypeReferenceNode(
						factory.Identifier(runtimeValueName),
						nil,
					),
					factory.KeywordTypeNode(
						tsgo.KeywordTypeSyntaxKindUndefinedKeyword,
					),
				}),
				nil,
			),
		},
		factory.TypePredicateNode(
			nil,
			value,
			factory.TypeReferenceNode(
				factory.Identifier(name),
				nil,
			),
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
						factory.CallExpression(
							factory.PropertyAccessExpression(
								value,
								nil,
								factory.Identifier(
									interfacevalue.ImplementsMember,
								),
								tsgo.NodeFlagsNone,
							),
							nil,
							nil,
							[]tsgo.Expression{
								factory.Identifier(name + "$contract"),
							},
							tsgo.NodeFlagsNone,
						),
					),
				),
			},
			true,
		),
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
