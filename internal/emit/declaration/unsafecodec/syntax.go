package unsafecodec

import (
	"go/types"
	"strconv"

	"github.com/tsoniclang/gotots/internal/emit/api"
	integervalue "github.com/tsoniclang/gotots/internal/emit/value/integer"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func (b *builder) id(name string) tsgo.Identifier {
	return b.factory.Identifier(name)
}

func (b *builder) number(value int64) tsgo.NumericLiteral {
	return b.factory.NumericLiteral(strconv.FormatInt(value, 10), tsgo.TokenFlagsNone)
}

func (b *builder) numberText(value string) tsgo.NumericLiteral {
	return b.factory.NumericLiteral(value, tsgo.TokenFlagsNone)
}

func (b *builder) numberType() tsgo.TypeNode {
	return b.factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindNumberKeyword)
}

func (b *builder) voidType() tsgo.TypeNode {
	return b.factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindVoidKeyword)
}

func (b *builder) byteArrayType() tsgo.TypeNode {
	return b.factory.TypeReferenceNode(b.id("Uint8Array"), nil)
}

func (b *builder) parameter(name string, target tsgo.TypeNode) tsgo.ParameterDeclaration {
	return b.factory.ParameterDeclaration(nil, nil, b.id(name), nil, target, nil)
}

func (b *builder) property(
	receiver tsgo.Expression,
	name string,
) tsgo.PropertyAccessExpression {
	return b.factory.PropertyAccessExpression(
		receiver,
		nil,
		b.id(name),
		tsgo.NodeFlagsNone,
	)
}

func (b *builder) element(
	receiver tsgo.Expression,
	index tsgo.Expression,
) tsgo.ElementAccessExpression {
	return b.factory.ElementAccessExpression(
		receiver,
		nil,
		index,
		tsgo.NodeFlagsNone,
	)
}

func (b *builder) call(
	callee tsgo.Expression,
	typeArguments []tsgo.TypeNode,
	arguments ...tsgo.Expression,
) tsgo.CallExpression {
	return b.factory.CallExpression(
		callee,
		nil,
		typeArguments,
		arguments,
		tsgo.NodeFlagsNone,
	)
}

func (b *builder) memberCall(
	receiver tsgo.Expression,
	name string,
	arguments ...tsgo.Expression,
) tsgo.CallExpression {
	return b.call(b.property(receiver, name), nil, arguments...)
}

func (b *builder) binary(
	left tsgo.Expression,
	operator tsgo.BinaryOperator,
	right tsgo.Expression,
) tsgo.BinaryExpression {
	return b.factory.BinaryExpression(
		nil,
		left,
		nil,
		b.factory.BinaryOperatorToken(operator),
		right,
	)
}

func (b *builder) variable(
	name string,
	target tsgo.TypeNode,
	value tsgo.Expression,
) tsgo.VariableStatement {
	return b.factory.VariableStatement(
		nil,
		b.factory.VariableDeclarationList(
			[]tsgo.VariableDeclaration{b.factory.VariableDeclaration(
				b.id(name),
				nil,
				target,
				value,
			)},
			tsgo.NodeFlagsConst,
		),
	)
}

func (b *builder) readArrow(
	storage tsgo.TypeNode,
	statements ...tsgo.Statement,
) tsgo.ArrowFunction {
	return b.factory.ArrowFunction(
		nil,
		nil,
		[]tsgo.ParameterDeclaration{
			b.parameter("bytes", b.byteArrayType()),
			b.parameter("offset", b.numberType()),
		},
		storage,
		b.factory.EqualsGreaterThanToken(),
		b.factory.Block(statements, true),
	)
}

func (b *builder) writeArrow(
	storage tsgo.TypeNode,
	statements ...tsgo.Statement,
) tsgo.ArrowFunction {
	return b.factory.ArrowFunction(
		nil,
		nil,
		[]tsgo.ParameterDeclaration{
			b.parameter("value", storage),
			b.parameter("bytes", b.byteArrayType()),
			b.parameter("offset", b.numberType()),
		},
		b.voidType(),
		b.factory.EqualsGreaterThanToken(),
		b.factory.Block(statements, true),
	)
}

func (b *builder) dataView() tsgo.NewExpression {
	bytes := b.id("bytes")
	return b.factory.NewExpression(
		b.id("DataView"),
		nil,
		[]tsgo.Expression{
			b.property(bytes, "buffer"),
			b.property(bytes, "byteOffset"),
			b.property(bytes, "byteLength"),
		},
	)
}

func (b *builder) integerZero() (tsgo.Expression, error) {
	return integervalue.Literal(b.context, types.Typ[types.Uintptr], "0")
}

func (b *builder) convertInteger(value tsgo.Expression) tsgo.Expression {
	intrinsic := api.TargetIntrinsicNumber
	if integervalue.TypeUsesBigInt(b.context, types.Typ[types.Uintptr]) {
		intrinsic = api.TargetIntrinsicBigInt
	}
	return b.call(intrinsic.Expression(b.factory), nil, value)
}

func (b *builder) littleEndian() tsgo.Expression {
	if b.context.MemoryByteOrder() == api.MemoryByteOrderLittleEndian {
		return b.factory.TrueLiteral()
	}
	return b.factory.FalseLiteral()
}
