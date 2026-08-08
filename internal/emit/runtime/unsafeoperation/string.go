package unsafeoperation

import (
	panicruntime "github.com/tsoniclang/gotots/internal/emit/runtime/panic"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func (b builder) stringFunction(name string) tsgo.FunctionDeclaration {
	integer := b.typeI()
	source := b.id("source")
	offset := b.id("numericOffset")
	length := b.id("numericLength")
	result := b.id("result")
	index := b.id("index")
	byteValue := b.call(
		source,
		"get",
		nil,
		b.binary(offset, tsgo.BinaryOperatorPlusToken, index),
	)
	character := b.call(
		b.property(b.id("globalThis"), "String"),
		"fromCharCode",
		nil,
		b.globalCall("Number", byteValue),
	)
	return b.factory.FunctionDeclaration(
		[]tsgo.ModifierLike{b.factory.ExportKeyword()}, nil, b.id(name),
		[]tsgo.TypeParameterDeclaration{b.typeParameter("I", b.integerType())},
		[]tsgo.ParameterDeclaration{
			b.parameter("source", b.sliceType(integer)),
			b.parameter("offset", b.integerType()),
			b.parameter("length", b.integerType()),
		},
		b.stringType(),
		b.factory.Block([]tsgo.Statement{
			b.variable(tsgo.NodeFlagsConst, "numericOffset", nil, b.globalCall("Number", b.id("offset"))),
			b.variable(tsgo.NodeFlagsConst, "numericLength", nil, b.globalCall("Number", b.id("length"))),
			b.factory.IfStatement(
				b.binary(length, tsgo.BinaryOperatorLessThanToken, b.number("0")),
				b.factory.ExpressionStatement(panicruntime.Call(
					b.factory,
					b.panicName,
					b.factory.StringLiteral("unsafe string length is negative", tsgo.TokenFlagsNone),
				)),
				nil,
			),
			b.variable(tsgo.NodeFlagsLet, "result", b.stringType(), b.factory.StringLiteral("", tsgo.TokenFlagsNone)),
			b.loop(length, b.factory.ExpressionStatement(
				b.binary(result, tsgo.BinaryOperatorPlusEqualsToken, character),
			)),
			b.factory.ReturnStatement(result),
		}, true),
	)
}
