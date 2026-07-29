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

func BuildAddressView(
	factory tsgo.Factory,
	functionName string,
	className string,
	pointerName string,
) tsgo.FunctionDeclaration {
	typeL := typeReference(factory, "L")
	typeS := typeReference(factory, "S")
	typeO := typeReference(factory, "O")
	sliceType := factory.TypeReferenceNode(
		factory.Identifier(className),
		[]tsgo.TypeNode{typeO},
	)
	pointerType := factory.TypeReferenceNode(
		factory.Identifier(pointerName),
		[]tsgo.TypeNode{typeL, typeS},
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
	toStorage := converterType(factory, typeO, typeS)
	fromStorage := converterType(factory, typeS, typeO)
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
			factory.Identifier("elementView"),
			tsgo.NodeFlagsNone,
		),
		nil,
		[]tsgo.TypeNode{typeL, typeS, typeO},
		[]tsgo.Expression{
			location,
			factory.Identifier("toStorage"),
			factory.Identifier("fromStorage"),
		},
		tsgo.NodeFlagsNone,
	)
	return factory.FunctionDeclaration(
		[]tsgo.ModifierLike{factory.ExportKeyword()},
		nil,
		factory.Identifier(functionName),
		[]tsgo.TypeParameterDeclaration{
			typeParameter(factory, "L"),
			typeParameter(factory, "S"),
			typeParameter(factory, "O"),
		},
		[]tsgo.ParameterDeclaration{
			parameter(factory, "value", sliceType),
			parameter(factory, "index", indexType),
			parameter(factory, "toStorage", toStorage),
			parameter(factory, "fromStorage", fromStorage),
		},
		pointerType,
		factory.Block(
			[]tsgo.Statement{factory.ReturnStatement(result)},
			true,
		),
	)
}

func typeReference(factory tsgo.Factory, name string) tsgo.TypeReferenceNode {
	return factory.TypeReferenceNode(factory.Identifier(name), nil)
}

func typeParameter(
	factory tsgo.Factory,
	name string,
) tsgo.TypeParameterDeclaration {
	return factory.TypeParameterDeclaration(
		nil,
		factory.Identifier(name),
		nil,
		nil,
		nil,
	)
}

func parameter(
	factory tsgo.Factory,
	name string,
	targetType tsgo.TypeNode,
) tsgo.ParameterDeclaration {
	return factory.ParameterDeclaration(
		nil,
		nil,
		factory.Identifier(name),
		nil,
		targetType,
		nil,
	)
}

func converterType(
	factory tsgo.Factory,
	source tsgo.TypeNode,
	target tsgo.TypeNode,
) tsgo.FunctionTypeNode {
	return factory.FunctionTypeNode(
		nil,
		[]tsgo.ParameterDeclaration{
			parameter(factory, "value", source),
		},
		target,
	)
}
