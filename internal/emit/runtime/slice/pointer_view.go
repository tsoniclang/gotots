package slice

import (
	"github.com/tsoniclang/gotots/internal/emit/runtime/memoryview"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func (b pointerSliceBuilder) pointerSlice() tsgo.MethodDeclaration {
	low, high, max := b.id("start"), b.id("end"), b.id("limit")
	invalid := b.binary(b.binary(low, tsgo.BinaryOperatorLessThanToken, b.number("0")), tsgo.BinaryOperatorBarBarToken,
		b.binary(b.binary(high, tsgo.BinaryOperatorLessThanToken, low), tsgo.BinaryOperatorBarBarToken,
			b.binary(b.binary(max, tsgo.BinaryOperatorLessThanToken, high), tsgo.BinaryOperatorBarBarToken,
				b.binary(max, tsgo.BinaryOperatorGreaterThanToken, b.thisProperty("extent")))))
	return b.method(nil, MemberName(MemberSlice), nil, []tsgo.ParameterDeclaration{
		b.parameter("low", b.integerInputType()), b.parameter("high", b.optionalIntegerInputType()), b.parameter("max", b.optionalIntegerInputType()),
	}, b.sliceType(),
		b.variable(tsgo.NodeFlagsConst, "start", b.globalCall("BigInt", b.id("low"))),
		b.variable(tsgo.NodeFlagsConst, "end", b.globalCall("BigInt", b.binary(b.id("high"), tsgo.BinaryOperatorQuestionQuestionToken, b.thisProperty("count")))),
		b.variable(tsgo.NodeFlagsConst, "limit", b.globalCall("BigInt", b.binary(b.id("max"), tsgo.BinaryOperatorQuestionQuestionToken, b.thisProperty("extent")))),
		b.factory.IfStatement(invalid, b.throwBounds(), nil),
		b.returnStatement(b.newPointerSlice(b.regionCall("goRegionView", b.thisProperty("region"), low), b.subtract(high, low), b.subtract(max, low))))
}

func (b pointerSliceBuilder) pointerLocation() tsgo.MethodDeclaration {
	return b.method(nil, MemberName(MemberArrayLocation), []tsgo.TypeParameterDeclaration{
		b.factory.TypeParameterDeclaration(nil, b.id("N"), b.integerInputType(), nil, nil),
	}, []tsgo.ParameterDeclaration{b.parameter("length", b.factory.TypeReferenceNode(b.id("N"), nil))},
		memoryview.RegionType(b.factory, b.typeT()),
		b.factory.IfStatement(b.binary(b.id("length"), tsgo.BinaryOperatorGreaterThanToken, b.thisProperty("count")), b.throwBounds(), nil),
		b.returnStatement(b.thisProperty("region")))
}

func (b pointerSliceBuilder) pointerInitialize() tsgo.MethodDeclaration {
	return b.method(nil, StorageInitializeMember, nil, []tsgo.ParameterDeclaration{
		b.parameter("index", b.integerInputType()), b.parameter("value", b.typeT()),
	}, b.factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindVoidKeyword),
		b.factory.IfStatement(b.invalidPointerIndex(b.id("index"), b.thisProperty("extent")), b.throwBounds(), nil),
		b.factory.ExpressionStatement(b.regionCall("goRegionWrite", b.thisProperty("region"), b.id("index"), b.id("value"))))
}

func (b pointerSliceBuilder) pointerWithLength() tsgo.MethodDeclaration {
	return b.method(nil, StorageWithLengthMember, nil, []tsgo.ParameterDeclaration{b.parameter("length", b.integerInputType())},
		b.sliceType(), b.returnStatement(b.call(b.factory.ThisExpression(), MemberName(MemberSlice), b.number("0"), b.id("length"), b.factory.NullLiteral())))
}

func (b pointerSliceBuilder) pointerClear() tsgo.MethodDeclaration {
	return b.method(nil, MemberName(MemberClear), nil, []tsgo.ParameterDeclaration{b.parameter("zero", b.typeT())},
		b.factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindVoidKeyword), b.exactLoop(b.thisProperty("count"),
			b.factory.ExpressionStatement(b.regionCall("goRegionWrite", b.thisProperty("region"), b.id("index"), b.id("zero")))))
}

func (b pointerSliceBuilder) exactLoop(limit tsgo.Expression, statements ...tsgo.Statement) tsgo.Statement {
	index := b.id("index")
	return b.factory.ForStatement(b.factory.VariableDeclarationList([]tsgo.VariableDeclaration{
		b.factory.VariableDeclaration(index, nil, nil, b.factory.BigIntLiteral("0n", tsgo.TokenFlagsNone)),
	}, tsgo.NodeFlagsLet), b.binary(index, tsgo.BinaryOperatorLessThanToken, limit),
		b.factory.PostfixUnaryExpression(index, tsgo.PostfixUnaryExpressionOperatorKindPlusPlusToken), b.factory.Block(statements, true))
}
