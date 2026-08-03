package unsafepointer

import "github.com/tsoniclang/gotots/internal/target/tsgo"

const (
	FromName        = "from"
	ToName          = "to"
	FromIntegerName = "fromInteger"
	ToIntegerName   = "toInteger"
	addressName     = "address"
	atName          = "at"
)

func Build(
	factory tsgo.Factory,
	className string,
	codecName string,
	panicName string,
	pointerName string,
	pointerMemoryName string,
	denseIndexName string,
) tsgo.ClassDeclaration {
	target := builder{
		factory:           factory,
		className:         className,
		codecName:         codecName,
		panicName:         panicName,
		pointerName:       pointerName,
		pointerMemoryName: pointerMemoryName,
		denseIndexName:    denseIndexName,
	}
	return factory.ClassDeclaration(
		[]tsgo.ModifierLike{factory.ExportKeyword()},
		factory.Identifier(className),
		nil,
		nil,
		[]tsgo.ClassElement{
			target.nextBaseProperty(),
			target.rootsProperty(),
			target.locationsProperty(),
			target.allocationsProperty(),
			target.constructor(),
			target.addressGetter(),
			target.flush(),
			target.refresh(),
			target.at(),
			target.from(),
			target.to(),
			target.fromInteger(),
			target.toIntegerNumberOverload(),
			target.toIntegerBigIntOverload(),
			target.toInteger(),
		},
	)
}

func (b builder) nextBaseProperty() tsgo.PropertyDeclaration {
	return b.factory.PropertyDeclaration(
		[]tsgo.ModifierLike{
			b.factory.PrivateKeyword(),
			b.factory.StaticKeyword(),
		},
		b.id("nextBase"),
		nil,
		b.numberType(),
		b.number("4096"),
	)
}

func (b builder) rootsProperty() tsgo.PropertyDeclaration {
	target := b.typeReference("WeakMap", b.objectType(), b.unsafeType())
	return b.staticCollectionProperty("roots", target, "WeakMap")
}

func (b builder) locationsProperty() tsgo.PropertyDeclaration {
	target := b.typeReference("WeakMap", b.objectType(), b.unsafeType())
	return b.staticCollectionProperty("locations", target, "WeakMap")
}

func (b builder) allocationsProperty() tsgo.PropertyDeclaration {
	target := b.factory.ArrayTypeNode(b.unsafeType())
	return b.factory.PropertyDeclaration(
		[]tsgo.ModifierLike{
			b.factory.PrivateKeyword(),
			b.factory.StaticKeyword(),
			b.factory.ReadonlyKeyword(),
		},
		b.id("allocations"),
		nil,
		target,
		b.factory.ArrayLiteralExpression(nil, false),
	)
}

func (b builder) staticCollectionProperty(
	name string,
	target tsgo.TypeNode,
	constructor string,
) tsgo.PropertyDeclaration {
	return b.factory.PropertyDeclaration(
		[]tsgo.ModifierLike{
			b.factory.PrivateKeyword(),
			b.factory.StaticKeyword(),
			b.factory.ReadonlyKeyword(),
		},
		b.id(name),
		nil,
		target,
		b.factory.NewExpression(b.id(constructor), nil, nil),
	)
}

func (b builder) constructor() tsgo.ConstructorDeclaration {
	privateReadonly := []tsgo.ModifierLike{
		b.factory.PrivateKeyword(),
		b.factory.ReadonlyKeyword(),
	}
	return b.factory.ConstructorDeclaration(
		[]tsgo.ModifierLike{b.factory.PrivateKeyword()},
		nil,
		[]tsgo.ParameterDeclaration{
			b.parameter("base", b.numberType(), privateReadonly),
			b.parameter("offset", b.numberType(), privateReadonly),
			b.parameter("length", b.numberType(), privateReadonly),
			b.parameter("readBytes", b.readBytesType(), privateReadonly),
			b.parameter("writeBytes", b.writeBytesType(), privateReadonly),
			b.parameter("children", b.childrenType(), privateReadonly),
			b.parameter("flushes", b.callbackArrayType(), privateReadonly),
			b.parameter("refreshes", b.callbackArrayType(), privateReadonly),
		},
		nil,
		b.factory.Block([]tsgo.Statement{
			b.factory.ExpressionStatement(b.call(
				b.property(b.id(b.className), "locations"),
				"set",
				b.factory.ThisExpression(),
				b.factory.ThisExpression(),
			)),
		}, true),
	)
}

func (b builder) callbackArrayType() tsgo.TypeNode {
	return b.factory.ArrayTypeNode(
		b.factory.FunctionTypeNode(nil, nil, b.voidType()),
	)
}

func (b builder) flush() tsgo.MethodDeclaration {
	return b.callbackMethod("flush", "flushes")
}

func (b builder) refresh() tsgo.MethodDeclaration {
	return b.callbackMethod("refresh", "refreshes")
}

func (b builder) callbackMethod(name string, property string) tsgo.MethodDeclaration {
	callback := b.id("callback")
	return b.method(
		[]tsgo.ModifierLike{b.factory.PrivateKeyword()},
		name,
		nil,
		nil,
		b.voidType(),
		b.factory.ForOfStatement(
			nil,
			b.factory.VariableDeclarationList(
				[]tsgo.VariableDeclaration{b.factory.VariableDeclaration(
					callback,
					nil,
					nil,
					nil,
				)},
				tsgo.NodeFlagsConst,
			),
			b.property(b.factory.ThisExpression(), property),
			b.factory.Block([]tsgo.Statement{b.factory.ExpressionStatement(
				b.factory.CallExpression(callback, nil, nil, nil, tsgo.NodeFlagsNone),
			)}, true),
		),
	)
}

func (b builder) addressGetter() tsgo.GetAccessorDeclaration {
	return b.factory.GetAccessorDeclaration(
		[]tsgo.ModifierLike{b.factory.PrivateKeyword()},
		b.id(addressName),
		nil,
		nil,
		b.numberType(),
		b.factory.Block([]tsgo.Statement{b.factory.ReturnStatement(b.binary(
			b.property(b.factory.ThisExpression(), "base"),
			tsgo.BinaryOperatorPlusToken,
			b.property(b.factory.ThisExpression(), "offset"),
		))}, true),
	)
}
