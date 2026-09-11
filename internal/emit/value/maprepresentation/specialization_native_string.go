package maprepresentation

import (
	runtimestring "github.com/tsoniclang/gotots/internal/emit/runtime/stringvalue"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func (b specializationBuilder) nativeIndexType() tsgo.TypeNode {
	if b.stringKey {
		return b.factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindStringKeyword)
	}
	return b.storageKeyType
}

func (b specializationBuilder) nativeValueType() tsgo.TypeNode {
	if b.stringKey {
		return b.factory.TupleTypeNode([]tsgo.TypeNode{b.storageKeyType, b.valueType})
	}
	return b.valueType
}

func (b specializationBuilder) nativeIndex(key tsgo.Expression) tsgo.Expression {
	if b.stringKey {
		return b.call(key, runtimestring.TextMember)
	}
	return key
}

func (b specializationBuilder) nativeStoredValue(entry tsgo.Expression) tsgo.Expression {
	if b.stringKey {
		return b.element(entry, b.number("1"))
	}
	return entry
}

func (b specializationBuilder) nativeEntry(key, value tsgo.Expression) tsgo.Expression {
	if b.stringKey {
		return b.factory.ArrayLiteralExpression([]tsgo.Expression{key, value}, false)
	}
	return value
}

func (b specializationBuilder) nativeStringKeysMethod() tsgo.MethodDeclaration {
	values := b.id("values")
	entry := b.id("entry")
	return b.method(nil, b.members.keys, nil, b.factory.ArrayTypeNode(b.keyType),
		b.variable(tsgo.NodeFlagsConst, "values", b.nativeStorageType(), b.property(b.factory.ThisExpression(), "values")),
		b.factory.IfStatement(b.undefined(values), b.returnBlock(b.factory.ArrayLiteralExpression(nil, false)), nil),
		b.factory.ReturnStatement(b.factory.CallExpression(b.property(b.id("Array"), "from"), nil, nil, []tsgo.Expression{
			b.call(values, "values"), b.factory.ArrowFunction(nil, nil, []tsgo.ParameterDeclaration{b.parameter("entry", b.nativeValueType())},
				b.keyType, b.factory.EqualsGreaterThanToken(), b.factory.Block([]tsgo.Statement{
					b.factory.ReturnStatement(b.reifyKeyExpression(b.element(entry, b.number("0")))),
				}, true)),
		}, tsgo.NodeFlagsNone)))
}
