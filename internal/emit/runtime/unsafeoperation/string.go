package unsafeoperation

import (
	pointerruntime "github.com/tsoniclang/gotots/internal/emit/runtime/pointer"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func (b builder) stringFunction(name string) tsgo.FunctionDeclaration {
	integer := b.typeI()
	location := b.id("location")
	length := b.id("numericLength")
	result := b.id("result")
	index := b.id("index")
	backing := b.factory.ElementAccessExpression(location, nil, b.number("0"), tsgo.NodeFlagsNone)
	offset := b.factory.ElementAccessExpression(location, nil, b.number("1"), tsgo.NodeFlagsNone)
	byteValue := b.call(
		b.id(b.denseIndexName),
		"get",
		nil,
		backing,
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
			b.parameter("pointer", b.optionalPointerType(integer, integer)),
			b.parameter("length", b.integerType()),
		},
		b.stringType(),
		b.factory.Block([]tsgo.Statement{
			b.variable(tsgo.NodeFlagsConst, "location", b.optionalRegionType(integer), b.pointerRegion(integer, integer, b.id("pointer"), b.id("length"))),
			b.factory.IfStatement(
				b.binary(location, tsgo.BinaryOperatorEqualsEqualsEqualsToken, b.undefined()),
				b.factory.Block([]tsgo.Statement{b.factory.ReturnStatement(b.factory.StringLiteral("", tsgo.TokenFlagsNone))}, true),
				nil,
			),
			b.variable(tsgo.NodeFlagsConst, "numericLength", nil, b.globalCall("Number", b.id("length"))),
			b.variable(tsgo.NodeFlagsLet, "result", b.stringType(), b.factory.StringLiteral("", tsgo.TokenFlagsNone)),
			b.loop(length, b.factory.ExpressionStatement(
				b.binary(result, tsgo.BinaryOperatorPlusEqualsToken, character),
			)),
			b.factory.ReturnStatement(result),
		}, true),
	)
}

func (b builder) stringDataFunction(name string) tsgo.FunctionDeclaration {
	integer := b.typeI()
	value := b.id("value")
	bytes := b.id("bytes")
	index := b.id("index")
	converterType := b.factory.FunctionTypeNode(
		nil,
		[]tsgo.ParameterDeclaration{b.parameter("value", b.numberType())},
		integer,
	)
	converted := b.factory.CallExpression(
		b.id("convert"), nil, nil,
		[]tsgo.Expression{b.call(value, "charCodeAt", nil, index)},
		tsgo.NodeFlagsNone,
	)
	return b.factory.FunctionDeclaration(
		[]tsgo.ModifierLike{b.factory.ExportKeyword()}, nil, b.id(name),
		[]tsgo.TypeParameterDeclaration{b.typeParameter("I", b.integerType())},
		[]tsgo.ParameterDeclaration{
			b.parameter("value", b.stringType()),
			b.parameter("convert", converterType),
		},
		b.optionalPointerType(integer, integer),
		b.factory.Block([]tsgo.Statement{
			b.factory.IfStatement(
				b.binary(b.property(value, "length"), tsgo.BinaryOperatorEqualsEqualsEqualsToken, b.number("0")),
				b.factory.Block([]tsgo.Statement{b.factory.ReturnStatement(b.undefined())}, true),
				nil,
			),
			b.variable(
				tsgo.NodeFlagsConst,
				"bytes",
				b.factory.ArrayTypeNode(integer),
				b.factory.ArrayLiteralExpression(nil, false),
			),
			b.loop(b.property(value, "length"), b.factory.ExpressionStatement(
				b.call(bytes, "push", nil, converted),
			)),
			b.factory.ReturnStatement(b.call(
				b.id(b.pointerName),
				pointerruntime.ElementName,
				[]tsgo.TypeNode{integer, integer},
				b.factory.ArrayLiteralExpression([]tsgo.Expression{bytes, b.number("0")}, false),
			)),
		}, true),
	)
}
