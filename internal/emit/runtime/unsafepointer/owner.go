package unsafepointer

import (
	panicruntime "github.com/tsoniclang/gotots/internal/emit/runtime/panic"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

const (
	brandName             = "$go$unsafePointer"
	FromName              = "from"
	ToName                = "to"
	FromIntegerName       = "fromInteger"
	ToIntegerName         = "toInteger"
	unresolvedPlaceholder = "unsafe.Pointer conversion requires an environment implementation"
)

func Build(
	factory tsgo.Factory,
	className string,
	panicName string,
) tsgo.ClassDeclaration {
	target := builder{
		factory:   factory,
		className: className,
		panicName: panicName,
	}
	return factory.ClassDeclaration(
		[]tsgo.ModifierLike{factory.ExportKeyword()},
		target.id(className),
		nil,
		nil,
		[]tsgo.ClassElement{
			target.brand(),
			target.constructor(),
			target.from(),
			target.to(),
			target.fromInteger(),
			target.toInteger(),
		},
	)
}

type builder struct {
	factory   tsgo.Factory
	className string
	panicName string
}

func (b builder) brand() tsgo.PropertyDeclaration {
	return b.factory.PropertyDeclaration(
		[]tsgo.ModifierLike{
			b.factory.DeclareKeyword(),
			b.factory.PrivateKeyword(),
			b.factory.ReadonlyKeyword(),
		},
		b.id(brandName),
		nil,
		b.factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindVoidKeyword),
		nil,
	)
}

func (b builder) constructor() tsgo.ConstructorDeclaration {
	return b.factory.ConstructorDeclaration(
		[]tsgo.ModifierLike{b.factory.PrivateKeyword()},
		nil,
		nil,
		nil,
		b.factory.Block(
			[]tsgo.Statement{
				b.factory.ExpressionStatement(b.unresolved()),
			},
			true,
		),
	)
}

func (b builder) from() tsgo.MethodDeclaration {
	return b.method(
		FromName,
		b.nullable(b.genericType()),
		b.nullable(b.unsafeType()),
	)
}

func (b builder) to() tsgo.MethodDeclaration {
	return b.method(
		ToName,
		b.nullable(b.unsafeType()),
		b.nullable(b.genericType()),
	)
}

func (b builder) fromInteger() tsgo.MethodDeclaration {
	value := b.id("value")
	zero := b.id("zero")
	return b.factory.MethodDeclaration(
		[]tsgo.ModifierLike{b.factory.StaticKeyword()},
		nil,
		b.id(FromIntegerName),
		nil,
		[]tsgo.TypeParameterDeclaration{b.integerTypeParameter("I")},
		[]tsgo.ParameterDeclaration{
			b.parameter("value", b.typeReference("I")),
			b.parameter("zero", b.typeReference("I")),
		},
		b.nullable(b.unsafeType()),
		b.factory.Block(
			[]tsgo.Statement{
				b.factory.IfStatement(
					b.equals(value, zero),
					b.factory.Block(
						[]tsgo.Statement{
							b.factory.ReturnStatement(b.undefined()),
						},
						true,
					),
					nil,
				),
				b.factory.ExpressionStatement(b.unresolved()),
			},
			true,
		),
	)
}

func (b builder) toInteger() tsgo.MethodDeclaration {
	value := b.id("value")
	zero := b.id("zero")
	return b.factory.MethodDeclaration(
		[]tsgo.ModifierLike{b.factory.StaticKeyword()},
		nil,
		b.id(ToIntegerName),
		nil,
		[]tsgo.TypeParameterDeclaration{b.integerTypeParameter("I")},
		[]tsgo.ParameterDeclaration{
			b.parameter("value", b.nullable(b.unsafeType())),
			b.parameter("zero", b.typeReference("I")),
		},
		b.typeReference("I"),
		b.factory.Block(
			[]tsgo.Statement{
				b.factory.IfStatement(
					b.equals(value, b.undefined()),
					b.factory.Block(
						[]tsgo.Statement{
							b.factory.ReturnStatement(zero),
						},
						true,
					),
					nil,
				),
				b.factory.ExpressionStatement(b.unresolved()),
			},
			true,
		),
	)
}

func (b builder) method(
	name string,
	parameterType tsgo.TypeNode,
	resultType tsgo.TypeNode,
) tsgo.MethodDeclaration {
	value := b.id("value")
	return b.factory.MethodDeclaration(
		[]tsgo.ModifierLike{b.factory.StaticKeyword()},
		nil,
		b.id(name),
		nil,
		[]tsgo.TypeParameterDeclaration{
			b.typeParameter("P"),
		},
		[]tsgo.ParameterDeclaration{
			b.factory.ParameterDeclaration(
				nil,
				nil,
				value,
				nil,
				parameterType,
				nil,
			),
		},
		resultType,
		b.factory.Block(
			[]tsgo.Statement{
				b.factory.IfStatement(
					b.factory.BinaryExpression(
						nil,
						value,
						nil,
						b.factory.BinaryOperatorToken(
							tsgo.BinaryOperatorEqualsEqualsEqualsToken,
						),
						b.undefined(),
					),
					b.factory.Block(
						[]tsgo.Statement{
							b.factory.ReturnStatement(b.undefined()),
						},
						true,
					),
					nil,
				),
				b.factory.ExpressionStatement(b.unresolved()),
			},
			true,
		),
	)
}

func (b builder) genericType() tsgo.TypeNode {
	return b.typeReference("P")
}

func (b builder) unsafeType() tsgo.TypeNode {
	return b.typeReference(b.className)
}

func (b builder) typeReference(name string) tsgo.TypeReferenceNode {
	return b.factory.TypeReferenceNode(b.id(name), nil)
}

func (b builder) integerType() tsgo.TypeNode {
	return b.factory.UnionTypeNode([]tsgo.TypeNode{
		b.factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindNumberKeyword),
		b.factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindBigIntKeyword),
	})
}

func (b builder) integerTypeParameter(name string) tsgo.TypeParameterDeclaration {
	return b.factory.TypeParameterDeclaration(
		nil,
		b.id(name),
		b.integerType(),
		nil,
		nil,
	)
}

func (b builder) parameter(name string, target tsgo.TypeNode) tsgo.ParameterDeclaration {
	return b.factory.ParameterDeclaration(
		nil,
		nil,
		b.id(name),
		nil,
		target,
		nil,
	)
}

func (b builder) equals(left, right tsgo.Expression) tsgo.BinaryExpression {
	return b.factory.BinaryExpression(
		nil,
		left,
		nil,
		b.factory.BinaryOperatorToken(
			tsgo.BinaryOperatorEqualsEqualsEqualsToken,
		),
		right,
	)
}

func (b builder) nullable(target tsgo.TypeNode) tsgo.TypeNode {
	return b.factory.UnionTypeNode([]tsgo.TypeNode{
		target,
		b.factory.KeywordTypeNode(
			tsgo.KeywordTypeSyntaxKindUndefinedKeyword,
		),
	})
}

func (b builder) typeParameter(
	name string,
) tsgo.TypeParameterDeclaration {
	return b.factory.TypeParameterDeclaration(
		nil,
		b.id(name),
		nil,
		nil,
		nil,
	)
}

func (b builder) unresolved() tsgo.CallExpression {
	return panicruntime.Call(
		b.factory,
		b.panicName,
		b.factory.StringLiteral(
			unresolvedPlaceholder,
			tsgo.TokenFlagsNone,
		),
	)
}

func (b builder) undefined() tsgo.Expression {
	return b.id("undefined")
}

func (b builder) id(name string) tsgo.Identifier {
	return b.factory.Identifier(name)
}
