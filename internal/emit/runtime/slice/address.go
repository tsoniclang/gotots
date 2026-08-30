package slice

import (
	panicruntime "github.com/tsoniclang/gotots/internal/emit/runtime/panic"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func (b builder) addressMethod() tsgo.MethodDeclaration {
	return b.method(
		nil,
		MemberName(MemberAddress),
		nil,
		[]tsgo.ParameterDeclaration{
			b.parameter("index", b.integerInputType()),
		},
		b.pointerType(b.typeT()),
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
			b.id(b.addressName),
			nil,
			[]tsgo.TypeNode{b.typeT()},
			[]tsgo.Expression{
				b.backingElement(b.id("backing"), b.id("numericIndex")),
			},
			tsgo.NodeFlagsNone,
		)),
	)
}

func (b builder) pointerType(element tsgo.TypeNode) tsgo.TypeReferenceNode {
	return b.factory.TypeReferenceNode(
		b.id(b.pointerName),
		[]tsgo.TypeNode{element},
	)
}

func (b builder) projectedAddressMethod() tsgo.MethodDeclaration {
	targetType := b.factory.TypeReferenceNode(b.id("U"), nil)
	numericIndex := b.id("numericIndex")
	backing := b.id("backing")
	address := b.factory.CallExpression(
		b.id(b.addressName),
		nil,
		[]tsgo.TypeNode{b.typeT()},
		[]tsgo.Expression{b.backingElement(backing, numericIndex)},
		tsgo.NodeFlagsNone,
	)
	projected := b.factory.CallExpression(
		b.id(b.pointerProjectName),
		nil,
		[]tsgo.TypeNode{b.typeT(), targetType},
		[]tsgo.Expression{
			address,
			b.id("fromSource"),
			b.id("toSource"),
		},
		tsgo.NodeFlagsNone,
	)
	return b.method(
		nil,
		StorageProjectedAddressMember,
		[]tsgo.TypeParameterDeclaration{b.factory.TypeParameterDeclaration(
			nil,
			b.id("U"),
			nil,
			nil,
			nil,
		)},
		[]tsgo.ParameterDeclaration{
			b.parameter("index", b.integerInputType()),
			b.parameter(
				"fromSource",
				b.factory.FunctionTypeNode(
					nil,
					[]tsgo.ParameterDeclaration{b.parameter("value", b.typeT())},
					targetType,
				),
			),
			b.parameter(
				"toSource",
				b.factory.FunctionTypeNode(
					nil,
					[]tsgo.ParameterDeclaration{b.parameter("value", targetType)},
					b.typeT(),
				),
			),
		},
		b.pointerType(targetType),
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
					backing,
					tsgo.BinaryOperatorEqualsEqualsEqualsToken,
					b.factory.NullLiteral(),
				),
				tsgo.BinaryOperatorBarBarToken,
				b.boundsCondition(numericIndex),
			),
			b.throwBounds(),
			nil,
		),
		b.returnStatement(projected),
	)
}

func (b projectionBuilder) addressMethod() tsgo.MethodDeclaration {
	targetType := b.typeReference("T")
	projected := b.factory.CallExpression(
		b.property(b.source(), StorageProjectedAddressMember),
		nil,
		[]tsgo.TypeNode{targetType},
		[]tsgo.Expression{
			b.id("index"),
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
		nil,
		[]tsgo.ParameterDeclaration{
			b.parameter(nil, "index", b.integerInputType()),
		},
		b.pointerType(targetType),
		b.factory.Block([]tsgo.Statement{b.returnStatement(projected)}, true),
	)
}

func (b projectionBuilder) projectedAddressMethod() tsgo.MethodDeclaration {
	targetType := b.typeReference("U")
	fromSource := b.factory.ArrowFunction(
		nil,
		nil,
		[]tsgo.ParameterDeclaration{
			b.parameter(nil, "value", b.typeReference("F")),
		},
		targetType,
		b.factory.EqualsGreaterThanToken(),
		b.factory.CallExpression(
			b.id("fromTarget"),
			nil,
			nil,
			[]tsgo.Expression{b.convert("fromSource", b.id("value"))},
			tsgo.NodeFlagsNone,
		),
	)
	toSource := b.factory.ArrowFunction(
		nil,
		nil,
		[]tsgo.ParameterDeclaration{
			b.parameter(nil, "value", targetType),
		},
		b.typeReference("F"),
		b.factory.EqualsGreaterThanToken(),
		b.convert(
			"toSource",
			b.factory.CallExpression(
				b.id("toTarget"),
				nil,
				nil,
				[]tsgo.Expression{b.id("value")},
				tsgo.NodeFlagsNone,
			),
		),
	)
	return b.factory.MethodDeclaration(
		[]tsgo.ModifierLike{b.factory.OverrideKeyword()},
		nil,
		b.id(StorageProjectedAddressMember),
		nil,
		[]tsgo.TypeParameterDeclaration{b.typeParameter("U")},
		[]tsgo.ParameterDeclaration{
			b.parameter(nil, "index", b.integerInputType()),
			b.parameter(nil, "fromTarget", b.converterType("T", "U")),
			b.parameter(nil, "toTarget", b.converterType("U", "T")),
		},
		b.pointerType(targetType),
		b.factory.Block([]tsgo.Statement{b.returnStatement(
			b.factory.CallExpression(
				b.property(b.source(), StorageProjectedAddressMember),
				nil,
				[]tsgo.TypeNode{targetType},
				[]tsgo.Expression{b.id("index"), fromSource, toSource},
				tsgo.NodeFlagsNone,
			),
		)}, true),
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
	typeT := typeReference(factory, "T")
	sliceType := factory.TypeReferenceNode(
		factory.Identifier(className),
		[]tsgo.TypeNode{typeT},
	)
	pointerType := factory.TypeReferenceNode(
		factory.Identifier(pointerName),
		[]tsgo.TypeNode{typeT},
	)
	indexType := factory.UnionTypeNode(
		[]tsgo.TypeNode{
			factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindNumberKeyword),
			factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindBigIntKeyword),
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
		nil,
		[]tsgo.Expression{factory.Identifier("index")},
		tsgo.NodeFlagsNone,
	)
	return factory.FunctionDeclaration(
		[]tsgo.ModifierLike{factory.ExportKeyword()},
		nil,
		factory.Identifier(functionName),
		[]tsgo.TypeParameterDeclaration{typeParameter(factory, "T")},
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
	typeNode tsgo.TypeNode,
) tsgo.ParameterDeclaration {
	return factory.ParameterDeclaration(
		nil,
		nil,
		factory.Identifier(name),
		nil,
		typeNode,
		nil,
	)
}
