package mapruntime

import "github.com/tsoniclang/gotots/internal/target/tsgo"

func storeTransport(
	factory tsgo.Factory,
	name string,
) tsgo.FunctionDeclaration {
	keyType := typeName(factory, keyTypeName)
	valueType := typeName(factory, valueTypeName)
	values := factory.Identifier(valuesName)
	key := factory.Identifier(keyName)
	value := factory.Identifier(valueName)
	return factory.FunctionDeclaration(
		[]tsgo.ModifierLike{factory.ExportKeyword()},
		nil,
		factory.Identifier(name),
		[]tsgo.TypeParameterDeclaration{
			typeParameter(factory, keyTypeName),
			typeParameter(factory, valueTypeName),
		},
		[]tsgo.ParameterDeclaration{
			parameter(
				factory,
				valuesName,
				factory.TypeReferenceNode(
					factory.Identifier("Map"),
					[]tsgo.TypeNode{keyType, valueType},
				),
			),
			parameter(factory, keyName, keyType),
			parameter(factory, valueName, valueType),
		},
		factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindVoidKeyword),
		factory.Block([]tsgo.Statement{
			factory.ExpressionStatement(factory.CallExpression(
				factory.PropertyAccessExpression(
					values,
					nil,
					factory.Identifier("set"),
					tsgo.NodeFlagsNone,
				),
				nil,
				nil,
				[]tsgo.Expression{key, value},
				tsgo.NodeFlagsNone,
			)),
		}, true),
	)
}
