package channel

import "github.com/tsoniclang/gotots/internal/target/tsgo"

func (b builder) sendReadyMethod() tsgo.MethodDeclaration {
	return b.method(
		[]tsgo.ModifierLike{b.factory.PrivateKeyword()},
		"sendReady",
		nil,
		b.booleanType(),
		b.returnStatement(b.logicalOr(
			b.thisProperty("closed"),
			b.binary(
				b.liveLength("buffer", "bufferHead"),
				tsgo.BinaryOperatorLessThanToken,
				b.thisProperty("capacity"),
			),
		)),
	)
}

func (b builder) receiveReadyMethod() tsgo.MethodDeclaration {
	return b.method(
		[]tsgo.ModifierLike{b.factory.PrivateKeyword()},
		"receiveReady",
		nil,
		b.booleanType(),
		b.returnStatement(b.logicalOr(
			b.binary(
				b.liveLength("buffer", "bufferHead"),
				tsgo.BinaryOperatorGreaterThanToken,
				b.number("0"),
			),
			b.thisProperty("closed"),
		)),
	)
}

func (b builder) compactBufferMethod() tsgo.MethodDeclaration {
	return b.compactMethod(
		"compactBuffer",
		"bufferHead",
		"buffer",
	)
}

func (b builder) compactMethod(
	name string,
	headName string,
	valuesName string,
) tsgo.MethodDeclaration {
	head := b.thisProperty(headName)
	values := b.thisProperty(valuesName)
	return b.method(
		[]tsgo.ModifierLike{b.factory.PrivateKeyword()},
		name,
		nil,
		b.voidType(),
		b.factory.IfStatement(
			b.strictEqual(head, b.arrayLength(values)),
			b.factory.Block([]tsgo.Statement{
				b.expression(b.assign(values, b.arrayLiteral())),
				b.expression(b.assign(head, b.number("0"))),
				b.factory.ReturnStatement(nil),
			}, true),
			nil,
		),
		b.factory.IfStatement(
			b.compactionNeeded(head, values),
			b.factory.Block([]tsgo.Statement{
				b.expression(b.assign(
					values,
					b.methodCall(values, "slice", head),
				)),
				b.expression(b.assign(head, b.number("0"))),
			}, true),
			nil,
		),
	)
}

func (b builder) liveLength(
	valuesName string,
	headName string,
) tsgo.Expression {
	return b.subtract(
		b.arrayLength(b.thisProperty(valuesName)),
		b.thisProperty(headName),
	)
}

func (b builder) compactionNeeded(
	head tsgo.Expression,
	values tsgo.Expression,
) tsgo.Expression {
	return b.logicalAnd(
		b.binary(
			head,
			tsgo.BinaryOperatorGreaterThanEqualsToken,
			b.number("64"),
		),
		b.binary(
			b.multiply(head, b.number("2")),
			tsgo.BinaryOperatorGreaterThanEqualsToken,
			b.arrayLength(values),
		),
	)
}
