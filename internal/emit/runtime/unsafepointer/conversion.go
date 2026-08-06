package unsafepointer

import (
	pointerruntime "github.com/tsoniclang/gotots/internal/emit/runtime/pointer"
	unsafecodecruntime "github.com/tsoniclang/gotots/internal/emit/runtime/unsafecodec"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func (b builder) to() tsgo.MethodDeclaration {
	typeL := b.typeReference("L")
	typeS := b.typeReference("S")
	value := b.id("value")
	codec := b.id("codec")
	bound := b.id("bound")
	current := b.id("current")
	read := b.factory.ArrowFunction(nil, nil, nil, typeS, b.factory.EqualsGreaterThanToken(), current)
	next := b.id("next")
	writeBytes := b.call(
		value,
		"writeBytes",
		b.property(value, "offset"),
		b.call(codec, unsafecodecruntime.EncodeName, current),
	)
	write := b.factory.ArrowFunction(
		nil,
		nil,
		[]tsgo.ParameterDeclaration{b.parameter("next", typeS, nil)},
		b.voidType(),
		b.factory.EqualsGreaterThanToken(),
		b.factory.Block([]tsgo.Statement{
			b.factory.ExpressionStatement(b.assign(current, next)),
			b.factory.ExpressionStatement(writeBytes),
			b.factory.ExpressionStatement(b.call(value, "refresh")),
		}, true),
	)
	flush := b.factory.ArrowFunction(
		nil,
		nil,
		nil,
		b.voidType(),
		b.factory.EqualsGreaterThanToken(),
		writeBytes,
	)
	refresh := b.factory.ArrowFunction(
		nil,
		nil,
		nil,
		b.voidType(),
		b.factory.EqualsGreaterThanToken(),
		b.factory.Block([]tsgo.Statement{b.factory.ExpressionStatement(b.assign(
			current,
			b.call(
				codec,
				unsafecodecruntime.ReadName,
				b.call(
					value,
					"readBytes",
					b.property(value, "offset"),
					b.property(codec, "size"),
				),
				b.number("0"),
			),
		))}, true),
	)
	return b.method(
		[]tsgo.ModifierLike{b.factory.PublicKeyword(), b.factory.StaticKeyword()},
		ToName,
		[]tsgo.TypeParameterDeclaration{
			b.typeParameter("L", nil),
			b.typeParameter("S", nil),
		},
		[]tsgo.ParameterDeclaration{
			b.parameter("value", b.optional(b.unsafeType()), nil),
			b.parameter("codec", b.codecType(typeS), nil),
		},
		b.optional(b.typeReference(b.pointerName, typeL, typeS)),
		b.factory.IfStatement(
			b.binary(value, tsgo.BinaryOperatorEqualsEqualsEqualsToken, b.undefined()),
			b.factory.Block([]tsgo.Statement{b.factory.ReturnStatement(b.undefined())}, true),
			nil,
		),
		b.variable(
			tsgo.NodeFlagsConst,
			"bound",
			nil,
			b.call(codec, unsafecodecruntime.BoundName, value),
		),
		b.factory.IfStatement(
			b.binary(
				bound,
				tsgo.BinaryOperatorExclamationEqualsEqualsToken,
				b.undefined(),
			),
			b.factory.Block([]tsgo.Statement{b.factory.ReturnStatement(
				b.typedCall(
					b.id(b.pointerName),
					pointerruntime.UnsafeViewName,
					[]tsgo.TypeNode{typeL, typeS},
					b.element(bound, b.number("0")),
					b.element(bound, b.number("1")),
					b.element(bound, b.number("2")),
					b.element(bound, b.number("3")),
				),
			)}, true),
			nil,
		),
		b.variable(
			tsgo.NodeFlagsLet,
			"current",
			typeS,
			b.call(
				codec,
				unsafecodecruntime.ReadName,
				b.call(
					value,
					"readBytes",
					b.property(value, "offset"),
					b.property(codec, "size"),
				),
				b.number("0"),
			),
		),
		b.variable(tsgo.NodeFlagsConst, "read", nil, read),
		b.variable(tsgo.NodeFlagsConst, "write", nil, write),
		b.variable(tsgo.NodeFlagsConst, "flush", nil, flush),
		b.variable(tsgo.NodeFlagsConst, "refresh", nil, refresh),
		b.factory.ExpressionStatement(b.call(
			b.property(value, "flushes"),
			"push",
			b.id("flush"),
		)),
		b.factory.ExpressionStatement(b.call(
			b.property(value, "refreshes"),
			"push",
			b.id("refresh"),
		)),
		b.factory.ExpressionStatement(b.call(
			codec,
			unsafecodecruntime.BindName,
			value,
			b.factory.ArrayLiteralExpression([]tsgo.Expression{
				value,
				b.id("read"),
				b.id("write"),
				b.undefined(),
			}, false),
		)),
		b.factory.ReturnStatement(b.typedCall(
			b.id(b.pointerName),
			pointerruntime.UnsafeViewName,
			[]tsgo.TypeNode{typeL, typeS},
			value,
			b.id("read"),
			b.id("write"),
			b.undefined(),
		)),
	)
}
