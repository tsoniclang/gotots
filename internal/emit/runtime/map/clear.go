package mapruntime

import "github.com/tsoniclang/gotots/internal/target/tsgo"

func clearMethod(
	factory tsgo.Factory,
	memberName string,
) tsgo.MethodDeclaration {
	return factory.MethodDeclaration(
		nil,
		nil,
		factory.Identifier(memberName),
		nil,
		nil,
		nil,
		factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindVoidKeyword),
		factory.Block([]tsgo.Statement{
			factory.IfStatement(
				definedValues(factory),
				factory.Block([]tsgo.Statement{
					factory.ExpressionStatement(
						methodCall(
							factory,
							field(factory, valuesName),
							"clear",
						),
					),
				}, true),
				nil,
			),
		}, true),
	)
}
