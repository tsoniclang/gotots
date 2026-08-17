package mapruntime

import "github.com/tsoniclang/gotots/internal/target/tsgo"

func constructor(
	factory tsgo.Factory,
	keyType tsgo.TypeNode,
	valueType tsgo.TypeNode,
) tsgo.ConstructorDeclaration {
	return factory.ConstructorDeclaration(
		[]tsgo.ModifierLike{factory.PrivateKeyword()},
		nil,
		[]tsgo.ParameterDeclaration{
			parameterProperty(factory, zeroName, valueType),
			parameterProperty(
				factory,
				valuesName,
				factory.UnionTypeNode([]tsgo.TypeNode{
					nativeMapType(factory, keyType, valueType),
					factory.KeywordTypeNode(
						tsgo.KeywordTypeSyntaxKindUndefinedKeyword,
					),
				}),
			),
		},
		nil,
		factory.Block([]tsgo.Statement{
			factory.ExpressionStatement(factory.CallExpression(
				factory.SuperExpression(),
				nil,
				nil,
				nil,
				tsgo.NodeFlagsNone,
			)),
		}, true),
	)
}
