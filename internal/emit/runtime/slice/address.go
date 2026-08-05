package slice

import (
	panicruntime "github.com/tsoniclang/gotots/internal/emit/runtime/panic"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func (b builder) addressMethod() tsgo.MethodDeclaration {
	typeL := b.factory.TypeReferenceNode(b.id("L"), nil)
	pointerType := b.factory.TypeReferenceNode(
		b.id(b.pointerName),
		[]tsgo.TypeNode{typeL, b.typeT()},
	)
	return b.method(
		nil,
		MemberName(MemberAddress),
		[]tsgo.TypeParameterDeclaration{b.factory.TypeParameterDeclaration(
			nil,
			b.id("L"),
			nil,
			nil,
			nil,
		)},
		[]tsgo.ParameterDeclaration{
			b.parameter("index", b.integerInputType()),
		},
		pointerType,
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
		b.returnStatement(b.factory.CallExpression(
			b.property(b.id(b.pointerName), "element"),
			nil,
			[]tsgo.TypeNode{typeL, b.typeT()},
			[]tsgo.Expression{b.factory.ArrayLiteralExpression(
				[]tsgo.Expression{
					b.id("backing"),
					b.add(
						b.thisProperty("offset"),
						b.id("numericIndex"),
					),
				},
				false,
			)},
			tsgo.NodeFlagsNone,
		)),
	)
}

func (b projectionBuilder) addressMethod() tsgo.MethodDeclaration {
	typeL := b.typeReference("L")
	sourcePointer := b.factory.CallExpression(
		b.property(b.source(), MemberName(MemberAddress)),
		nil,
		[]tsgo.TypeNode{typeL},
		[]tsgo.Expression{b.id("index")},
		tsgo.NodeFlagsNone,
	)
	projected := b.factory.CallExpression(
		b.id(b.pointerProject),
		nil,
		[]tsgo.TypeNode{
			typeL,
			b.typeReference("F"),
			typeL,
			b.typeReference("T"),
		},
		[]tsgo.Expression{
			sourcePointer,
			b.thisProperty("fromSource"),
			b.thisProperty("toSource"),
		},
		tsgo.NodeFlagsNone,
	)
	return b.factory.MethodDeclaration(
		[]tsgo.ModifierLike{b.factory.OverrideKeyword()},
		nil,
		b.id(MemberName(MemberAddress)),
		nil,
		[]tsgo.TypeParameterDeclaration{b.typeParameter("L")},
		[]tsgo.ParameterDeclaration{
			b.parameter(nil, "index", b.integerInputType()),
		},
		b.pointerType(typeL, b.typeReference("T")),
		b.factory.Block([]tsgo.Statement{b.returnStatement(projected)}, true),
	)
}

func (b projectionBuilder) arrayLocationMethod() tsgo.MethodDeclaration {
	typeN := b.typeReference("N")
	sourceLocation := b.factory.CallExpression(
		b.property(b.source(), MemberName(MemberArrayLocation)),
		nil,
		[]tsgo.TypeNode{typeN},
		[]tsgo.Expression{b.id("length")},
		tsgo.NodeFlagsNone,
	)
	return b.factory.MethodDeclaration(
		[]tsgo.ModifierLike{b.factory.OverrideKeyword()},
		nil,
		b.id(MemberName(MemberArrayLocation)),
		nil,
		[]tsgo.TypeParameterDeclaration{b.factory.TypeParameterDeclaration(
			nil,
			b.id("N"),
			b.numberType(),
			nil,
			nil,
		)},
		[]tsgo.ParameterDeclaration{
			b.parameter(nil, "length", typeN),
		},
		b.factory.UnionTypeNode([]tsgo.TypeNode{
			b.factory.TypeOperatorNode(
				tsgo.TypeOperatorNodeOperatorKindReadonlyKeyword,
				b.factory.TupleTypeNode([]tsgo.TypeNode{
					b.factory.ArrayTypeNode(b.typeReference("T")),
					b.numberType(),
				}),
			),
			b.factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindUndefinedKeyword),
		}),
		b.factory.Block([]tsgo.Statement{
			b.variable(tsgo.NodeFlagsConst, "sourceLocation", sourceLocation),
			b.factory.IfStatement(
				b.factory.BinaryExpression(
					nil,
					b.id("sourceLocation"),
					nil,
					b.factory.BinaryOperatorToken(
						tsgo.BinaryOperatorEqualsEqualsEqualsToken,
					),
					b.factory.VoidExpression(b.number("0")),
				),
				b.factory.Block([]tsgo.Statement{
					b.returnStatement(b.factory.VoidExpression(b.number("0"))),
				}, true),
				nil,
			),
			b.returnStatement(panicruntime.Call(
				b.factory,
				b.panicName,
				b.factory.StringLiteral(
					"projected slice has no contiguous target representation",
					tsgo.TokenFlagsNone,
				),
			)),
		}, true),
	)
}

func BuildAddress(
	factory tsgo.Factory,
	functionName string,
	className string,
	pointerName string,
) tsgo.FunctionDeclaration {
	logicalParameter := typeParameter(factory, "L")
	storageParameter := typeParameter(factory, "S")
	sliceType := factory.TypeReferenceNode(
		factory.Identifier(className),
		[]tsgo.TypeNode{typeReference(factory, "S")},
	)
	pointerType := factory.TypeReferenceNode(
		factory.Identifier(pointerName),
		[]tsgo.TypeNode{
			typeReference(factory, "L"),
			typeReference(factory, "S"),
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
	result := factory.CallExpression(
		factory.PropertyAccessExpression(
			factory.Identifier("value"),
			nil,
			factory.Identifier(MemberName(MemberAddress)),
			tsgo.NodeFlagsNone,
		),
		nil,
		[]tsgo.TypeNode{typeReference(factory, "L")},
		[]tsgo.Expression{factory.Identifier("index")},
		tsgo.NodeFlagsNone,
	)
	return factory.FunctionDeclaration(
		[]tsgo.ModifierLike{factory.ExportKeyword()},
		nil,
		factory.Identifier(functionName),
		[]tsgo.TypeParameterDeclaration{
			logicalParameter,
			storageParameter,
		},
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
