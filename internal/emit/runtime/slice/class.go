package slice

import (
	panicruntime "github.com/tsoniclang/gotots/internal/emit/runtime/panic"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type builder struct {
	factory   tsgo.Factory
	className string
	panicName string
}

type Capabilities struct {
	Address      bool
	Storage      bool
	AppendSlice  bool
	Clear        bool
	ArrayPointer bool
}

func Build(
	factory tsgo.Factory,
	className string,
	panicName string,
) tsgo.ClassDeclaration {
	return BuildWithCapabilities(
		factory,
		className,
		panicName,
		Capabilities{},
	)
}

func BuildWithAddress(
	factory tsgo.Factory,
	className string,
	panicName string,
	withAddress bool,
) tsgo.ClassDeclaration {
	return BuildWithCapabilities(
		factory,
		className,
		panicName,
		Capabilities{Address: withAddress},
	)
}

func BuildWithCapabilities(
	factory tsgo.Factory,
	className string,
	panicName string,
	capabilities Capabilities,
) tsgo.ClassDeclaration {
	target := builder{
		factory:   factory,
		className: className,
		panicName: panicName,
	}
	members := []tsgo.ClassElement{target.constructor()}
	members = append(
		members,
		target.nilMethod(),
		target.makeMethod(),
		target.literalMethod(),
		target.isNilMethod(),
		target.getMethod(),
		target.setMethod(),
		target.sliceMethod(),
		target.appendMethod(),
		target.copyMethod(),
	)
	if capabilities.Storage {
		members = append(members, target.storageMethods()...)
	}
	if capabilities.AppendSlice {
		members = append(members, target.appendSliceMethod())
	}
	if capabilities.Clear {
		members = append(members, target.clearMethod())
	}
	if capabilities.Address {
		members = append(members, target.addressMethod())
	}
	if capabilities.ArrayPointer {
		members = append(members, target.arrayLocationMethod())
	}
	return factory.ClassDeclaration(
		[]tsgo.ModifierLike{factory.ExportKeyword()},
		factory.Identifier(className),
		[]tsgo.TypeParameterDeclaration{target.typeParameter()},
		nil,
		members,
	)
}

func (b builder) typeParameter() tsgo.TypeParameterDeclaration {
	return b.factory.TypeParameterDeclaration(
		nil,
		b.id("T"),
		nil,
		nil,
		nil,
	)
}

func (b builder) typeT() tsgo.TypeNode {
	return b.factory.TypeReferenceNode(b.id("T"), nil)
}

func (b builder) sliceType() tsgo.TypeNode {
	return b.factory.TypeReferenceNode(
		b.id(b.className),
		[]tsgo.TypeNode{b.typeT()},
	)
}

func (b builder) numberType() tsgo.TypeNode {
	return b.factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindNumberKeyword)
}

func (b builder) booleanType() tsgo.TypeNode {
	return b.factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindBooleanKeyword)
}

func (b builder) bigIntType() tsgo.TypeNode {
	return b.factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindBigIntKeyword)
}

func (b builder) integerInputType() tsgo.TypeNode {
	return b.factory.UnionTypeNode([]tsgo.TypeNode{
		b.numberType(),
		b.bigIntType(),
	})
}

func (b builder) optionalIntegerInputType() tsgo.TypeNode {
	return b.factory.UnionTypeNode([]tsgo.TypeNode{
		b.integerInputType(),
		b.factory.LiteralTypeNode(b.factory.NullLiteral()),
	})
}

func (b builder) backingType() tsgo.TypeNode {
	return b.factory.UnionTypeNode([]tsgo.TypeNode{
		b.factory.ArrayTypeNode(b.typeT()),
		b.factory.LiteralTypeNode(b.factory.NullLiteral()),
	})
}

func (b builder) id(name string) tsgo.Identifier {
	return b.factory.Identifier(name)
}

func (b builder) number(value string) tsgo.NumericLiteral {
	return b.factory.NumericLiteral(value, tsgo.TokenFlagsNone)
}

func (b builder) parameter(
	name string,
	typeNode tsgo.TypeNode,
) tsgo.ParameterDeclaration {
	return b.factory.ParameterDeclaration(
		nil,
		nil,
		b.id(name),
		nil,
		typeNode,
		nil,
	)
}

func (b builder) property(
	receiver tsgo.Expression,
	name string,
) tsgo.PropertyAccessExpression {
	return b.factory.PropertyAccessExpression(
		receiver,
		nil,
		b.id(name),
		tsgo.NodeFlagsNone,
	)
}

