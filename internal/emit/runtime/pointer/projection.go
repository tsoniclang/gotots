package pointer

import "github.com/tsoniclang/gotots/internal/target/tsgo"

func (b builder) cellMethod() tsgo.MethodDeclaration {
	storageValue := b.factory.ElementAccessExpression(
		b.id("storage"),
		nil,
		b.factory.NumericLiteral("0", tsgo.TokenFlagsNone),
		tsgo.NodeFlagsNone,
	)
	return b.method(
		[]tsgo.ModifierLike{b.factory.StaticKeyword()},
		CellName,
		[]tsgo.TypeParameterDeclaration{b.typeParameter("T", nil)},
		[]tsgo.ParameterDeclaration{b.parameter("value", b.typeT())},
		b.pointerType(b.typeT()),
		b.variable(
			tsgo.NodeFlagsConst,
			"storage",
			b.factory.TupleTypeNode([]tsgo.TypeNode{b.typeT()}),
			b.factory.ArrayLiteralExpression(
				[]tsgo.Expression{b.id("value")},
				false,
			),
		),
		b.factory.ReturnStatement(
			b.newPointer(
				b.typeT(),
				b.id("storage"),
				storageValue,
				storageValue,
			),
		),
	)
}

func (b builder) fieldMethod() tsgo.MethodDeclaration {
	typeO := b.typeReference("O")
	typeK := b.typeReference("K")
	valueType := b.factory.IndexedAccessTypeNode(typeO, typeK)
	parentValue := b.property(b.id("parent"), CellValueName)
	field := b.factory.ElementAccessExpression(
		parentValue,
		nil,
		b.id("key"),
		tsgo.NodeFlagsNone,
	)
	address := b.call(
		b.id(b.className),
		"child",
		b.property(b.id("parent"), "address"),
		b.id("key"),
	)
	return b.method(
		[]tsgo.ModifierLike{b.factory.StaticKeyword()},
		FieldName,
		[]tsgo.TypeParameterDeclaration{
			b.typeParameter("O", b.objectType()),
			b.typeParameter(
				"K",
				b.factory.TypeOperatorNode(
					tsgo.TypeOperatorNodeOperatorKindKeyOfKeyword,
					typeO,
				),
			),
		},
		[]tsgo.ParameterDeclaration{
			b.parameter("parent", b.pointerType(typeO)),
			b.parameter("key", typeK),
		},
		b.pointerType(valueType),
		b.factory.ReturnStatement(
			b.newPointer(valueType, address, field, field),
		),
	)
}

func (b builder) objectFieldMethod() tsgo.MethodDeclaration {
	typeO := b.typeReference("O")
	typeK := b.typeReference("K")
	valueType := b.factory.IndexedAccessTypeNode(typeO, typeK)
	field := b.factory.ElementAccessExpression(
		b.id("owner"),
		nil,
		b.id("key"),
		tsgo.NodeFlagsNone,
	)
	root := b.call(b.id(b.className), "root", b.id("owner"))
	address := b.call(b.id(b.className), "child", root, b.id("key"))
	return b.method(
		[]tsgo.ModifierLike{b.factory.StaticKeyword()},
		ObjectFieldName,
		[]tsgo.TypeParameterDeclaration{
			b.typeParameter("O", b.objectType()),
			b.typeParameter(
				"K",
				b.factory.TypeOperatorNode(
					tsgo.TypeOperatorNodeOperatorKindKeyOfKeyword,
					typeO,
				),
			),
		},
		[]tsgo.ParameterDeclaration{
			b.parameter("owner", typeO),
			b.parameter("key", typeK),
		},
		b.pointerType(valueType),
		b.factory.ReturnStatement(
			b.newPointer(valueType, address, field, field),
		),
	)
}

func (b builder) elementMethod() tsgo.MethodDeclaration {
	arrayType := b.factory.ArrayTypeNode(b.typeT())
	locationType := b.factory.TypeOperatorNode(
		tsgo.TypeOperatorNodeOperatorKindReadonlyKeyword,
		b.factory.TupleTypeNode(
			[]tsgo.TypeNode{arrayType, b.numberType()},
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
	return b.method(
		[]tsgo.ModifierLike{b.factory.StaticKeyword()},
		ElementName,
		[]tsgo.TypeParameterDeclaration{b.typeParameter("T", nil)},
		[]tsgo.ParameterDeclaration{
			b.parameter("location", locationType),
		},
		b.pointerType(b.typeT()),
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
			b.newPointer(
				b.typeT(),
				b.call(
					b.id(b.className),
					"child",
					b.call(b.id(b.className), "root", backing),
					index,
				),
				element,
				element,
			),
		),
	)
}

func (b builder) indexMethod() tsgo.MethodDeclaration {
	typeT := b.typeReference("T")
	typeO := b.typeReference("O")
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
				typeT,
			),
			b.factory.MethodSignatureDeclaration(
				nil,
				b.id("set"),
				nil,
				nil,
				[]tsgo.ParameterDeclaration{
					b.parameter("index", integerType),
					b.parameter("value", typeT),
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
		b.id("Number"),
		nil,
		nil,
		[]tsgo.Expression{b.id("index")},
		tsgo.NodeFlagsNone,
	)
	read := b.call(selectedValue, "get", b.id("numericIndex"))
	write := b.call(
		selectedValue,
		"set",
		b.id("numericIndex"),
		b.id("next"),
	)
	return b.method(
		[]tsgo.ModifierLike{b.factory.StaticKeyword()},
		IndexName,
		[]tsgo.TypeParameterDeclaration{
			b.typeParameter("T", nil),
			b.typeParameter("O", indexedType),
		},
		[]tsgo.ParameterDeclaration{
			b.parameter("parent", b.pointerType(typeO)),
			b.parameter("index", integerType),
		},
		b.pointerType(typeT),
		b.variable(tsgo.NodeFlagsConst, "selected", nil, selected),
		b.variable(tsgo.NodeFlagsConst, "numericIndex", nil, numericIndex),
		b.factory.ExpressionStatement(read),
		b.factory.ReturnStatement(
			b.newPointerWithWrite(
				typeT,
				b.call(
					b.id(b.className),
					"child",
					b.property(b.id("selected"), "address"),
					b.id("numericIndex"),
				),
				read,
				write,
			),
		),
	)
}
