package interfacemethodtoken

import (
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Build(
	factory tsgo.Factory,
	name string,
	modifiers []tsgo.ModifierLike,
	initializer tsgo.Expression,
) tsgo.Statement {
	if initializer == nil {
		initializer = factory.CallExpression(
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
		)
	}
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
					initializer,
				),
			},
			tsgo.NodeFlagsConst,
		),
	)
}
