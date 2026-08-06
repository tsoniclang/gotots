package pointer

import (
	"github.com/tsoniclang/gotots/internal/emit/api"
	indexedstorage "github.com/tsoniclang/gotots/internal/emit/runtime/indexedstorage"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

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
		[]tsgo.TypeParameterDeclaration{
			b.typeParameter("L", nil),
			b.typeParameter("S", nil),
		},
		[]tsgo.ParameterDeclaration{b.parameter("value", b.typeS())},
		b.pointerType(b.typeL(), b.typeS()),
		b.variable(
			tsgo.NodeFlagsConst,
			"storage",
			b.factory.TupleTypeNode([]tsgo.TypeNode{b.typeS()}),
			b.factory.ArrayLiteralExpression(
				[]tsgo.Expression{b.id("value")},
				false,
			),
		),
		b.factory.ReturnStatement(
			b.newPointerWithRegion(
				b.typeL(),
				b.typeS(),
				b.id("storage"),
				storageValue,
				storageValue,
				b.factory.ArrayLiteralExpression(
					[]tsgo.Expression{b.id("storage"), b.factory.NumericLiteral("0", tsgo.TokenFlagsNone)},
					false,
				),
			),
		),
	)
}

func (b builder) fieldMethod() tsgo.MethodDeclaration {
	typeL := b.typeReference("L")
	typePL := b.typeReference("PL")
	typeO := b.typeReference("PS")
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
		b.property(b.id("parent"), AddressName),
		b.id("key"),
	)
	return b.method(
		[]tsgo.ModifierLike{b.factory.StaticKeyword()},
		FieldName,
		[]tsgo.TypeParameterDeclaration{
			b.typeParameter("L", nil),
			b.typeParameter("PL", nil),
			b.typeParameter("PS", b.objectType()),
			b.typeParameter(
				"K",
				b.factory.TypeOperatorNode(
					tsgo.TypeOperatorNodeOperatorKindKeyOfKeyword,
					typeO,
				),
			),
		},
		[]tsgo.ParameterDeclaration{
			b.parameter("parent", b.pointerType(typePL, typeO)),
			b.parameter("key", typeK),
		},
		b.pointerType(typeL, valueType),
		b.factory.ReturnStatement(
			b.newPointer(typeL, valueType, address, field, field),
		),
	)
}

func (b builder) objectFieldMethod() tsgo.MethodDeclaration {
	typeL := b.typeReference("L")
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
			b.typeParameter("L", nil),
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
		b.pointerType(typeL, valueType),
		b.factory.ReturnStatement(
			b.newPointer(typeL, valueType, address, field, field),
		),
	)
}

func (b builder) elementMethod() tsgo.MethodDeclaration {
	typeL := b.typeReference("L")
	typeS := b.typeReference("S")
	arrayType := b.factory.ArrayTypeNode(typeS)
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
	readElement := indexedstorage.Element(
		b.factory,
		b.denseIndexName,
		backing,
		index,
	)
	return b.method(
		[]tsgo.ModifierLike{b.factory.StaticKeyword()},
		ElementName,
		[]tsgo.TypeParameterDeclaration{
			b.typeParameter("L", nil),
			b.typeParameter("S", nil),
		},
		[]tsgo.ParameterDeclaration{
			b.parameter("location", locationType),
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
			b.newPointerWithRegion(
				typeL,
				typeS,
				b.call(
					b.id(b.className),
					"child",
					b.call(b.id(b.className), "root", backing),
					index,
				),
				readElement,
				element,
				b.id("location"),
			),
		),
	)
}

