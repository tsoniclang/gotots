package maprepresentation

import (
	panicruntime "github.com/tsoniclang/gotots/internal/emit/runtime/panic"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func (b specializationBuilder) buildNative() []tsgo.ClassElement {
	return []tsgo.ClassElement{
		b.nativeConstructor(),
		b.operationMethod(
			specializationZeroOperation,
			nil,
			b.valueType,
			b.zero,
		),
		b.operationMethod(
			specializationCopyValueOperation,
			[]tsgo.ParameterDeclaration{
				b.parameter("$value", b.valueType),
			},
			b.valueType,
			b.copyValue,
		),
		b.nativeNilMethod(),
		b.nativeMakeMethod(),
		b.nativeLookupMethod(),
		b.nativeLookupOKMethod(),
		b.nativeStoreMethod(),
		b.nativeDeleteMethod(),
		b.nativeLengthMethod(),
		b.nativeIsNilMethod(),
		b.nativeClearMethod(),
		b.nativeKeysMethod(),
	}
}

func (b specializationBuilder) nativeCellType() tsgo.TypeNode {
	return b.factory.TupleTypeNode([]tsgo.TypeNode{b.valueType})
}

func (b specializationBuilder) nativeStorageType() tsgo.TypeNode {
	return b.factory.UnionTypeNode([]tsgo.TypeNode{
		b.factory.TypeReferenceNode(
			b.id("Map"),
			[]tsgo.TypeNode{b.storageKeyType, b.nativeCellType()},
		),
		b.undefinedType(),
	})
}

func (b specializationBuilder) nativeConstructor() tsgo.ConstructorDeclaration {
	return b.factory.ConstructorDeclaration(
		[]tsgo.ModifierLike{b.factory.PrivateKeyword()},
		nil,
		[]tsgo.ParameterDeclaration{
			b.parameterProperty("zeroValue", b.valueType, true),
			b.parameterProperty("values", b.nativeStorageType(), true),
		},
		nil,
		b.factory.Block([]tsgo.Statement{b.superCall()}, true),
	)
}

func (b specializationBuilder) nativeNilMethod() tsgo.MethodDeclaration {
	return b.method(
		[]tsgo.ModifierLike{b.factory.StaticKeyword()},
		b.members.nilMember,
		nil,
		b.classType(),
		b.factory.ReturnStatement(b.factory.NewExpression(
			b.id(b.className),
			nil,
			[]tsgo.Expression{
				b.staticCall(specializationZeroOperation),
				b.id("undefined"),
			},
		)),
	)
}

func (b specializationBuilder) nativeMakeMethod() tsgo.MethodDeclaration {
	result := b.id("result")
	entry := b.id("entry")
	return b.method(
		[]tsgo.ModifierLike{b.factory.StaticKeyword()},
		b.members.makeMember,
		[]tsgo.ParameterDeclaration{
			b.parameter(
				"size",
				b.factory.UnionTypeNode([]tsgo.TypeNode{
					b.numberType(),
					b.factory.KeywordTypeNode(
						tsgo.KeywordTypeSyntaxKindBigIntKeyword,
					),
				}),
			),
			b.parameter("entries", b.factory.ArrayTypeNode(b.entryType())),
		},
		b.classType(),
		b.variable(
			tsgo.NodeFlagsConst,
			"result",
			b.classType(),
			b.factory.NewExpression(
				b.id(b.className),
				nil,
				[]tsgo.Expression{
					b.staticCall(specializationZeroOperation),
					b.factory.NewExpression(
						b.id("Map"),
						[]tsgo.TypeNode{
							b.storageKeyType,
							b.nativeCellType(),
						},
						nil,
					),
				},
			),
		),
		b.forEntries(
			b.id("entries"),
			b.factory.ExpressionStatement(b.call(
				result,
				b.members.store,
				b.element(entry, b.number("0")),
				b.element(entry, b.number("1")),
			)),
		),
		b.factory.ReturnStatement(result),
	)
}

func (b specializationBuilder) nativeLookupMethod() tsgo.MethodDeclaration {
	values := b.id("values")
	entry := b.id("entry")
	return b.method(
		nil,
		b.members.lookup,
		[]tsgo.ParameterDeclaration{b.parameter("key", b.keyType)},
		b.valueType,
		b.variable(
			tsgo.NodeFlagsConst,
			"values",
			b.nativeStorageType(),
			b.property(b.factory.ThisExpression(), "values"),
		),
		b.factory.IfStatement(
			b.undefined(values),
			b.returnBlock(b.staticCall(
				specializationCopyValueOperation,
				b.property(b.factory.ThisExpression(), "zeroValue"),
			)),
			nil,
		),
		b.variable(
			tsgo.NodeFlagsConst,
			"entry",
			b.factory.UnionTypeNode([]tsgo.TypeNode{
				b.nativeCellType(),
				b.undefinedType(),
			}),
			b.call(values, "get", b.id("key")),
		),
		b.factory.ReturnStatement(b.staticCall(
			specializationCopyValueOperation,
			b.factory.ConditionalExpression(
				b.undefined(entry),
				b.factory.QuestionToken(),
				b.property(b.factory.ThisExpression(), "zeroValue"),
				b.factory.ColonToken(),
				b.element(entry, b.number("0")),
			),
		)),
	)
}

func (b specializationBuilder) nativeLookupOKMethod() tsgo.MethodDeclaration {
	values := b.id("values")
	entry := b.id("entry")
	resultType := b.factory.TupleTypeNode([]tsgo.TypeNode{
		b.valueType,
		b.booleanType(),
	})
	missing := func() tsgo.Expression {
		return b.factory.ArrayLiteralExpression(
			[]tsgo.Expression{
				b.staticCall(
					specializationCopyValueOperation,
					b.property(b.factory.ThisExpression(), "zeroValue"),
				),
				b.factory.FalseLiteral(),
			},
			false,
		)
	}
	return b.method(
		nil,
		b.members.lookupOK,
		[]tsgo.ParameterDeclaration{b.parameter("key", b.keyType)},
		resultType,
		b.variable(
			tsgo.NodeFlagsConst,
			"values",
			b.nativeStorageType(),
			b.property(b.factory.ThisExpression(), "values"),
		),
		b.factory.IfStatement(
			b.undefined(values),
			b.returnBlock(missing()),
			nil,
		),
		b.variable(
			tsgo.NodeFlagsConst,
			"entry",
			b.factory.UnionTypeNode([]tsgo.TypeNode{
				b.nativeCellType(),
				b.undefinedType(),
			}),
			b.call(values, "get", b.id("key")),
		),
		b.factory.IfStatement(
			b.undefined(entry),
			b.returnBlock(missing()),
			nil,
		),
		b.factory.ReturnStatement(b.factory.ArrayLiteralExpression(
			[]tsgo.Expression{
				b.staticCall(
					specializationCopyValueOperation,
					b.element(entry, b.number("0")),
				),
				b.factory.TrueLiteral(),
			},
			false,
		)),
	)
}

func (b specializationBuilder) nativeStoreMethod() tsgo.MethodDeclaration {
	values := b.id("values")
	return b.method(
		nil,
		b.members.store,
		[]tsgo.ParameterDeclaration{
			b.parameter("key", b.keyType),
			b.parameter("value", b.valueType),
		},
		b.voidType(),
		b.variable(
			tsgo.NodeFlagsConst,
			"values",
			b.nativeStorageType(),
			b.property(b.factory.ThisExpression(), "values"),
		),
		b.factory.IfStatement(
			b.undefined(values),
			b.factory.ExpressionStatement(panicruntime.Call(
				b.factory,
				b.panicName,
				b.factory.StringLiteral(
					"assignment to entry in nil map",
					tsgo.TokenFlagsNone,
				),
			)),
			nil,
		),
		b.factory.ExpressionStatement(b.call(
			values,
			"set",
			b.id("key"),
			b.factory.ArrayLiteralExpression(
				[]tsgo.Expression{b.staticCall(
					specializationCopyValueOperation,
					b.id("value"),
				)},
				false,
			),
		)),
	)
}

func (b specializationBuilder) nativeDeleteMethod() tsgo.MethodDeclaration {
	values := b.id("values")
	return b.method(
		nil,
		b.members.deleteMember,
		[]tsgo.ParameterDeclaration{b.parameter("key", b.keyType)},
		b.voidType(),
		b.variable(
			tsgo.NodeFlagsConst,
			"values",
			b.nativeStorageType(),
			b.property(b.factory.ThisExpression(), "values"),
		),
		b.factory.IfStatement(
			b.factory.PrefixUnaryExpression(
				tsgo.PrefixUnaryExpressionOperatorKindExclamationToken,
				b.undefined(values),
			),
			b.factory.ExpressionStatement(b.call(values, "delete", b.id("key"))),
			nil,
		),
	)
}

func (b specializationBuilder) nativeLengthMethod() tsgo.MethodDeclaration {
	values := b.property(b.factory.ThisExpression(), "values")
	return b.method(
		nil,
		b.members.length,
		nil,
		b.numberType(),
		b.factory.ReturnStatement(b.factory.ConditionalExpression(
			b.undefined(values),
			b.factory.QuestionToken(),
			b.number("0"),
			b.factory.ColonToken(),
			b.property(values, "size"),
		)),
	)
}

func (b specializationBuilder) nativeIsNilMethod() tsgo.MethodDeclaration {
	return b.method(
		nil,
		b.members.isNil,
		nil,
		b.booleanType(),
		b.factory.ReturnStatement(b.undefined(
			b.property(b.factory.ThisExpression(), "values"),
		)),
	)
}

func (b specializationBuilder) nativeClearMethod() tsgo.MethodDeclaration {
	values := b.id("values")
	return b.method(
		nil,
		b.members.clear,
		nil,
		b.voidType(),
		b.variable(
			tsgo.NodeFlagsConst,
			"values",
			b.nativeStorageType(),
			b.property(b.factory.ThisExpression(), "values"),
		),
		b.factory.IfStatement(
			b.factory.PrefixUnaryExpression(
				tsgo.PrefixUnaryExpressionOperatorKindExclamationToken,
				b.undefined(values),
			),
			b.factory.ExpressionStatement(b.call(values, "clear")),
			nil,
		),
	)
}

func (b specializationBuilder) nativeKeysMethod() tsgo.MethodDeclaration {
	values := b.id("values")
	return b.method(
		nil,
		b.members.keys,
		nil,
		b.factory.ArrayTypeNode(b.keyType),
		b.variable(
			tsgo.NodeFlagsConst,
			"values",
			b.nativeStorageType(),
			b.property(b.factory.ThisExpression(), "values"),
		),
		b.factory.IfStatement(
			b.undefined(values),
			b.returnBlock(b.factory.ArrayLiteralExpression(nil, false)),
			nil,
		),
		b.factory.ReturnStatement(b.factory.CallExpression(
			b.property(b.id("Array"), "from"),
			nil,
			nil,
			[]tsgo.Expression{b.call(values, "keys")},
			tsgo.NodeFlagsNone,
		)),
	)
}