func (b builder) thisProperty(name string) tsgo.PropertyAccessExpression {
	return b.property(b.factory.ThisExpression(), name)
}

func (b builder) call(
	receiver tsgo.Expression,
	name string,
	arguments ...tsgo.Expression,
) tsgo.CallExpression {
	return b.factory.CallExpression(
		b.property(receiver, name),
		nil,
		nil,
		arguments,
		tsgo.NodeFlagsNone,
	)
}

func (b builder) globalCall(
	name string,
	argument tsgo.Expression,
) tsgo.CallExpression {
	return b.factory.CallExpression(
		b.id(name),
		nil,
		nil,
		[]tsgo.Expression{argument},
		tsgo.NodeFlagsNone,
	)
}

func (b builder) toNumber(value tsgo.Expression) tsgo.CallExpression {
	return b.globalCall("Number", value)
}

func (b builder) binary(
	left tsgo.Expression,
	operator tsgo.BinaryOperator,
	right tsgo.Expression,
) tsgo.BinaryExpression {
	return b.factory.BinaryExpression(
		nil,
		left,
		nil,
		b.factory.BinaryOperatorToken(operator),
		right,
	)
}

func (b builder) assign(
	left tsgo.Expression,
	right tsgo.Expression,
) tsgo.BinaryExpression {
	return b.binary(left, tsgo.BinaryOperatorEqualsToken, right)
}

func (b builder) add(
	left tsgo.Expression,
	right tsgo.Expression,
) tsgo.BinaryExpression {
	return b.binary(left, tsgo.BinaryOperatorPlusToken, right)
}

func (b builder) subtract(
	left tsgo.Expression,
	right tsgo.Expression,
) tsgo.BinaryExpression {
	return b.binary(left, tsgo.BinaryOperatorMinusToken, right)
}

func (b builder) variable(
	flags tsgo.NodeFlags,
	name string,
	value tsgo.Expression,
) tsgo.VariableStatement {
	return b.factory.VariableStatement(
		nil,
		b.factory.VariableDeclarationList(
			[]tsgo.VariableDeclaration{
				b.factory.VariableDeclaration(b.id(name), nil, nil, value),
			},
			flags,
		),
	)
}

func (b builder) returnStatement(value tsgo.Expression) tsgo.ReturnStatement {
	return b.factory.ReturnStatement(value)
}

func (b builder) newSlice(
	backing tsgo.Expression,
	offset tsgo.Expression,
	length tsgo.Expression,
	capacity tsgo.Expression,
) tsgo.NewExpression {
	return b.factory.NewExpression(
		b.id(b.className),
		[]tsgo.TypeNode{b.typeT()},
		[]tsgo.Expression{backing, offset, length, capacity},
	)
}

func (b builder) throwBounds() tsgo.ExpressionStatement {
	return b.factory.ExpressionStatement(
		panicruntime.Call(
			b.factory,
			b.panicName,
			b.factory.StringLiteral(
				"slice bounds out of range",
				tsgo.TokenFlagsNone,
			),
		),
	)
}

func (b builder) method(
	modifiers []tsgo.ModifierLike,
	name string,
	typeParameters []tsgo.TypeParameterDeclaration,
	parameters []tsgo.ParameterDeclaration,
	result tsgo.TypeNode,
	statements ...tsgo.Statement,
) tsgo.MethodDeclaration {
	return b.factory.MethodDeclaration(
		modifiers,
		nil,
		b.id(name),
		nil,
		typeParameters,
		parameters,
		result,
		b.factory.Block(statements, true),
	)
}

func (b builder) index(
	value tsgo.Expression,
	index tsgo.Expression,
) tsgo.ElementAccessExpression {
	return b.factory.ElementAccessExpression(
		value,
		nil,
		index,
		tsgo.NodeFlagsNone,
	)
}

func (b builder) loop(
	limit tsgo.Expression,
	body ...tsgo.Statement,
) tsgo.ForStatement {
	return b.factory.ForStatement(
		b.factory.VariableDeclarationList(
			[]tsgo.VariableDeclaration{
				b.factory.VariableDeclaration(
					b.id("index"),
					nil,
					nil,
					b.number("0"),
				),
			},
			tsgo.NodeFlagsLet,
		),
		b.binary(
			b.id("index"),
			tsgo.BinaryOperatorLessThanToken,
			limit,
		),
		b.assign(
			b.id("index"),
			b.add(b.id("index"), b.number("1")),
		),
		b.factory.Block(body, true),
	)
}
