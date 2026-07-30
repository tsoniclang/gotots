package pointer

import (
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func (b builder) elementViewMethod() tsgo.MethodDeclaration {
	typeL := b.typeReference("L")
	typeS := b.typeReference("S")
	typeO := b.typeReference("O")
	backingType := b.factory.ArrayTypeNode(typeO)
	locationType := b.factory.TypeOperatorNode(
		tsgo.TypeOperatorNodeOperatorKindReadonlyKeyword,
		b.factory.TupleTypeNode(
			[]tsgo.TypeNode{backingType, b.numberType()},
		),
	)
	backing := b.id("backing")
	index := b.id("index")
	element := b.factory.ElementAccessExpression(
		backing,
		nil,
		index,
		tsgo.NodeFlagsNone,
	)
	read := b.functionCall(b.id("toStorage"), element)
	write := b.assign(
		element,
		b.functionCall(b.id("fromStorage"), b.id("next")),
	)
	return b.method(
		[]tsgo.ModifierLike{b.factory.StaticKeyword()},
		ElementViewName,
		[]tsgo.TypeParameterDeclaration{
			b.typeParameter("L", nil),
			b.typeParameter("S", nil),
			b.typeParameter("O", nil),
		},
		[]tsgo.ParameterDeclaration{
			b.parameter("location", locationType),
			b.parameter("toStorage", b.converterType(typeO, typeS)),
			b.parameter("fromStorage", b.converterType(typeS, typeO)),
		},
		b.pointerType(typeL, typeS),
		b.variable(
			tsgo.NodeFlagsConst,
			"backing",
			nil,
			b.factory.ElementAccessExpression(
				b.id("location"),
				nil,
				b.factory.NumericLiteral("0", tsgo.TokenFlagsNone),
				tsgo.NodeFlagsNone,
			),
		),
		b.variable(
			tsgo.NodeFlagsConst,
			"index",
			nil,
			b.factory.ElementAccessExpression(
				b.id("location"),
				nil,
				b.factory.NumericLiteral("1", tsgo.TokenFlagsNone),
				tsgo.NodeFlagsNone,
			),
		),
		b.factory.ReturnStatement(
			b.newPointerWithWrite(
				typeL,
				typeS,
				b.call(
					b.id(b.className),
					"child",
					b.call(b.id(b.className), "root", backing),
					index,
				),
				read,
				write,
			),
		),
	)
}

func (b builder) indexViewMethod() tsgo.MethodDeclaration {
	typeL := b.typeReference("L")
	typeS := b.typeReference("S")
	typePL := b.typeReference("PL")
	typeO := b.typeReference("O")
	typeV := b.typeReference("V")
	integerType := b.factory.UnionTypeNode(
		[]tsgo.TypeNode{
			b.numberType(),
			b.factory.KeywordTypeNode(
				tsgo.KeywordTypeSyntaxKindBigIntKeyword,
			),
		},
	)
	indexedType := b.factory.TypeLiteralNode(
		[]tsgo.TypeElement{
			b.factory.MethodSignatureDeclaration(
				nil,
				b.id("get"),
				nil,
				nil,
				[]tsgo.ParameterDeclaration{
					b.parameter("index", integerType),
				},
				typeV,
			),
			b.factory.MethodSignatureDeclaration(
				nil,
				b.id("set"),
				nil,
				nil,
				[]tsgo.ParameterDeclaration{
					b.parameter("index", integerType),
					b.parameter("value", typeV),
				},
				b.factory.KeywordTypeNode(
					tsgo.KeywordTypeSyntaxKindVoidKeyword,
				),
			),
		},
	)
	selected := b.call(
		b.id(b.className),
		DereferenceName,
		b.id("parent"),
	)
	selectedValue := b.property(b.id("selected"), CellValueName)
	numericIndex := b.factory.CallExpression(
		api.TargetIntrinsicNumber.Expression(b.factory),
		nil,
		nil,
		[]tsgo.Expression{b.id("index")},
		tsgo.NodeFlagsNone,
	)
	stored := b.call(selectedValue, "get", b.id("numericIndex"))
	read := b.functionCall(b.id("toStorage"), stored)
	write := b.call(
		selectedValue,
		"set",
		b.id("numericIndex"),
		b.functionCall(b.id("fromStorage"), b.id("next")),
	)
	return b.method(
		[]tsgo.ModifierLike{b.factory.StaticKeyword()},
		IndexViewName,
		[]tsgo.TypeParameterDeclaration{
			b.typeParameter("L", nil),
			b.typeParameter("S", nil),
			b.typeParameter("PL", nil),
			b.typeParameter("V", nil),
			b.typeParameter("O", indexedType),
		},
		[]tsgo.ParameterDeclaration{
			b.parameter("parent", b.pointerType(typePL, typeO)),
			b.parameter("index", integerType),
			b.parameter("toStorage", b.converterType(typeV, typeS)),
			b.parameter("fromStorage", b.converterType(typeS, typeV)),
		},
		b.pointerType(typeL, typeS),
		b.variable(tsgo.NodeFlagsConst, "selected", nil, selected),
		b.variable(tsgo.NodeFlagsConst, "numericIndex", nil, numericIndex),
		b.factory.ReturnStatement(
			b.newPointerWithWrite(
				typeL,
				typeS,
				b.call(
					b.id(b.className),
					"child",
					b.property(b.id("selected"), AddressName),
					b.id("numericIndex"),
				),
				read,
				write,
			),
		),
	)
}

func (b builder) converterType(
	source tsgo.TypeNode,
	target tsgo.TypeNode,
) tsgo.FunctionTypeNode {
	return b.factory.FunctionTypeNode(
		nil,
		[]tsgo.ParameterDeclaration{b.parameter("value", source)},
		target,
	)
}

func (b builder) functionCall(
	function tsgo.Expression,
	argument tsgo.Expression,
) tsgo.CallExpression {
	return b.factory.CallExpression(
		function,
		nil,
		nil,
		[]tsgo.Expression{argument},
		tsgo.NodeFlagsNone,
	)
}
