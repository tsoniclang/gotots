package pointer

import (
	mapruntime "github.com/tsoniclang/gotots/internal/emit/runtime/map"
	panicruntime "github.com/tsoniclang/gotots/internal/emit/runtime/panic"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func (b builder) equalMethod() tsgo.MethodDeclaration {
	leftType := b.factory.UnionTypeNode(
		[]tsgo.TypeNode{
			b.pointerType(
				b.typeReference("LL"),
				b.typeReference("LS"),
			),
			b.undefinedType(),
		},
	)
	rightType := b.factory.UnionTypeNode(
		[]tsgo.TypeNode{
			b.pointerType(
				b.typeReference("RL"),
				b.typeReference("RS"),
			),
			b.undefinedType(),
		},
	)
	left := b.id("left")
	right := b.id("right")
	bothDefined := b.binary(
		b.binary(
			left,
			tsgo.BinaryOperatorExclamationEqualsEqualsToken,
			b.undefined(),
		),
		tsgo.BinaryOperatorAmpersandAmpersandToken,
		b.binary(
			right,
			tsgo.BinaryOperatorExclamationEqualsEqualsToken,
			b.undefined(),
		),
	)
	sameAddress := b.binary(
		b.property(left, AddressName),
		tsgo.BinaryOperatorEqualsEqualsEqualsToken,
		b.property(right, AddressName),
	)
	return b.method(
		[]tsgo.ModifierLike{b.factory.StaticKeyword()},
		EqualName,
		[]tsgo.TypeParameterDeclaration{
			b.typeParameter("LL", nil),
			b.typeParameter("LS", nil),
			b.typeParameter("RL", nil),
			b.typeParameter("RS", nil),
		},
		[]tsgo.ParameterDeclaration{
			b.parameter("left", leftType),
			b.parameter("right", rightType),
		},
		b.booleanType(),
		b.factory.ReturnStatement(
			b.binary(
				b.binary(
					left,
					tsgo.BinaryOperatorEqualsEqualsEqualsToken,
					right,
				),
				tsgo.BinaryOperatorBarBarToken,
				b.binary(
					bothDefined,
					tsgo.BinaryOperatorAmpersandAmpersandToken,
					sameAddress,
				),
			),
		),
	)
}

func Hash(
	factory tsgo.Factory,
	functionName string,
	pointerName string,
	mapHashName string,
) tsgo.FunctionDeclaration {
	b := builder{factory: factory, className: pointerName}
	pointerType := b.pointerType(b.typeL(), b.typeS())
	pointer := b.id("pointer")
	return factory.FunctionDeclaration(
		[]tsgo.ModifierLike{factory.ExportKeyword()},
		nil,
		factory.Identifier(functionName),
		[]tsgo.TypeParameterDeclaration{
			b.typeParameter("L", nil),
			b.typeParameter("S", nil),
		},
		[]tsgo.ParameterDeclaration{
			b.parameter(
				"pointer",
				b.factory.UnionTypeNode(
					[]tsgo.TypeNode{pointerType, b.undefinedType()},
				),
			),
		},
		b.numberType(),
		factory.Block(
			[]tsgo.Statement{
				factory.ReturnStatement(
					factory.ConditionalExpression(
						b.binary(
							pointer,
							tsgo.BinaryOperatorEqualsEqualsEqualsToken,
							b.undefined(),
						),
						factory.QuestionToken(),
						factory.NumericLiteral("0", tsgo.TokenFlagsNone),
						factory.ColonToken(),
						b.call(
							b.id(mapHashName),
							mapruntime.HashObjectMember,
							b.property(pointer, AddressName),
						),
					),
				),
			},
			true,
		),
	)
}

func (b builder) dereferenceMethod() tsgo.MethodDeclaration {
	pointerType := b.pointerType(b.typeL(), b.typeS())
	pointer := b.id("pointer")
	return b.method(
		[]tsgo.ModifierLike{b.factory.StaticKeyword()},
		DereferenceName,
		[]tsgo.TypeParameterDeclaration{
			b.typeParameter("L", nil),
			b.typeParameter("S", nil),
		},
		[]tsgo.ParameterDeclaration{
			b.parameter(
				"pointer",
				b.factory.UnionTypeNode(
					[]tsgo.TypeNode{pointerType, b.undefinedType()},
				),
			),
		},
		pointerType,
		b.factory.IfStatement(
			b.binary(
				pointer,
				tsgo.BinaryOperatorEqualsEqualsEqualsToken,
				b.undefined(),
			),
			b.factory.Block(
				[]tsgo.Statement{
					b.factory.ExpressionStatement(
						panicruntime.Call(
							b.factory,
							b.panicName,
							b.factory.StringLiteral(
								"nil pointer dereference",
								tsgo.TokenFlagsNone,
							),
						),
					),
				},
				true,
			),
			nil,
		),
		b.factory.ReturnStatement(pointer),
	)
}

func (b builder) valueGetter() tsgo.GetAccessorDeclaration {
	return b.factory.GetAccessorDeclaration(
		nil,
		b.id(CellValueName),
		nil,
		nil,
		b.typeS(),
		b.factory.Block(
			[]tsgo.Statement{
				b.factory.ReturnStatement(
					b.factory.CallExpression(
						b.property(
							b.factory.ThisExpression(),
							"read",
						),
						nil,
						nil,
						nil,
						tsgo.NodeFlagsNone,
					),
				),
			},
			true,
		),
	)
}

func (b builder) valueSetter() tsgo.SetAccessorDeclaration {
	return b.factory.SetAccessorDeclaration(
		nil,
		b.id(CellValueName),
		nil,
		[]tsgo.ParameterDeclaration{
			b.parameter("value", b.typeS()),
		},
		nil,
		b.factory.Block(
			[]tsgo.Statement{
				b.factory.ExpressionStatement(
					b.factory.CallExpression(
						b.property(
							b.factory.ThisExpression(),
							"write",
						),
						nil,
						nil,
						[]tsgo.Expression{b.id("value")},
						tsgo.NodeFlagsNone,
					),
				),
			},
			true,
		),
	)
}

func (b builder) newPointer(
	logicalType tsgo.TypeNode,
	storageType tsgo.TypeNode,
	address tsgo.Expression,
	read tsgo.Expression,
	write tsgo.Expression,
) tsgo.NewExpression {
	return b.newPointerWithWrite(
		logicalType,
		storageType,
		address,
		read,
		b.assign(write, b.id("next")),
	)
}

func (b builder) newPointerWithWrite(
	logicalType tsgo.TypeNode,
	storageType tsgo.TypeNode,
	address tsgo.Expression,
	read tsgo.Expression,
	write tsgo.Expression,
) tsgo.NewExpression {
	return b.newPointerWithWriteBody(
		logicalType,
		storageType,
		address,
		read,
		[]tsgo.Statement{b.factory.ExpressionStatement(write)},
	)
}

func (b builder) newPointerWithWriteBody(
	logicalType tsgo.TypeNode,
	storageType tsgo.TypeNode,
	address tsgo.Expression,
	read tsgo.Expression,
	write []tsgo.Statement,
) tsgo.NewExpression {
	readArrow := b.factory.ArrowFunction(
		nil,
		nil,
		nil,
		nil,
		b.factory.EqualsGreaterThanToken(),
		b.factory.Block(
			[]tsgo.Statement{b.factory.ReturnStatement(read)},
			true,
		),
	)
	writeArrow := b.factory.ArrowFunction(
		nil,
		nil,
		[]tsgo.ParameterDeclaration{
			b.parameter("next", storageType),
		},
		nil,
		b.factory.EqualsGreaterThanToken(),
		b.factory.Block(write, true),
	)
	return b.factory.NewExpression(
		b.id(b.className),
		[]tsgo.TypeNode{logicalType, storageType},
		[]tsgo.Expression{address, readArrow, writeArrow},
	)
}

func (b builder) viewMethod() tsgo.MethodDeclaration {
	from := b.typeReference("F")
	to := b.typeReference("T")
	storage := b.typeReference("S")
	pointer := b.id("pointer")
	sourceType := b.factory.UnionTypeNode([]tsgo.TypeNode{
		b.pointerType(from, storage),
		b.undefinedType(),
	})
	targetType := b.factory.UnionTypeNode([]tsgo.TypeNode{
		b.pointerType(to, storage),
		b.undefinedType(),
	})
	return b.method(
		[]tsgo.ModifierLike{b.factory.StaticKeyword()},
		ViewName,
		[]tsgo.TypeParameterDeclaration{
			b.typeParameter("F", nil),
			b.typeParameter("T", nil),
			b.typeParameter("S", nil),
		},
		[]tsgo.ParameterDeclaration{b.parameter("pointer", sourceType)},
		targetType,
		b.factory.IfStatement(
			b.binary(
				pointer,
				tsgo.BinaryOperatorEqualsEqualsEqualsToken,
				b.undefined(),
			),
			b.factory.Block(
				[]tsgo.Statement{b.factory.ReturnStatement(b.undefined())},
				true,
			),
			nil,
		),
		b.factory.ReturnStatement(
			b.factory.NewExpression(
				b.id(b.className),
				[]tsgo.TypeNode{to, storage},
				[]tsgo.Expression{
					b.property(pointer, AddressName),
					b.property(pointer, "read"),
					b.property(pointer, "write"),
				},
			),
		),
	)
}
