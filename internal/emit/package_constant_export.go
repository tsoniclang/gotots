package emit

import "github.com/tsoniclang/gotots/internal/target/tsgo"

func deferredConstantPackageExport(
	factory tsgo.Factory,
	publicName string,
	deferredName string,
) (tsgo.VariableStatement, tsgo.ExpressionStatement) {
	declaration := factory.VariableStatement(
		[]tsgo.ModifierLike{factory.ExportKeyword()},
		factory.VariableDeclarationList(
			[]tsgo.VariableDeclaration{factory.VariableDeclaration(
				factory.Identifier(publicName),
				nil,
				factory.TypeReferenceNode(
					factory.Identifier("ReturnType"),
					[]tsgo.TypeNode{factory.TypeQueryNode(
						factory.Identifier(deferredName),
						nil,
					)},
				),
				nil,
			)},
			tsgo.NodeFlagsLet,
		),
	)
	initialization := factory.ExpressionStatement(factory.BinaryExpression(
		nil,
		factory.Identifier(publicName),
		nil,
		factory.BinaryOperatorToken(tsgo.BinaryOperatorEqualsToken),
		factory.CallExpression(
			factory.Identifier(deferredName),
			nil,
			nil,
			nil,
			tsgo.NodeFlagsNone,
		),
	))
	return declaration, initialization
}
