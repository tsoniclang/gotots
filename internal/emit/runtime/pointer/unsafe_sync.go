package pointer

import "github.com/tsoniclang/gotots/internal/target/tsgo"

func (b builder) unsafeSyncType() tsgo.TypeNode {
	callback := b.factory.FunctionTypeNode(nil, nil, b.unsafeVoidType())
	return b.factory.TypeOperatorNode(
		tsgo.TypeOperatorNodeOperatorKindReadonlyKeyword,
		b.factory.TupleTypeNode([]tsgo.TypeNode{callback, callback}),
	)
}

func (b builder) unsafeSyncProperty() tsgo.PropertyDeclaration {
	return b.factory.PropertyDeclaration(
		[]tsgo.ModifierLike{b.factory.PrivateKeyword()},
		b.id(unsafeSyncName),
		b.factory.QuestionToken(),
		b.unsafeSyncType(),
		nil,
	)
}

func (b builder) unsafeBindMethod() tsgo.MethodDeclaration {
	typeL := b.typeReference("L")
	typeS := b.typeReference("S")
	pointer := b.id("pointer")
	return b.method(
		[]tsgo.ModifierLike{b.factory.StaticKeyword()},
		UnsafeBindName,
		[]tsgo.TypeParameterDeclaration{
			b.typeParameter("L", nil),
			b.typeParameter("S", nil),
		},
		[]tsgo.ParameterDeclaration{
			b.parameter("pointer", b.pointerType(typeL, typeS)),
			b.parameter("before", b.factory.FunctionTypeNode(nil, nil, b.unsafeVoidType())),
			b.parameter("after", b.factory.FunctionTypeNode(nil, nil, b.unsafeVoidType())),
		},
		b.unsafeVoidType(),
		b.factory.ExpressionStatement(b.assign(
			b.property(pointer, unsafeSyncName),
			b.factory.ArrayLiteralExpression(
				[]tsgo.Expression{b.id("before"), b.id("after")},
				false,
			),
		)),
	)
}

func (b builder) unsafeSyncCall(index string) tsgo.Statement {
	sync := b.id("sync")
	return b.factory.Block([]tsgo.Statement{
		b.variable(tsgo.NodeFlagsConst, "sync", nil, b.property(b.factory.ThisExpression(), unsafeSyncName)),
		b.factory.IfStatement(
			b.binary(
				sync,
				tsgo.BinaryOperatorExclamationEqualsEqualsToken,
				b.undefined(),
			),
			b.factory.Block([]tsgo.Statement{b.factory.ExpressionStatement(
				b.factory.CallExpression(
					b.factory.ElementAccessExpression(
						sync,
						nil,
						b.factory.NumericLiteral(index, tsgo.TokenFlagsNone),
						tsgo.NodeFlagsNone,
					),
					nil,
					nil,
					nil,
					tsgo.NodeFlagsNone,
				),
			)}, true),
			nil,
		),
	}, true)
}

func (b builder) unsafeVoidType() tsgo.TypeNode {
	return b.factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindVoidKeyword)
}
