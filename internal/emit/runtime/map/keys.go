package mapruntime

import "github.com/tsoniclang/gotots/internal/target/tsgo"

func keysMethod(
	factory tsgo.Factory,
	memberName string,
) tsgo.MethodDeclaration {
	keyType := typeName(factory, keyTypeName)
	return factory.MethodDeclaration(
		nil,
		nil,
		factory.Identifier(memberName),
		nil,
		nil,
		nil,
		factory.ArrayTypeNode(keyType),
		factory.Block([]tsgo.Statement{
			factory.ReturnStatement(
				factory.ConditionalExpression(
					definedValues(factory),
					factory.QuestionToken(),
					factory.CallExpression(
						factory.PropertyAccessExpression(
							factory.Identifier("Array"),
							nil,
							factory.Identifier("from"),
							tsgo.NodeFlagsNone,
						),
						nil,
						nil,
						[]tsgo.Expression{
							methodCall(
								factory,
								field(factory, valuesName),
								"keys",
							),
						},
						tsgo.NodeFlagsNone,
					),
					factory.ColonToken(),
					factory.ArrayLiteralExpression(nil, false),
				),
			),
		}, true),
	)
}
