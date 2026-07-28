package maprepresentation

import (
	panicruntime "github.com/tsoniclang/gotots/internal/emit/runtime/panic"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func (b specializationBuilder) foundType() tsgo.TypeNode {
	return b.factory.UnionTypeNode([]tsgo.TypeNode{
		b.factory.TupleTypeNode([]tsgo.TypeNode{
			b.bucketType(),
			b.numberType(),
		}),
		b.undefinedType(),
	})
}

func (b specializationBuilder) findMethod() tsgo.MethodDeclaration {
	buckets := b.id("buckets")
	bucket := b.id("bucket")
	index := b.id("index")
	entry := b.element(bucket, index)
	return b.method(
		[]tsgo.ModifierLike{b.factory.PrivateKeyword()},
		specializationFindOperation,
		[]tsgo.ParameterDeclaration{b.parameter("key", b.keyType)},
		b.foundType(),
		b.variable(
			tsgo.NodeFlagsConst,
			"buckets",
			b.storageType(),
			b.property(b.factory.ThisExpression(), "buckets"),
		),
		b.factory.IfStatement(
			b.undefined(buckets),
			b.returnBlock(b.id("undefined")),
			nil,
		),
		b.variable(
			tsgo.NodeFlagsConst,
			"bucket",
			b.factory.UnionTypeNode([]tsgo.TypeNode{
				b.bucketType(),
				b.undefinedType(),
			}),
			b.call(
				buckets,
				"get",
				b.staticCall(
					specializationHashOperation,
					b.id("key"),
				),
			),
		),
		b.factory.IfStatement(
			b.undefined(bucket),
			b.returnBlock(b.id("undefined")),
			nil,
		),
		b.factory.ForStatement(
			b.factory.VariableDeclarationList(
				[]tsgo.VariableDeclaration{
					b.factory.VariableDeclaration(
						index,
						nil,
						nil,
						b.number("0"),
					),
				},
				tsgo.NodeFlagsLet,
			),
			b.binary(
				index,
				tsgo.BinaryOperatorLessThanToken,
				b.property(bucket, "length"),
			),
			b.factory.PostfixUnaryExpression(
				index,
				tsgo.PostfixUnaryExpressionOperatorKindPlusPlusToken,
			),
			b.factory.Block(
				[]tsgo.Statement{b.factory.IfStatement(
					b.staticCall(
						specializationEqualOperation,
						b.element(entry, b.number("0")),
						b.id("key"),
					),
					b.returnBlock(
						b.factory.ArrayLiteralExpression(
							[]tsgo.Expression{bucket, index},
							false,
						),
					),
					nil,
				)},
				true,
			),
		),
		b.factory.ReturnStatement(b.id("undefined")),
	)
}

func (b specializationBuilder) lookupMethod() tsgo.MethodDeclaration {
	found := b.id("found")
	return b.method(
		nil,
		b.members.lookup,
		[]tsgo.ParameterDeclaration{b.parameter("key", b.keyType)},
		b.valueType,
		b.variable(
			tsgo.NodeFlagsConst,
			"found",
			b.foundType(),
			b.call(
				b.factory.ThisExpression(),
				specializationFindOperation,
				b.id("key"),
			),
		),
		b.factory.ReturnStatement(
			b.staticCall(
				specializationCopyValueOperation,
				b.factory.ConditionalExpression(
					b.undefined(found),
					b.factory.QuestionToken(),
					b.property(b.factory.ThisExpression(), "zeroValue"),
					b.factory.ColonToken(),
					b.foundValue(found),
				),
			),
		),
	)
}

func (b specializationBuilder) lookupOKMethod() tsgo.MethodDeclaration {
	found := b.id("found")
	resultType := b.factory.TupleTypeNode([]tsgo.TypeNode{
		b.valueType,
		b.booleanType(),
	})
	return b.method(
		nil,
		b.members.lookupOK,
		[]tsgo.ParameterDeclaration{b.parameter("key", b.keyType)},
		resultType,
		b.variable(
			tsgo.NodeFlagsConst,
			"found",
			b.foundType(),
			b.call(
				b.factory.ThisExpression(),
				specializationFindOperation,
				b.id("key"),
			),
		),
		b.factory.IfStatement(
			b.undefined(found),
			b.returnBlock(
				b.factory.ArrayLiteralExpression(
					[]tsgo.Expression{
						b.staticCall(
							specializationCopyValueOperation,
							b.property(b.factory.ThisExpression(), "zeroValue"),
						),
						b.factory.FalseLiteral(),
					},
					false,
				),
			),
			nil,
		),
		b.factory.ReturnStatement(
			b.factory.ArrayLiteralExpression(
				[]tsgo.Expression{
					b.staticCall(
						specializationCopyValueOperation,
						b.foundValue(found),
					),
					b.factory.TrueLiteral(),
				},
				false,
			),
		),
	)
}

func (b specializationBuilder) foundValue(
	found tsgo.Expression,
) tsgo.Expression {
	bucket := b.element(found, b.number("0"))
	index := b.element(found, b.number("1"))
	return b.element(
		b.element(bucket, index),
		b.number("1"),
	)
}

func (b specializationBuilder) storeMethod() tsgo.MethodDeclaration {
	buckets := b.id("buckets")
	hash := b.id("hash")
	bucket := b.id("bucket")
	entry := b.id("entry")
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
			"buckets",
			b.storageType(),
			b.property(b.factory.ThisExpression(), "buckets"),
		),
		b.factory.IfStatement(
			b.undefined(buckets),
			b.factory.Block(
				[]tsgo.Statement{b.factory.ExpressionStatement(
					panicCall(b),
				)},
				true,
			),
			nil,
		),
		b.variable(
			tsgo.NodeFlagsConst,
			"hash",
			b.numberType(),
			b.staticCall(specializationHashOperation, b.id("key")),
		),
		b.variable(
			tsgo.NodeFlagsLet,
			"bucket",
			b.factory.UnionTypeNode([]tsgo.TypeNode{
				b.bucketType(),
				b.undefinedType(),
			}),
			b.call(buckets, "get", hash),
		),
		b.factory.IfStatement(
			b.undefined(bucket),
			b.factory.Block(
				[]tsgo.Statement{
					b.factory.ExpressionStatement(
						b.assign(
							bucket,
							b.factory.ArrayLiteralExpression(nil, false),
						),
					),
					b.factory.ExpressionStatement(
						b.call(buckets, "set", hash, bucket),
					),
				},
				true,
			),
			nil,
		),
		b.forEntries(
			bucket,
			b.factory.IfStatement(
				b.staticCall(
					specializationEqualOperation,
					b.element(entry, b.number("0")),
					b.id("key"),
				),
				b.factory.Block(
					[]tsgo.Statement{
						b.factory.ExpressionStatement(
							b.assign(
								b.element(entry, b.number("1")),
								b.staticCall(
									specializationCopyValueOperation,
									b.id("value"),
								),
							),
						),
						b.factory.ReturnStatement(nil),
					},
					true,
				),
				nil,
			),
		),
		b.factory.ExpressionStatement(
			b.call(
				bucket,
				"push",
				b.factory.ArrayLiteralExpression(
					[]tsgo.Expression{
						b.staticCall(
							specializationCopyOperation,
							b.id("key"),
						),
						b.staticCall(
							specializationCopyValueOperation,
							b.id("value"),
						),
					},
					false,
				),
			),
		),
		b.factory.ExpressionStatement(
			b.factory.PostfixUnaryExpression(
				b.property(b.factory.ThisExpression(), "count"),
				tsgo.PostfixUnaryExpressionOperatorKindPlusPlusToken,
			),
		),
	)
}

