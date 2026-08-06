package array

import (
	"github.com/tsoniclang/gotots/internal/emit/api"
	arraymember "github.com/tsoniclang/gotots/internal/emit/runtime/array/member"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func checkMethod(
	factory tsgo.Factory,
	panicName string,
) tsgo.MethodDeclaration {
	index := factory.Identifier("index")
	offset := factory.Identifier("offset")
	negative := binary(
		factory,
		offset,
		tsgo.BinaryOperatorLessThanToken,
		factory.NumericLiteral("0", tsgo.TokenFlagsNone),
	)
	tooLarge := binary(
		factory,
		offset,
		tsgo.BinaryOperatorGreaterThanEqualsToken,
		runtimeProperty(
			factory,
			factory.ThisExpression(),
			arraymember.Length,
		),
	)
	return method(
		factory,
		[]tsgo.ModifierLike{factory.PrivateKeyword()},
		"$check",
		nil,
		[]tsgo.ParameterDeclaration{parameter(
			factory,
			nil,
			"index",
			indexType(factory),
		)},
		factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindNumberKeyword),
		[]tsgo.Statement{
			variable(
				factory,
				tsgo.NodeFlagsConst,
				"offset",
				factory.KeywordTypeNode(
					tsgo.KeywordTypeSyntaxKindNumberKeyword,
				),
				call(
					factory,
					api.TargetIntrinsicNumber.Expression(factory),
					nil,
					index,
				),
			),
			factory.IfStatement(
				binary(
					factory,
					binary(
						factory,
						factory.PrefixUnaryExpression(
							tsgo.PrefixUnaryExpressionOperatorKindExclamationToken,
							call(
								factory,
								property(
									factory,
									api.TargetIntrinsicNumber.Expression(factory),
									"isInteger",
								),
								nil,
								offset,
							),
						),
						tsgo.BinaryOperatorBarBarToken,
						negative,
					),
					tsgo.BinaryOperatorBarBarToken,
					tooLarge,
				),
				factory.Block([]tsgo.Statement{
					boundsPanic(
						factory,
						panicName,
						"array index out of bounds",
					),
				}, true),
				nil,
			),
			factory.ReturnStatement(offset),
		},
	)
}

func indexType(factory tsgo.Factory) tsgo.UnionTypeNode {
	return factory.UnionTypeNode([]tsgo.TypeNode{
		factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindNumberKeyword),
		factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindBigIntKeyword),
	})
}
