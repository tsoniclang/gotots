package slice

import "github.com/tsoniclang/gotots/internal/target/tsgo"

func (b builder) arrayLocationMethod() tsgo.MethodDeclaration {
	lengthType := b.factory.TypeReferenceNode(b.id("N"), nil)
	locationType := b.factory.TypeOperatorNode(
		tsgo.TypeOperatorNodeOperatorKindReadonlyKeyword,
		b.factory.TupleTypeNode([]tsgo.TypeNode{
			b.factory.ArrayTypeNode(b.typeT()),
			b.numberType(),
		}),
	)
	resultType := b.factory.UnionTypeNode([]tsgo.TypeNode{
		locationType,
		b.factory.KeywordTypeNode(
			tsgo.KeywordTypeSyntaxKindUndefinedKeyword,
		),
	})
	return b.method(
		nil,
		MemberName(MemberArrayLocation),
		[]tsgo.TypeParameterDeclaration{
			b.factory.TypeParameterDeclaration(
				nil,
				b.id("N"),
				b.numberType(),
				nil,
				nil,
			),
		},
		[]tsgo.ParameterDeclaration{
			b.parameter("length", lengthType),
		},
		resultType,
		b.variable(
			tsgo.NodeFlagsConst,
			"requested",
			b.toNumber(b.id("length")),
		),
		b.factory.IfStatement(
			b.binary(
				b.thisProperty(MemberName(MemberLength)),
				tsgo.BinaryOperatorLessThanToken,
				b.id("requested"),
			),
			b.throwBounds(),
			nil,
		),
		b.variable(
			tsgo.NodeFlagsConst,
			"backing",
			b.thisProperty("backing"),
		),
		b.factory.IfStatement(
			b.binary(
				b.id("backing"),
				tsgo.BinaryOperatorEqualsEqualsEqualsToken,
				b.factory.NullLiteral(),
			),
			b.factory.Block([]tsgo.Statement{
				b.returnStatement(
					b.factory.VoidExpression(
						b.number("0"),
					),
				),
			}, true),
			nil,
		),
		b.returnStatement(b.factory.ArrayLiteralExpression(
			[]tsgo.Expression{
				b.id("backing"),
				b.thisProperty("offset"),
			},
			false,
		)),
	)
}

func BuildArrayPointer(
	factory tsgo.Factory,
	functionName string,
	sliceName string,
	pointerName string,
	arrayName string,
	arrayViewName string,
) tsgo.FunctionDeclaration {
	typeL := typeReference(factory, "L")
	typeT := typeReference(factory, "T")
	typeN := typeReference(factory, "N")
	arrayType := factory.TypeReferenceNode(
		factory.Identifier(arrayName),
		[]tsgo.TypeNode{typeT, typeN},
	)
	pointerType := factory.TypeReferenceNode(
		factory.Identifier(pointerName),
		[]tsgo.TypeNode{typeL, arrayType},
	)
	resultType := factory.UnionTypeNode([]tsgo.TypeNode{
		pointerType,
		factory.KeywordTypeNode(
			tsgo.KeywordTypeSyntaxKindUndefinedKeyword,
		),
	})
	location := factory.Identifier("location")
	locationElement := func(index string) tsgo.ElementAccessExpression {
		return factory.ElementAccessExpression(
			location,
			nil,
			factory.NumericLiteral(index, tsgo.TokenFlagsNone),
			tsgo.NodeFlagsNone,
		)
	}
	view := factory.CallExpression(
		factory.Identifier(arrayViewName),
		nil,
		[]tsgo.TypeNode{typeT, typeN},
		[]tsgo.Expression{
			locationElement("0"),
			locationElement("1"),
			factory.Identifier("length"),
		},
		tsgo.NodeFlagsNone,
	)
	result := factory.CallExpression(
		factory.PropertyAccessExpression(
			factory.Identifier(pointerName),
			nil,
			factory.Identifier("arrayRegion"),
			tsgo.NodeFlagsNone,
		),
		nil,
		[]tsgo.TypeNode{typeL, typeT, arrayType},
		[]tsgo.Expression{location, view},
		tsgo.NodeFlagsNone,
	)
	return factory.FunctionDeclaration(
		[]tsgo.ModifierLike{factory.ExportKeyword()},
		nil,
		factory.Identifier(functionName),
		[]tsgo.TypeParameterDeclaration{
			typeParameter(factory, "L"),
			typeParameter(factory, "T"),
			factory.TypeParameterDeclaration(
				nil,
				factory.Identifier("N"),
				factory.KeywordTypeNode(
					tsgo.KeywordTypeSyntaxKindNumberKeyword,
				),
				nil,
				nil,
			),
		},
		[]tsgo.ParameterDeclaration{
			parameter(
				factory,
				"value",
				factory.TypeReferenceNode(
					factory.Identifier(sliceName),
					[]tsgo.TypeNode{typeT},
				),
			),
			parameter(factory, "length", typeN),
		},
		resultType,
		factory.Block([]tsgo.Statement{
			arrayPointerVariable(
				factory,
				tsgo.NodeFlagsConst,
				"location",
				factory.CallExpression(
					factory.PropertyAccessExpression(
						factory.Identifier("value"),
						nil,
						factory.Identifier(
							MemberName(MemberArrayLocation),
						),
						tsgo.NodeFlagsNone,
					),
					nil,
					[]tsgo.TypeNode{typeN},
					[]tsgo.Expression{factory.Identifier("length")},
					tsgo.NodeFlagsNone,
				),
			),
			factory.IfStatement(
				factory.BinaryExpression(
					nil,
					location,
					nil,
					factory.BinaryOperatorToken(
						tsgo.BinaryOperatorEqualsEqualsEqualsToken,
					),
					factory.Identifier("undefined"),
				),
				factory.Block([]tsgo.Statement{
					factory.ReturnStatement(
						factory.VoidExpression(
							factory.NumericLiteral(
								"0",
								tsgo.TokenFlagsNone,
							),
						),
					),
				}, true),
				nil,
			),
			factory.ReturnStatement(result),
		}, true),
	)
}

func arrayPointerVariable(
	factory tsgo.Factory,
	flags tsgo.NodeFlags,
	name string,
	value tsgo.Expression,
) tsgo.VariableStatement {
	return factory.VariableStatement(
		nil,
		factory.VariableDeclarationList(
			[]tsgo.VariableDeclaration{
				factory.VariableDeclaration(
					factory.Identifier(name),
					nil,
					nil,
					value,
				),
			},
			flags,
		),
	)
}
