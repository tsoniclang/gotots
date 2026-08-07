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
	sameAddress := b.binary(
		b.factory.PropertyAccessExpression(
			left,
			b.factory.QuestionDotToken(),
			b.id(AddressName),
			tsgo.NodeFlagsOptionalChain,
		),
		tsgo.BinaryOperatorEqualsEqualsEqualsToken,
		b.factory.PropertyAccessExpression(
			right,
			b.factory.QuestionDotToken(),
			b.id(AddressName),
			tsgo.NodeFlagsOptionalChain,
		),
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
				b.binary(left, tsgo.BinaryOperatorEqualsEqualsEqualsToken, right),
				tsgo.BinaryOperatorBarBarToken,
				sameAddress,
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

func (b builder) directMethod() tsgo.MethodDeclaration {
	pointerType := b.factory.UnionTypeNode(
		[]tsgo.TypeNode{b.typeL(), b.undefinedType()},
	)
	pointer := b.id("pointer")
	return b.method(
		[]tsgo.ModifierLike{b.factory.StaticKeyword()},
		DirectName,
		[]tsgo.TypeParameterDeclaration{b.typeParameter("L", nil)},
		[]tsgo.ParameterDeclaration{b.parameter("pointer", pointerType)},
		b.typeL(),
		b.factory.IfStatement(
			b.binary(
				pointer,
				tsgo.BinaryOperatorEqualsEqualsEqualsToken,
				b.undefined(),
			),
			b.factory.Block(
				[]tsgo.Statement{b.factory.ExpressionStatement(
					panicruntime.Call(
						b.factory,
						b.panicName,
						b.factory.StringLiteral(
							"nil pointer dereference",
							tsgo.TokenFlagsNone,
						),
					),
				)},
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
		b.factory.Block([]tsgo.Statement{b.factory.ReturnStatement(
			b.factory.CallExpression(
				b.property(b.factory.ThisExpression(), "read"),
				nil,
				nil,
				nil,
				tsgo.NodeFlagsNone,
			),
		)}, true),
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
		b.factory.Block([]tsgo.Statement{b.factory.ExpressionStatement(
			b.factory.CallExpression(
				b.property(b.factory.ThisExpression(), "write"),
				nil,
				nil,
				[]tsgo.Expression{b.id("value")},
				tsgo.NodeFlagsNone,
			),
		)}, true),
	)
}

func (b builder) newPointer(
	logicalType tsgo.TypeNode,
	storageType tsgo.TypeNode,
	address tsgo.ConciseBody,
	read tsgo.ConciseBody,
	write tsgo.Expression,
) tsgo.NewExpression {
	return b.newPointerWithRegion(
		logicalType,
		storageType,
		address,
		read,
		write,
		nil,
	)
}

func (b builder) newPointerWithRegion(
	logicalType tsgo.TypeNode,
	storageType tsgo.TypeNode,
	address tsgo.ConciseBody,
	read tsgo.ConciseBody,
	write tsgo.Expression,
	region tsgo.Expression,
) tsgo.NewExpression {
	return b.newPointerWithAccessBodies(
		logicalType,
		storageType,
		address,
		read,
		b.assign(write, b.id("next")),
		region,
	)
}

func (b builder) newPointerWithWrite(
	logicalType tsgo.TypeNode,
	storageType tsgo.TypeNode,
	address tsgo.ConciseBody,
	read tsgo.ConciseBody,
	write tsgo.ConciseBody,
) tsgo.NewExpression {
	return b.newPointerWithAccessBodies(
		logicalType,
		storageType,
		address,
		read,
		write,
		nil,
	)
}

func (b builder) newPointerWithWriteBody(
	logicalType tsgo.TypeNode,
	storageType tsgo.TypeNode,
	address tsgo.ConciseBody,
	read tsgo.ConciseBody,
	write []tsgo.Statement,
) tsgo.NewExpression {
	return b.newPointerWithWriteBodyAndRegion(
		logicalType,
		storageType,
		address,
		read,
		write,
		nil,
	)
}

func (b builder) newPointerWithWriteBodyAndRegion(
	logicalType tsgo.TypeNode,
	storageType tsgo.TypeNode,
	address tsgo.ConciseBody,
	read tsgo.ConciseBody,
	write []tsgo.Statement,
	region tsgo.Expression,
) tsgo.NewExpression {
	return b.newPointerWithAccessBodies(
		logicalType,
		storageType,
		address,
		read,
		b.factory.Block(write, true),
		region,
	)
}

func (b builder) newPointerWithAccessBodies(
	logicalType tsgo.TypeNode,
	storageType tsgo.TypeNode,
	address tsgo.ConciseBody,
	read tsgo.ConciseBody,
	write tsgo.ConciseBody,
	region tsgo.Expression,
) tsgo.NewExpression {
	addressArrow := b.factory.ArrowFunction(
		nil,
		nil,
		nil,
		nil,
		b.factory.EqualsGreaterThanToken(),
		address,
	)
	readArrow := b.factory.ArrowFunction(
		nil,
		nil,
		nil,
		nil,
		b.factory.EqualsGreaterThanToken(),
		read,
	)
	writeArrow := b.factory.ArrowFunction(
		nil,
		nil,
		[]tsgo.ParameterDeclaration{
			b.parameter("next", storageType),
		},
		nil,
		b.factory.EqualsGreaterThanToken(),
		write,
	)
	arguments := []tsgo.Expression{addressArrow, readArrow, writeArrow}
	if b.capabilities.Region && region != nil {
		arguments = append(arguments, region)
	}
	return b.factory.NewExpression(
		b.id(b.className),
		[]tsgo.TypeNode{logicalType, storageType},
		arguments,
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
		b.factory.ReturnStatement(func() tsgo.Expression {
			address := b.factory.ArrowFunction(
				nil,
				nil,
				nil,
				nil,
				b.factory.EqualsGreaterThanToken(),
				b.property(pointer, AddressName),
			)
			arguments := []tsgo.Expression{
				address,
				b.property(pointer, "read"),
				b.property(pointer, "write"),
			}
			if b.capabilities.Region {
				arguments = append(arguments, b.property(pointer, RegionName))
			}
			return b.factory.NewExpression(
				b.id(b.className),
				[]tsgo.TypeNode{to, storage},
				arguments,
			)
		}()),
	)
}
