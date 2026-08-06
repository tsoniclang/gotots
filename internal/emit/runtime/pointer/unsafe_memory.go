package pointer

import "github.com/tsoniclang/gotots/internal/target/tsgo"

func (b builder) unsafeMemoryType(storage tsgo.TypeNode) tsgo.TypeNode {
	readType := b.factory.FunctionTypeNode(nil, nil, storage)
	writeType := b.factory.FunctionTypeNode(
		nil,
		[]tsgo.ParameterDeclaration{b.parameter("value", storage)},
		b.factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindVoidKeyword),
	)
	return b.factory.TypeOperatorNode(
		tsgo.TypeOperatorNodeOperatorKindReadonlyKeyword,
		b.factory.TupleTypeNode([]tsgo.TypeNode{
			b.objectType(),
			readType,
			writeType,
			b.optionalRegionType(storage),
		}),
	)
}

func (b builder) unsafeMemoryMethod() tsgo.MethodDeclaration {
	pointer := b.id("pointer")
	read := b.factory.ArrowFunction(
		nil,
		nil,
		nil,
		nil,
		b.factory.EqualsGreaterThanToken(),
		b.factory.CallExpression(
			b.property(pointer, "read"),
			nil,
			nil,
			nil,
			tsgo.NodeFlagsNone,
		),
	)
	next := b.id("next")
	write := b.factory.ArrowFunction(
		nil,
		nil,
		[]tsgo.ParameterDeclaration{b.parameter("next", b.typeS())},
		nil,
		b.factory.EqualsGreaterThanToken(),
		b.factory.CallExpression(
			b.property(pointer, "write"),
			nil,
			nil,
			[]tsgo.Expression{next},
			tsgo.NodeFlagsNone,
		),
	)
	return b.method(
		[]tsgo.ModifierLike{b.factory.StaticKeyword()},
		UnsafeMemoryName,
		[]tsgo.TypeParameterDeclaration{
			b.typeParameter("L", nil),
			b.typeParameter("S", nil),
		},
		[]tsgo.ParameterDeclaration{
			b.parameter("pointer", b.pointerType(b.typeL(), b.typeS())),
		},
		b.unsafeMemoryType(b.typeS()),
		b.factory.ReturnStatement(b.factory.ArrayLiteralExpression(
			[]tsgo.Expression{
				b.property(pointer, AddressName),
				read,
				write,
				b.property(pointer, RegionName),
			},
			false,
		)),
	)
}

func (b builder) unsafeViewMethod() tsgo.MethodDeclaration {
	typeL := b.typeReference("L")
	typeS := b.typeReference("S")
	readType := b.factory.FunctionTypeNode(nil, nil, typeS)
	writeType := b.factory.FunctionTypeNode(
		nil,
		[]tsgo.ParameterDeclaration{b.parameter("value", typeS)},
		b.factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindVoidKeyword),
	)
	return b.method(
		[]tsgo.ModifierLike{b.factory.StaticKeyword()},
		UnsafeViewName,
		[]tsgo.TypeParameterDeclaration{
			b.typeParameter("L", nil),
			b.typeParameter("S", nil),
		},
		[]tsgo.ParameterDeclaration{
			b.parameter("address", b.objectType()),
			b.parameter("read", readType),
			b.parameter("write", writeType),
			b.parameter("region", b.optionalRegionType(typeS)),
		},
		b.pointerType(typeL, typeS),
		b.factory.ReturnStatement(b.factory.NewExpression(
			b.id(b.className),
			[]tsgo.TypeNode{typeL, typeS},
			[]tsgo.Expression{
				b.factory.ArrowFunction(
					nil,
					nil,
					nil,
					nil,
					b.factory.EqualsGreaterThanToken(),
					b.id("address"),
				),
				b.id("read"),
				b.id("write"),
				b.id("region"),
			},
		)),
	)
}

func UnsafeMemory(
	factory tsgo.Factory,
	functionName string,
	pointerName string,
) tsgo.FunctionDeclaration {
	b := builder{factory: factory, className: pointerName}
	typeL := b.typeReference("L")
	typeS := b.typeReference("S")
	pointerType := b.pointerType(typeL, typeS)
	resultType := b.unsafeMemoryType(typeS)
	return factory.FunctionDeclaration(
		[]tsgo.ModifierLike{factory.ExportKeyword()},
		nil,
		factory.Identifier(functionName),
		[]tsgo.TypeParameterDeclaration{
			b.typeParameter("L", nil),
			b.typeParameter("S", nil),
		},
		[]tsgo.ParameterDeclaration{b.parameter("pointer", pointerType)},
		resultType,
		factory.Block([]tsgo.Statement{factory.ReturnStatement(
			factory.CallExpression(
				factory.PropertyAccessExpression(
					factory.Identifier(pointerName),
					nil,
					factory.Identifier(UnsafeMemoryName),
					tsgo.NodeFlagsNone,
				),
				nil,
				[]tsgo.TypeNode{typeL, typeS},
				[]tsgo.Expression{factory.Identifier("pointer")},
				tsgo.NodeFlagsNone,
			),
		)}, true),
	)
}
