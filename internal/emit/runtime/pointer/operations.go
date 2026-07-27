package pointer

import (
	panicruntime "github.com/tsoniclang/gotots/internal/emit/runtime/panic"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func (b builder) equalMethod() tsgo.MethodDeclaration {
	pointerType := b.factory.UnionTypeNode(
		[]tsgo.TypeNode{
			b.pointerType(b.typeT()),
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
		b.property(left, "address"),
		tsgo.BinaryOperatorEqualsEqualsEqualsToken,
		b.property(right, "address"),
	)
	return b.method(
		[]tsgo.ModifierLike{b.factory.StaticKeyword()},
		EqualName,
		[]tsgo.TypeParameterDeclaration{b.typeParameter("T", nil)},
		[]tsgo.ParameterDeclaration{
			b.parameter("left", pointerType),
			b.parameter("right", pointerType),
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

func (b builder) dereferenceMethod() tsgo.MethodDeclaration {
	pointerType := b.pointerType(b.typeT())
	pointer := b.id("pointer")
	return b.method(
		[]tsgo.ModifierLike{b.factory.StaticKeyword()},
		DereferenceName,
		[]tsgo.TypeParameterDeclaration{b.typeParameter("T", nil)},
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
		b.typeT(),
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
			b.parameter("value", b.typeT()),
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
	targetType tsgo.TypeNode,
	address tsgo.Expression,
	read tsgo.Expression,
	write tsgo.Expression,
) tsgo.NewExpression {
	return b.newPointerWithWrite(
		targetType,
		address,
		read,
		b.assign(write, b.id("next")),
	)
}

func (b builder) newPointerWithWrite(
	targetType tsgo.TypeNode,
	address tsgo.Expression,
	read tsgo.Expression,
	write tsgo.Expression,
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
			b.parameter("next", targetType),
		},
		nil,
		b.factory.EqualsGreaterThanToken(),
		b.factory.Block(
			[]tsgo.Statement{
				b.factory.ExpressionStatement(write),
			},
			true,
		),
	)
	return b.factory.NewExpression(
		b.id(b.className),
		[]tsgo.TypeNode{targetType},
		[]tsgo.Expression{address, readArrow, writeArrow},
	)
}
