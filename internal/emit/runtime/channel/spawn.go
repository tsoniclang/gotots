package channel

import "github.com/tsoniclang/gotots/internal/target/tsgo"

func (b builder) spawnFunction() tsgo.FunctionDeclaration {
	operation := b.parameter(
		"operation",
		b.factory.FunctionTypeNode(nil, nil, b.voidType()),
	)
	return b.factory.FunctionDeclaration(
		[]tsgo.ModifierLike{b.factory.ExportKeyword()},
		nil,
		b.id(b.spawnName),
		nil,
		[]tsgo.ParameterDeclaration{operation},
		b.voidType(),
		b.factory.Block(
			[]tsgo.Statement{b.factory.ExpressionStatement(
				b.factory.CallExpression(
					b.id("operation"),
					nil,
					nil,
					nil,
					tsgo.NodeFlagsNone,
				),
			)},
			true,
		),
	)
}
