package pointer

import "github.com/tsoniclang/gotots/internal/target/tsgo"

func (b builder) unsafeRawType() tsgo.TypeNode {
	read := b.factory.FunctionTypeNode(nil, nil, b.typeS())
	write := b.factory.FunctionTypeNode(
		nil,
		[]tsgo.ParameterDeclaration{b.parameter("value", b.typeS())},
		b.unsafeVoidType(),
	)
	return b.factory.TypeOperatorNode(
		tsgo.TypeOperatorNodeOperatorKindReadonlyKeyword,
		b.factory.TupleTypeNode([]tsgo.TypeNode{read, write}),
	)
}

func (b builder) unsafeRawProperty() tsgo.PropertyDeclaration {
	return b.factory.PropertyDeclaration(
		[]tsgo.ModifierLike{b.factory.PrivateKeyword()},
		b.id(unsafeRawName),
		b.factory.QuestionToken(),
		b.unsafeRawType(),
		nil,
	)
}

func (b builder) unsafeBindMethod() tsgo.MethodDeclaration {
	typeL := b.typeReference("L")
	typeS := b.typeReference("S")
	pointer := b.id("pointer")
	raw := b.id("raw")
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
		b.variable(
			tsgo.NodeFlagsConst,
			"raw",
			nil,
			b.binary(
				b.property(pointer, unsafeRawName),
				tsgo.BinaryOperatorQuestionQuestionEqualsToken,
				b.factory.ArrayLiteralExpression(
					[]tsgo.Expression{
						b.property(pointer, "read"),
						b.property(pointer, "write"),
					},
					false,
				),
			),
		),
		b.factory.ExpressionStatement(b.assign(
			b.property(pointer, "read"),
			b.factory.ArrowFunction(
				nil,
				nil,
				nil,
				nil,
				b.factory.EqualsGreaterThanToken(),
				b.factory.Block([]tsgo.Statement{
					b.factory.ExpressionStatement(b.factory.CallExpression(
						b.id("before"), nil, nil, nil, tsgo.NodeFlagsNone,
					)),
					b.factory.ReturnStatement(b.factory.CallExpression(
						b.factory.ElementAccessExpression(
							raw, nil, b.factory.NumericLiteral("0", tsgo.TokenFlagsNone), tsgo.NodeFlagsNone,
						),
						nil, nil, nil, tsgo.NodeFlagsNone,
					)),
				}, true),
			),
		)),
		b.factory.ExpressionStatement(b.assign(
			b.property(pointer, "write"),
			b.factory.ArrowFunction(
				nil,
				nil,
				[]tsgo.ParameterDeclaration{b.parameter("value", typeS)},
				nil,
				b.factory.EqualsGreaterThanToken(),
				b.factory.Block([]tsgo.Statement{
					b.factory.ExpressionStatement(b.factory.CallExpression(
						b.id("before"), nil, nil, nil, tsgo.NodeFlagsNone,
					)),
					b.factory.ExpressionStatement(b.factory.CallExpression(
						b.factory.ElementAccessExpression(
							raw, nil, b.factory.NumericLiteral("1", tsgo.TokenFlagsNone), tsgo.NodeFlagsNone,
						),
						nil, nil, []tsgo.Expression{b.id("value")}, tsgo.NodeFlagsNone,
					)),
					b.factory.ExpressionStatement(b.factory.CallExpression(
						b.id("after"), nil, nil, nil, tsgo.NodeFlagsNone,
					)),
				}, true),
			),
		)),
	)
}

func (b builder) unsafeVoidType() tsgo.TypeNode {
	return b.factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindVoidKeyword)
}
