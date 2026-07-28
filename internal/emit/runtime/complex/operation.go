package complex

import (
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func buildOperation(
	factory tsgo.Factory,
	symbol api.RuntimeSymbol,
	name string,
	className string,
	divideName string,
) (tsgo.FunctionDeclaration, bool) {
	target := builder{factory: factory}
	switch symbol {
	case api.RuntimeComplex64Add, api.RuntimeComplex128Add:
		return target.binaryOperation(
			name,
			className,
			tsgo.BinaryOperatorPlusToken,
		), true
	case api.RuntimeComplex64Sub, api.RuntimeComplex128Sub:
		return target.binaryOperation(
			name,
			className,
			tsgo.BinaryOperatorMinusToken,
		), true
	case api.RuntimeComplex64Mul, api.RuntimeComplex128Mul:
		return target.multiplyOperation(name, className), true
	case api.RuntimeComplex64Div, api.RuntimeComplex128Div:
		return target.divideOperation(name, className, divideName), true
	case api.RuntimeComplex64Neg, api.RuntimeComplex128Neg:
		return target.negateOperation(name, className), true
	case api.RuntimeComplex64Equal, api.RuntimeComplex128Equal:
		return target.equalOperation(name, className), true
	default:
		return nil, false
	}
}

func (b builder) binaryOperation(
	name string,
	className string,
	operator tsgo.BinaryOperator,
) tsgo.FunctionDeclaration {
	left := b.id("left")
	right := b.id("right")
	return b.operation(
		name,
		className,
		b.call(
			b.property(b.id(className), MakeMember),
			b.binary(b.property(left, RealMember), operator, b.property(right, RealMember)),
			b.binary(b.property(left, ImagMember), operator, b.property(right, ImagMember)),
		),
	)
}

func (b builder) multiplyOperation(
	name string,
	className string,
) tsgo.FunctionDeclaration {
	left := b.id("left")
	right := b.id("right")
	leftReal := b.property(left, RealMember)
	leftImag := b.property(left, ImagMember)
	rightReal := b.property(right, RealMember)
	rightImag := b.property(right, ImagMember)
	realPart := b.binary(
		b.binary(leftReal, tsgo.BinaryOperatorAsteriskToken, rightReal),
		tsgo.BinaryOperatorMinusToken,
		b.binary(leftImag, tsgo.BinaryOperatorAsteriskToken, rightImag),
	)
	imaginaryPart := b.binary(
		b.binary(leftReal, tsgo.BinaryOperatorAsteriskToken, rightImag),
		tsgo.BinaryOperatorPlusToken,
		b.binary(leftImag, tsgo.BinaryOperatorAsteriskToken, rightReal),
	)
	return b.operation(
		name,
		className,
		b.call(
			b.property(b.id(className), MakeMember),
			realPart,
			imaginaryPart,
		),
	)
}

func (b builder) divideOperation(
	name string,
	className string,
	divideName string,
) tsgo.FunctionDeclaration {
	left := b.id("left")
	right := b.id("right")
	result := b.id("result")
	body := []tsgo.Statement{
		b.variable(
			tsgo.NodeFlagsConst,
			"result",
			nil,
			b.call(
				b.id(divideName),
				b.property(left, RealMember),
				b.property(left, ImagMember),
				b.property(right, RealMember),
				b.property(right, ImagMember),
			),
		),
		b.factory.ReturnStatement(b.call(
			b.property(b.id(className), MakeMember),
			b.factory.ElementAccessExpression(
				result,
				nil,
				b.zero(),
				tsgo.NodeFlagsNone,
			),
			b.factory.ElementAccessExpression(
				result,
				nil,
				b.one(),
				tsgo.NodeFlagsNone,
			),
		)),
	}
	return b.function(name, className, body)
}

func (b builder) negateOperation(
	name string,
	className string,
) tsgo.FunctionDeclaration {
	value := b.id("value")
	return b.unaryOperation(
		name,
		className,
		b.call(
			b.property(b.id(className), MakeMember),
			b.prefix(
				tsgo.PrefixUnaryExpressionOperatorKindMinusToken,
				b.property(value, RealMember),
			),
			b.prefix(
				tsgo.PrefixUnaryExpressionOperatorKindMinusToken,
				b.property(value, ImagMember),
			),
		),
	)
}

func (b builder) equalOperation(
	name string,
	className string,
) tsgo.FunctionDeclaration {
	left := b.id("left")
	right := b.id("right")
	return b.factory.FunctionDeclaration(
		[]tsgo.ModifierLike{b.factory.ExportKeyword()},
		nil,
		b.id(name),
		nil,
		[]tsgo.ParameterDeclaration{
			b.parameter("left", b.typeReference(className)),
			b.parameter("right", b.typeReference(className)),
		},
		b.booleanType(),
		b.factory.Block([]tsgo.Statement{
			b.factory.ReturnStatement(b.binary(
				b.binary(
					b.property(left, RealMember),
					tsgo.BinaryOperatorEqualsEqualsEqualsToken,
					b.property(right, RealMember),
				),
				tsgo.BinaryOperatorAmpersandAmpersandToken,
				b.binary(
					b.property(left, ImagMember),
					tsgo.BinaryOperatorEqualsEqualsEqualsToken,
					b.property(right, ImagMember),
				),
			)),
		}, true),
	)
}

func (b builder) operation(
	name string,
	className string,
	result tsgo.Expression,
) tsgo.FunctionDeclaration {
	return b.function(
		name,
		className,
		[]tsgo.Statement{b.factory.ReturnStatement(result)},
	)
}

func (b builder) unaryOperation(
	name string,
	className string,
	result tsgo.Expression,
) tsgo.FunctionDeclaration {
	return b.factory.FunctionDeclaration(
		[]tsgo.ModifierLike{b.factory.ExportKeyword()},
		nil,
		b.id(name),
		nil,
		[]tsgo.ParameterDeclaration{
			b.parameter("value", b.typeReference(className)),
		},
		b.typeReference(className),
		b.factory.Block(
			[]tsgo.Statement{b.factory.ReturnStatement(result)},
			true,
		),
	)
}

func (b builder) function(
	name string,
	className string,
	body []tsgo.Statement,
) tsgo.FunctionDeclaration {
	return b.factory.FunctionDeclaration(
		[]tsgo.ModifierLike{b.factory.ExportKeyword()},
		nil,
		b.id(name),
		nil,
		[]tsgo.ParameterDeclaration{
			b.parameter("left", b.typeReference(className)),
			b.parameter("right", b.typeReference(className)),
		},
		b.typeReference(className),
		b.factory.Block(body, true),
	)
}