func panicCall(b specializationBuilder) tsgo.CallExpression {
	return panicruntime.Call(
		b.factory,
		b.panicName,
		b.factory.StringLiteral(
			"assignment to entry in nil map",
			tsgo.TokenFlagsNone,
		),
	)
}

func (b specializationBuilder) deleteMethod() tsgo.MethodDeclaration {
	found := b.id("found")
	bucket := b.element(found, b.number("0"))
	index := b.element(found, b.number("1"))
	hash := b.staticCall(specializationHashOperation, b.id("key"))
	return b.method(
		nil,
		b.members.deleteMember,
		[]tsgo.ParameterDeclaration{b.parameter("key", b.keyType)},
		b.voidType(),
		b.variable(
			tsgo.NodeFlagsConst,
			"found",
			b.foundType(),
			b.call(
				b.factory.ThisExpression(),
				specializationFindOperation,
				b.id("key"),
			),
		),
		b.factory.IfStatement(
			b.undefined(found),
			b.factory.Block(
				[]tsgo.Statement{b.factory.ReturnStatement(nil)},
				true,
			),
			nil,
		),
		b.factory.ExpressionStatement(
			b.call(bucket, "splice", index, b.number("1")),
		),
		b.factory.IfStatement(
			b.binary(
				b.property(bucket, "length"),
				tsgo.BinaryOperatorEqualsEqualsEqualsToken,
				b.number("0"),
			),
			b.factory.Block(
				[]tsgo.Statement{b.factory.IfStatement(
					b.factory.PrefixUnaryExpression(
						tsgo.PrefixUnaryExpressionOperatorKindExclamationToken,
						b.undefined(
							b.property(
								b.factory.ThisExpression(),
								"buckets",
							),
						),
					),
					b.factory.Block(
						[]tsgo.Statement{b.factory.ExpressionStatement(
							b.call(
								b.property(
									b.factory.ThisExpression(),
									"buckets",
								),
								"delete",
								hash,
							),
						)},
						true,
					),
					nil,
				)},
				true,
			),
			nil,
		),
		b.factory.ExpressionStatement(
			b.factory.PostfixUnaryExpression(
				b.property(b.factory.ThisExpression(), "count"),
				tsgo.PostfixUnaryExpressionOperatorKindMinusMinusToken,
			),
		),
	)
}

func (b specializationBuilder) lengthMethod() tsgo.MethodDeclaration {
	return b.method(
		nil,
		b.members.length,
		nil,
		b.numberType(),
		b.factory.ReturnStatement(
			b.property(b.factory.ThisExpression(), "count"),
		),
	)
}

func (b specializationBuilder) isNilMethod() tsgo.MethodDeclaration {
	return b.method(
		nil,
		b.members.isNil,
		nil,
		b.booleanType(),
		b.factory.ReturnStatement(
			b.undefined(
				b.property(b.factory.ThisExpression(), "buckets"),
			),
		),
	)
}

func (b specializationBuilder) clearMethod() tsgo.MethodDeclaration {
	return b.method(
		nil,
		b.members.clear,
		nil,
		b.voidType(),
		b.factory.IfStatement(
			b.undefined(b.property(b.factory.ThisExpression(), "buckets")),
			b.factory.Block([]tsgo.Statement{
				b.factory.ReturnStatement(nil),
			}, true),
			nil,
		),
		b.factory.ExpressionStatement(
			b.call(
				b.property(b.factory.ThisExpression(), "buckets"),
				"clear",
			),
		),
		b.factory.ExpressionStatement(
			b.assign(
				b.property(b.factory.ThisExpression(), "count"),
				b.number("0"),
			),
		),
	)
}
