package pointer

import "github.com/tsoniclang/gotots/internal/target/tsgo"

func (b builder) logicalProperty() tsgo.PropertyDeclaration {
	identityType := b.factory.FunctionTypeNode(
		nil,
		[]tsgo.ParameterDeclaration{
			b.parameter("value", b.typeL()),
		},
		b.typeL(),
	)
	return b.factory.PropertyDeclaration(
		[]tsgo.ModifierLike{
			b.factory.DeclareKeyword(),
			b.factory.PrivateKeyword(),
			b.factory.ReadonlyKeyword(),
		},
		b.id("logical"),
		nil,
		identityType,
		nil,
	)
}

func (b builder) rootsProperty() tsgo.PropertyDeclaration {
	targetType := b.typeReference(
		"WeakMap",
		b.objectType(),
		b.objectType(),
	)
	return b.factory.PropertyDeclaration(
		[]tsgo.ModifierLike{
			b.factory.PrivateKeyword(),
			b.factory.StaticKeyword(),
			b.factory.ReadonlyKeyword(),
		},
		b.id("roots"),
		nil,
		targetType,
		b.factory.NewExpression(
			b.id("WeakMap"),
			[]tsgo.TypeNode{b.objectType(), b.objectType()},
			nil,
		),
	)
}

func (b builder) childrenProperty() tsgo.PropertyDeclaration {
	childMap := b.typeReference(
		"Map",
		b.addressKeyType(),
		b.objectType(),
	)
	targetType := b.typeReference(
		"WeakMap",
		b.objectType(),
		childMap,
	)
	return b.factory.PropertyDeclaration(
		[]tsgo.ModifierLike{
			b.factory.PrivateKeyword(),
			b.factory.StaticKeyword(),
			b.factory.ReadonlyKeyword(),
		},
		b.id("children"),
		nil,
		targetType,
		b.factory.NewExpression(
			b.id("WeakMap"),
			[]tsgo.TypeNode{b.objectType(), childMap},
			nil,
		),
	)
}

func (b builder) constructor() tsgo.ConstructorDeclaration {
	privateReadonly := []tsgo.ModifierLike{
		b.factory.PrivateKeyword(),
		b.factory.ReadonlyKeyword(),
	}
	readType := b.factory.FunctionTypeNode(nil, nil, b.typeS())
	writeType := b.factory.FunctionTypeNode(
		nil,
		[]tsgo.ParameterDeclaration{b.parameter("value", b.typeS())},
		b.factory.KeywordTypeNode(
			tsgo.KeywordTypeSyntaxKindVoidKeyword,
		),
	)
	return b.factory.ConstructorDeclaration(
		[]tsgo.ModifierLike{b.factory.PrivateKeyword()},
		nil,
		[]tsgo.ParameterDeclaration{
			b.factory.ParameterDeclaration(
				privateReadonly,
				nil,
				b.id("address"),
				nil,
				b.objectType(),
				nil,
			),
			b.factory.ParameterDeclaration(
				privateReadonly,
				nil,
				b.id("read"),
				nil,
				readType,
				nil,
			),
			b.factory.ParameterDeclaration(
				privateReadonly,
				nil,
				b.id("write"),
				nil,
				writeType,
				nil,
			),
		},
		nil,
		b.factory.Block(nil, true),
	)
}

func (b builder) rootMethod() tsgo.MethodDeclaration {
	roots := b.property(b.id(b.className), "roots")
	return b.method(
		[]tsgo.ModifierLike{
			b.factory.PrivateKeyword(),
			b.factory.StaticKeyword(),
		},
		"root",
		nil,
		[]tsgo.ParameterDeclaration{b.parameter("owner", b.objectType())},
		b.objectType(),
		b.variable(
			tsgo.NodeFlagsLet,
			"address",
			nil,
			b.call(roots, "get", b.id("owner")),
		),
		b.factory.IfStatement(
			b.binary(
				b.id("address"),
				tsgo.BinaryOperatorEqualsEqualsEqualsToken,
				b.undefined(),
			),
			b.factory.Block(
				[]tsgo.Statement{
					b.factory.ExpressionStatement(
						b.assign(
							b.id("address"),
							b.factory.ObjectLiteralExpression(nil, false),
						),
					),
					b.factory.ExpressionStatement(
						b.call(
							roots,
							"set",
							b.id("owner"),
							b.id("address"),
						),
					),
				},
				true,
			),
			nil,
		),
		b.factory.ReturnStatement(b.id("address")),
	)
}

func (b builder) childMethod() tsgo.MethodDeclaration {
	childrenProperty := b.property(b.id(b.className), "children")
	return b.method(
		[]tsgo.ModifierLike{
			b.factory.PrivateKeyword(),
			b.factory.StaticKeyword(),
		},
		"child",
		nil,
		[]tsgo.ParameterDeclaration{
			b.parameter("parent", b.objectType()),
			b.parameter("key", b.addressKeyType()),
		},
		b.objectType(),
		b.variable(
			tsgo.NodeFlagsLet,
			"children",
			nil,
			b.call(childrenProperty, "get", b.id("parent")),
		),
		b.factory.IfStatement(
			b.binary(
				b.id("children"),
				tsgo.BinaryOperatorEqualsEqualsEqualsToken,
				b.undefined(),
			),
			b.factory.Block(
				[]tsgo.Statement{
					b.factory.ExpressionStatement(
						b.assign(
							b.id("children"),
							b.factory.NewExpression(
								b.id("Map"),
								[]tsgo.TypeNode{
									b.addressKeyType(),
									b.objectType(),
								},
								nil,
							),
						),
					),
					b.factory.ExpressionStatement(
						b.call(
							childrenProperty,
							"set",
							b.id("parent"),
							b.id("children"),
						),
					),
				},
				true,
			),
			nil,
		),
		b.variable(
			tsgo.NodeFlagsLet,
			"address",
			nil,
			b.call(b.id("children"), "get", b.id("key")),
		),
		b.factory.IfStatement(
			b.binary(
				b.id("address"),
				tsgo.BinaryOperatorEqualsEqualsEqualsToken,
				b.undefined(),
			),
			b.factory.Block(
				[]tsgo.Statement{
					b.factory.ExpressionStatement(
						b.assign(
							b.id("address"),
							b.factory.ObjectLiteralExpression(nil, false),
						),
					),
					b.factory.ExpressionStatement(
						b.call(
							b.id("children"),
							"set",
							b.id("key"),
							b.id("address"),
						),
					),
				},
				true,
			),
			nil,
		),
		b.factory.ReturnStatement(b.id("address")),
	)
}
