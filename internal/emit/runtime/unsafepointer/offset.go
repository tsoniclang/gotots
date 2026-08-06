package unsafepointer

import "github.com/tsoniclang/gotots/internal/target/tsgo"

func (b builder) at() tsgo.MethodDeclaration {
	offset := b.id("offset")
	existing := b.id("existing")
	self := b.factory.ThisExpression()
	return b.method(
		[]tsgo.ModifierLike{b.factory.PrivateKeyword()},
		atName,
		nil,
		[]tsgo.ParameterDeclaration{
			b.parameter("offset", b.numberType(), nil),
		},
		b.unsafeType(),
		b.factory.IfStatement(
			b.binary(
				b.notSafeInteger(offset),
				tsgo.BinaryOperatorBarBarToken,
				b.binary(
					b.binary(
						offset,
						tsgo.BinaryOperatorLessThanToken,
						b.number("0"),
					),
					tsgo.BinaryOperatorBarBarToken,
					b.binary(
						offset,
						tsgo.BinaryOperatorGreaterThanToken,
						b.property(self, "length"),
					),
				),
			),
			b.panic("unsafe pointer offset is outside its allocation"),
			nil,
		),
		b.variable(
			tsgo.NodeFlagsLet,
			"existing",
			b.optional(b.unsafeType()),
			b.call(b.property(self, "children"), "get", offset),
		),
		b.factory.IfStatement(
			b.binary(
				existing,
				tsgo.BinaryOperatorExclamationEqualsEqualsToken,
				b.undefined(),
			),
			b.factory.Block([]tsgo.Statement{
				b.factory.ReturnStatement(existing),
			}, true),
			nil,
		),
		b.factory.ExpressionStatement(b.assign(
			existing,
			b.factory.NewExpression(
				b.id(b.className),
				nil,
				[]tsgo.Expression{
					b.property(self, "base"),
					offset,
					b.property(self, "length"),
					b.property(self, "readBytes"),
					b.property(self, "writeBytes"),
					b.property(self, "children"),
					b.property(self, "flushes"),
					b.property(self, "refreshes"),
				},
			),
		)),
		b.factory.ExpressionStatement(b.call(
			b.property(self, "children"),
			"set",
			offset,
			existing,
		)),
		b.factory.ReturnStatement(existing),
	)
}
