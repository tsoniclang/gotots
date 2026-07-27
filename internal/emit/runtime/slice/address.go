package slice

import "github.com/tsoniclang/gotots/internal/target/tsgo"

func (b builder) addressMethod() tsgo.MethodDeclaration {
	tuple := b.factory.TypeOperatorNode(
		tsgo.TypeOperatorNodeOperatorKindReadonlyKeyword,
		b.factory.TupleTypeNode(
			[]tsgo.TypeNode{
				b.factory.ArrayTypeNode(b.typeT()),
				b.numberType(),
			},
		),
	)
	return b.method(
		nil,
		MemberName(MemberAddress),
		nil,
		[]tsgo.ParameterDeclaration{
			b.parameter("index", b.integerInputType()),
		},
		tuple,
		b.variable(
			tsgo.NodeFlagsConst,
			"numericIndex",
			b.toNumber(b.id("index")),
		),
		b.variable(
			tsgo.NodeFlagsConst,
			"backing",
			b.thisProperty("backing"),
		),
		b.factory.IfStatement(
			b.binary(
				b.binary(
					b.id("backing"),
					tsgo.BinaryOperatorEqualsEqualsEqualsToken,
					b.factory.NullLiteral(),
				),
				tsgo.BinaryOperatorBarBarToken,
				b.boundsCondition(b.id("numericIndex")),
			),
			b.throwBounds(),
			nil,
		),
		b.returnStatement(
			b.factory.ArrayLiteralExpression(
				[]tsgo.Expression{
					b.id("backing"),
					b.add(
						b.thisProperty("offset"),
						b.id("numericIndex"),
					),
				},
				false,
			),
		),
	)
}

func BuildAddress(
	factory tsgo.Factory,
	functionName string,
	className string,
	pointerName string,
) tsgo.FunctionDeclaration {
	typeParameter := factory.TypeParameterDeclaration(
		nil,
		factory.Identifier("T"),
		nil,
		nil,
		nil,
	)
	sliceType := factory.TypeReferenceNode(
		factory.Identifier(className),
		[]tsgo.TypeNode{
			factory.TypeReferenceNode(factory.Identifier("T"), nil),
		},
	)
	pointerType := factory.TypeReferenceNode(
		factory.Identifier(pointerName),
		[]tsgo.TypeNode{
			factory.TypeReferenceNode(factory.Identifier("T"), nil),
		},
	)
	indexType := factory.UnionTypeNode(
		[]tsgo.TypeNode{
			factory.KeywordTypeNode(
				tsgo.KeywordTypeSyntaxKindNumberKeyword,
			),
			factory.KeywordTypeNode(
				tsgo.KeywordTypeSyntaxKindBigIntKeyword,
			),
		},
	)
	location := factory.CallExpression(
		factory.PropertyAccessExpression(
			factory.Identifier("value"),
			nil,
			factory.Identifier(MemberName(MemberAddress)),
			tsgo.NodeFlagsNone,
		),
		nil,
		nil,
		[]tsgo.Expression{factory.Identifier("index")},
		tsgo.NodeFlagsNone,
	)
	result := factory.CallExpression(
		factory.PropertyAccessExpression(
			factory.Identifier(pointerName),
			nil,
			factory.Identifier("element"),
			tsgo.NodeFlagsNone,
		),
		nil,
		[]tsgo.TypeNode{
			factory.TypeReferenceNode(factory.Identifier("T"), nil),
		},
		[]tsgo.Expression{location},
		tsgo.NodeFlagsNone,
	)
	return factory.FunctionDeclaration(
		[]tsgo.ModifierLike{factory.ExportKeyword()},
		nil,
		factory.Identifier(functionName),
		[]tsgo.TypeParameterDeclaration{typeParameter},
		[]tsgo.ParameterDeclaration{
			factory.ParameterDeclaration(
				nil,
				nil,
				factory.Identifier("value"),
				nil,
				sliceType,
				nil,
			),
			factory.ParameterDeclaration(
				nil,
				nil,
				factory.Identifier("index"),
				nil,
				indexType,
				nil,
			),
		},
		pointerType,
		factory.Block(
			[]tsgo.Statement{factory.ReturnStatement(result)},
			true,
		),
	)
}
