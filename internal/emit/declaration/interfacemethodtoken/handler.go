package interfacemethodtoken

import (
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Build(
	factory tsgo.Factory,
	name string,
	modifiers []tsgo.ModifierLike,
) tsgo.Statement {
	return factory.VariableStatement(
		modifiers,
		factory.VariableDeclarationList(
			[]tsgo.VariableDeclaration{
				factory.VariableDeclaration(
					factory.Identifier(name),
					nil,
					factory.KeywordTypeNode(
						tsgo.KeywordTypeSyntaxKindObjectKeyword,
					),
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
							factory.ObjectLiteralExpression(nil, false),
						},
						tsgo.NodeFlagsNone,
					),
				),
			},
			tsgo.NodeFlagsConst,
		),
	)
}
