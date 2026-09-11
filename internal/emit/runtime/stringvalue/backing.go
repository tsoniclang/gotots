package stringvalue

import (
	"github.com/tsoniclang/gotots/internal/emit/runtime/memoryview"
	"github.com/tsoniclang/gotots/internal/emit/typescriptclass"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func (b builder) textBacking(name string) tsgo.Statement {
	byteType := b.reference("uint8")
	storage := b.id("storage")
	index := b.id("index")
	address := b.factory.CallExpression(b.id("addressOf"), nil, []tsgo.TypeNode{byteType}, []tsgo.Expression{
		b.factory.ElementAccessExpression(storage, nil, b.call(b.id("Number"), index), tsgo.NodeFlagsNone),
	}, tsgo.NodeFlagsNone)
	copyBytes := b.call(b.member(b.id("Array"), "from"), b.own("value"), b.factory.ArrowFunction(nil, nil,
		[]tsgo.ParameterDeclaration{b.parameter("character", b.textType())}, byteType, b.factory.EqualsGreaterThanToken(),
		b.factory.AsExpression(b.call(b.member(b.id("character"), "charCodeAt"), b.number("0")), byteType)))
	return typescriptclass.Declaration(b.factory, []tsgo.ModifierLike{b.factory.ExportKeyword()}, b.id(name), nil, nil, []tsgo.ClassElement{
		b.factory.PropertyDeclaration([]tsgo.ModifierLike{b.factory.PrivateKeyword()}, b.id("storage"), nil,
			b.optional(b.factory.ArrayTypeNode(byteType)), b.undefined()),
		b.factory.ConstructorDeclaration(nil, nil, []tsgo.ParameterDeclaration{b.fieldParameter("value", b.textType())}, nil, b.factory.Block(nil, true)),
		b.method("read", []tsgo.ParameterDeclaration{b.parameter("index", b.integerType())}, byteType,
			b.result(b.factory.AsExpression(b.call(b.member(b.own("value"), "charCodeAt"), b.call(b.id("Number"), index)), byteType))),
		b.method("text", []tsgo.ParameterDeclaration{b.parameter("offset", b.integerType()), b.parameter("length", b.integerType())}, b.textType(),
			b.variable("start", b.call(b.id("Number"), b.id("offset"))),
			b.result(b.call(b.member(b.own("value"), "slice"), b.id("start"),
				b.binary(b.id("start"), tsgo.BinaryOperatorPlusToken, b.call(b.id("Number"), b.id("length")))))),
		b.method("address", []tsgo.ParameterDeclaration{b.parameter("index", b.integerType())}, b.optional(b.reference("Pointer", byteType)),
			b.factory.IfStatement(b.binary(index, tsgo.BinaryOperatorGreaterThanEqualsToken, b.member(b.own("value"), "length")), b.result(b.undefined()), nil),
			b.variable("storage", b.binary(b.own("storage"), tsgo.BinaryOperatorQuestionQuestionToken,
				b.factory.ParenthesizedExpression(b.binary(b.own("storage"), tsgo.BinaryOperatorEqualsToken, copyBytes)))), b.result(address)),
	})
}

func (b builder) pointerBacking(name string) tsgo.Statement {
	byteType := b.reference("uint8")
	index := b.id("index")
	offset := b.id("start")
	location := b.binary(offset, tsgo.BinaryOperatorPlusToken, index)
	read := b.factory.CallExpression(b.id("goRegionRead"), nil, []tsgo.TypeNode{byteType}, []tsgo.Expression{b.own("region"), location}, tsgo.NodeFlagsNone)
	return typescriptclass.Declaration(b.factory, []tsgo.ModifierLike{b.factory.ExportKeyword()}, b.id(name), nil, nil, []tsgo.ClassElement{
		b.factory.ConstructorDeclaration(nil, nil, []tsgo.ParameterDeclaration{b.fieldParameter("region", memoryview.RegionType(b.factory, byteType))}, nil, b.factory.Block(nil, true)),
		b.method("read", []tsgo.ParameterDeclaration{b.parameter("index", b.integerType())}, byteType,
			b.result(b.factory.CallExpression(b.id("goRegionRead"), nil, []tsgo.TypeNode{byteType}, []tsgo.Expression{b.own("region"), index}, tsgo.NodeFlagsNone))),
		b.method("address", []tsgo.ParameterDeclaration{b.parameter("index", b.integerType())}, b.reference("Pointer", byteType),
			b.result(b.factory.CallExpression(b.id("goRegionAddress"), nil, []tsgo.TypeNode{byteType}, []tsgo.Expression{b.own("region"), index}, tsgo.NodeFlagsNone))),
		b.method("text", []tsgo.ParameterDeclaration{b.parameter("offset", b.integerType()), b.parameter("length", b.integerType())}, b.textType(),
			b.variable("start", b.call(b.id("BigInt"), b.id("offset"))),
			b.factory.VariableStatement(nil, b.factory.VariableDeclarationList([]tsgo.VariableDeclaration{
				b.factory.VariableDeclaration(b.id("text"), nil, b.textType(), b.text("")),
			}, tsgo.NodeFlagsLet)),
			b.factory.ForStatement(b.factory.VariableDeclarationList([]tsgo.VariableDeclaration{
				b.factory.VariableDeclaration(index, nil, nil, b.factory.BigIntLiteral("0n", tsgo.TokenFlagsNone)),
			}, tsgo.NodeFlagsLet), b.binary(index, tsgo.BinaryOperatorLessThanToken, b.id("length")),
				b.factory.PostfixUnaryExpression(index, tsgo.PostfixUnaryExpressionOperatorKindPlusPlusToken), b.factory.Block([]tsgo.Statement{
					b.invoke(b.binary(b.id("text"), tsgo.BinaryOperatorPlusEqualsToken, b.call(b.member(b.member(b.id("globalThis"), "String"), "fromCharCode"), read))),
				}, true)), b.result(b.id("text"))),
	})
}
