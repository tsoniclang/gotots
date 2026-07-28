package slice

import "github.com/tsoniclang/gotots/internal/target/tsgo"

func (b builder) clearMethod() tsgo.MethodDeclaration {
	backing := b.thisProperty("backing")
	return b.method(
		nil,
		MemberName(MemberClear),
		nil,
		[]tsgo.ParameterDeclaration{b.parameter("zero", b.typeT())},
		b.factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindVoidKeyword),
		b.factory.IfStatement(
			b.binary(
				backing,
				tsgo.BinaryOperatorEqualsEqualsEqualsToken,
				b.factory.NullLiteral(),
			),
			b.factory.Block([]tsgo.Statement{
				b.factory.ReturnStatement(nil),
			}, true),
			nil,
		),
		b.loop(
			b.thisProperty(MemberName(MemberLength)),
			b.factory.ExpressionStatement(
				b.assign(
					b.backingElement(backing, b.id("index")),
					b.id("zero"),
				),
			),
		),
	)
}

func (b builder) aggregateClearMethod() tsgo.MethodDeclaration {
	backing := b.thisProperty("backing")
	return b.method(
		nil,
		MemberName(MemberClearWith),
		nil,
		[]tsgo.ParameterDeclaration{b.parameter("zero", b.valueFactoryType())},
		b.factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindVoidKeyword),
		b.factory.IfStatement(
			b.binary(
				backing,
				tsgo.BinaryOperatorEqualsEqualsEqualsToken,
				b.factory.NullLiteral(),
			),
			b.factory.Block([]tsgo.Statement{
				b.factory.ReturnStatement(nil),
			}, true),
			nil,
		),
		b.loop(
			b.thisProperty(MemberName(MemberLength)),
			b.factory.ExpressionStatement(
				b.assign(
					b.backingElement(backing, b.id("index")),
					b.invoke(b.id("zero")),
				),
			),
		),
	)
}