func (b builder) indexMethod() tsgo.MethodDeclaration {
	typeL := b.typeReference("L")
	typeS := b.typeReference("S")
	typePL := b.typeReference("PL")
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
				typeS,
			),
			b.factory.MethodSignatureDeclaration(
				nil,
				b.id("set"),
				nil,
				nil,
				[]tsgo.ParameterDeclaration{
					b.parameter("index", integerType),
					b.parameter("value", typeS),
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
			b.typeParameter("L", nil),
			b.typeParameter("S", nil),
			b.typeParameter("PL", nil),
			b.typeParameter("O", indexedType),
		},
		[]tsgo.ParameterDeclaration{
			b.parameter("parent", b.pointerType(typePL, typeO)),
			b.parameter("index", integerType),
		},
		b.pointerType(typeL, typeS),
		b.variable(tsgo.NodeFlagsConst, "selected", nil, selected),
		b.variable(tsgo.NodeFlagsConst, "numericIndex", nil, numericIndex),
		b.factory.ExpressionStatement(read),
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

func (b builder) projectMethod() tsgo.MethodDeclaration {
	typeFL := b.typeReference("FL")
	typeFS := b.typeReference("FS")
	typeTL := b.typeReference("TL")
	typeTS := b.typeReference("TS")
	pointer := b.id("pointer")
	read := b.factory.CallExpression(
		b.id("fromSource"),
		nil,
		nil,
		[]tsgo.Expression{b.property(pointer, CellValueName)},
		tsgo.NodeFlagsNone,
	)
	write := b.factory.BinaryExpression(
		nil,
		b.property(pointer, CellValueName),
		nil,
		b.factory.BinaryOperatorToken(tsgo.BinaryOperatorEqualsToken),
		b.factory.CallExpression(
			b.id("toSource"),
			nil,
			nil,
			[]tsgo.Expression{b.id("next")},
			tsgo.NodeFlagsNone,
		),
	)
	converter := func(
		name string,
		from tsgo.TypeNode,
		to tsgo.TypeNode,
	) tsgo.ParameterDeclaration {
		return b.parameter(
			name,
			b.factory.FunctionTypeNode(
				nil,
				[]tsgo.ParameterDeclaration{b.parameter("value", from)},
				to,
			),
		)
	}
	return b.method(
		[]tsgo.ModifierLike{b.factory.StaticKeyword()},
		ProjectName,
		[]tsgo.TypeParameterDeclaration{
			b.typeParameter("FL", nil),
			b.typeParameter("FS", nil),
			b.typeParameter("TL", nil),
			b.typeParameter("TS", nil),
		},
		[]tsgo.ParameterDeclaration{
			b.parameter("pointer", b.pointerType(typeFL, typeFS)),
			converter("fromSource", typeFS, typeTS),
			converter("toSource", typeTS, typeFS),
		},
		b.pointerType(typeTL, typeTS),
		b.factory.ReturnStatement(b.newPointerWithWrite(
			typeTL,
			typeTS,
			b.property(pointer, AddressName),
			read,
			write,
		)),
	)
}

func Project(
	factory tsgo.Factory,
	functionName string,
	pointerName string,
) tsgo.FunctionDeclaration {
	b := builder{factory: factory, className: pointerName}
	typeFL := b.typeReference("FL")
	typeFS := b.typeReference("FS")
	typeTL := b.typeReference("TL")
	typeTS := b.typeReference("TS")
	converter := func(
		name string,
		from tsgo.TypeNode,
		to tsgo.TypeNode,
	) tsgo.ParameterDeclaration {
		return b.parameter(
			name,
			factory.FunctionTypeNode(
				nil,
				[]tsgo.ParameterDeclaration{b.parameter("value", from)},
				to,
			),
		)
	}
	return factory.FunctionDeclaration(
		[]tsgo.ModifierLike{factory.ExportKeyword()},
		nil,
		factory.Identifier(functionName),
		[]tsgo.TypeParameterDeclaration{
			b.typeParameter("FL", nil),
			b.typeParameter("FS", nil),
			b.typeParameter("TL", nil),
			b.typeParameter("TS", nil),
		},
		[]tsgo.ParameterDeclaration{
			b.parameter("pointer", b.pointerType(typeFL, typeFS)),
			converter("fromSource", typeFS, typeTS),
			converter("toSource", typeTS, typeFS),
		},
		b.pointerType(typeTL, typeTS),
		factory.Block([]tsgo.Statement{factory.ReturnStatement(
			factory.CallExpression(
				b.property(factory.Identifier(pointerName), ProjectName),
				nil,
				[]tsgo.TypeNode{typeFL, typeFS, typeTL, typeTS},
				[]tsgo.Expression{
					factory.Identifier("pointer"),
					factory.Identifier("fromSource"),
					factory.Identifier("toSource"),
				},
				tsgo.NodeFlagsNone,
			),
		)}, true),
	)
}
