package mapruntime

import (
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

const (
	HashBooleanMember = "boolean"
	HashNumberMember  = "number"
	HashBigIntMember  = "bigint"
	HashStringMember  = "string"
	HashObjectMember  = "object"
	HashMixMember     = "mix"
)

func buildHash(factory tsgo.Factory) (tsgo.Statement, error) {
	contract, err := api.RuntimeContract(api.RuntimeMapHash)
	if err != nil {
		return nil, err
	}
	className := contract.ExportedName()
	return factory.ClassDeclaration(
		[]tsgo.ModifierLike{factory.ExportKeyword()},
		factory.Identifier(className),
		nil,
		nil,
		[]tsgo.ClassElement{
			hashObjectStorage(factory),
			hashObjectSequence(factory),
			hashBooleanMethod(factory),
			hashNumberMethod(factory),
			hashBigIntMethod(factory),
			hashStringMethod(factory, className),
			hashObjectMethod(factory, className),
			hashMixMethod(factory),
		},
	), nil
}

func hashObjectStorage(
	factory tsgo.Factory,
) tsgo.PropertyDeclaration {
	objectType := factory.KeywordTypeNode(
		tsgo.KeywordTypeSyntaxKindObjectKeyword,
	)
	numberType := factory.KeywordTypeNode(
		tsgo.KeywordTypeSyntaxKindNumberKeyword,
	)
	return factory.PropertyDeclaration(
		[]tsgo.ModifierLike{
			factory.PrivateKeyword(),
			factory.StaticKeyword(),
			factory.ReadonlyKeyword(),
		},
		factory.Identifier("objects"),
		nil,
		factory.TypeReferenceNode(
			factory.Identifier("WeakMap"),
			[]tsgo.TypeNode{objectType, numberType},
		),
		factory.NewExpression(
			factory.Identifier("WeakMap"),
			[]tsgo.TypeNode{objectType, numberType},
			nil,
		),
	)
}

func hashObjectSequence(
	factory tsgo.Factory,
) tsgo.PropertyDeclaration {
	return factory.PropertyDeclaration(
		[]tsgo.ModifierLike{
			factory.PrivateKeyword(),
			factory.StaticKeyword(),
		},
		factory.Identifier("nextObject"),
		nil,
		factory.KeywordTypeNode(
			tsgo.KeywordTypeSyntaxKindNumberKeyword,
		),
		factory.NumericLiteral("1", tsgo.TokenFlagsNone),
	)
}

func hashBooleanMethod(factory tsgo.Factory) tsgo.MethodDeclaration {
	return hashMethod(
		factory,
		HashBooleanMember,
		factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindBooleanKeyword),
		[]tsgo.Statement{factory.ReturnStatement(
			factory.ConditionalExpression(
				factory.Identifier("value"),
				factory.QuestionToken(),
				factory.NumericLiteral("1", tsgo.TokenFlagsNone),
				factory.ColonToken(),
				factory.NumericLiteral("0", tsgo.TokenFlagsNone),
			),
		)},
	)
}

func hashNumberMethod(factory tsgo.Factory) tsgo.MethodDeclaration {
	truncated := runtimeCall(
		factory,
		factory.Identifier("Math"),
		"trunc",
		factory.Identifier("value"),
	)
	return hashMethod(
		factory,
		HashNumberMember,
		factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindNumberKeyword),
		[]tsgo.Statement{factory.ReturnStatement(unsigned32(factory, truncated))},
	)
}

func hashBigIntMethod(factory tsgo.Factory) tsgo.MethodDeclaration {
	low := runtimeCall(
		factory,
		factory.Identifier("BigInt"),
		"asUintN",
		factory.NumericLiteral("32", tsgo.TokenFlagsNone),
		factory.Identifier("value"),
	)
	number := factory.CallExpression(
		api.TargetIntrinsicNumber.Expression(factory),
		nil,
		nil,
		[]tsgo.Expression{low},
		tsgo.NodeFlagsNone,
	)
	return hashMethod(
		factory,
		HashBigIntMember,
		factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindBigIntKeyword),
		[]tsgo.Statement{factory.ReturnStatement(number)},
	)
}

func hashStringMethod(
	factory tsgo.Factory,
	className string,
) tsgo.MethodDeclaration {
	index := factory.Identifier("index")
	hash := factory.Identifier("hash")
	value := factory.Identifier("value")
	next := runtimeCall(factory, value, "charCodeAt", index)
	mixed := runtimeCall(
		factory,
		factory.Identifier(className),
		HashMixMember,
		hash,
		next,
	)
	return hashMethod(
		factory,
		HashStringMember,
		factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindStringKeyword),
		[]tsgo.Statement{
			hashVariable(
				factory,
				tsgo.NodeFlagsLet,
				"hash",
				factory.NumericLiteral("2166136261", tsgo.TokenFlagsNone),
			),
			factory.ForStatement(
				factory.VariableDeclarationList(
					[]tsgo.VariableDeclaration{
						factory.VariableDeclaration(
							index,
							nil,
							nil,
							factory.NumericLiteral("0", tsgo.TokenFlagsNone),
						),
					},
					tsgo.NodeFlagsLet,
				),
				factory.BinaryExpression(
					nil,
					index,
					nil,
					factory.BinaryOperatorToken(
						tsgo.BinaryOperatorLessThanToken,
					),
					factory.PropertyAccessExpression(
						value,
						nil,
						factory.Identifier("length"),
						tsgo.NodeFlagsNone,
					),
				),
				factory.PostfixUnaryExpression(
					index,
					tsgo.PostfixUnaryExpressionOperatorKindPlusPlusToken,
				),
				factory.Block([]tsgo.Statement{
					factory.ExpressionStatement(
						factory.BinaryExpression(
							nil,
							hash,
							nil,
							factory.BinaryOperatorToken(
								tsgo.BinaryOperatorEqualsToken,
							),
							mixed,
						),
					),
				}, true),
			),
			factory.ReturnStatement(hash),
		},
	)
}

func hashObjectMethod(
	factory tsgo.Factory,
	className string,
) tsgo.MethodDeclaration {
	owner := factory.Identifier(className)
	value := factory.Identifier("value")
	result := factory.Identifier("result")
	objects := factory.PropertyAccessExpression(
		owner,
		nil,
		factory.Identifier("objects"),
		tsgo.NodeFlagsNone,
	)
	next := factory.PropertyAccessExpression(
		owner,
		nil,
		factory.Identifier("nextObject"),
		tsgo.NodeFlagsNone,
	)
	return hashMethod(
		factory,
		HashObjectMember,
		factory.KeywordTypeNode(
			tsgo.KeywordTypeSyntaxKindObjectKeyword,
		),
		[]tsgo.Statement{
			hashVariable(
				factory,
				tsgo.NodeFlagsLet,
				"result",
				runtimeCall(factory, objects, "get", value),
			),
			factory.IfStatement(
				factory.BinaryExpression(
					nil,
					result,
					nil,
					factory.BinaryOperatorToken(
						tsgo.BinaryOperatorEqualsEqualsEqualsToken,
					),
					factory.Identifier("undefined"),
				),
				factory.Block(
					[]tsgo.Statement{
						factory.ExpressionStatement(
							factory.BinaryExpression(
								nil,
								result,
								nil,
								factory.BinaryOperatorToken(
									tsgo.BinaryOperatorEqualsToken,
								),
								next,
							),
						),
						factory.ExpressionStatement(
							factory.PostfixUnaryExpression(
								next,
								tsgo.PostfixUnaryExpressionOperatorKindPlusPlusToken,
							),
						),
						factory.ExpressionStatement(
							runtimeCall(
								factory,
								objects,
								"set",
								value,
								result,
							),
						),
					},
					true,
				),
				nil,
			),
			factory.ReturnStatement(result),
		},
	)
}

func hashMixMethod(factory tsgo.Factory) tsgo.MethodDeclaration {
	xor := factory.BinaryExpression(
		nil,
		factory.Identifier("hash"),
		nil,
		factory.BinaryOperatorToken(tsgo.BinaryOperatorCaretToken),
		factory.Identifier("next"),
	)
	product := runtimeCall(
		factory,
		factory.Identifier("Math"),
		"imul",
		xor,
		factory.NumericLiteral("16777619", tsgo.TokenFlagsNone),
	)
	return factory.MethodDeclaration(
		[]tsgo.ModifierLike{factory.StaticKeyword()},
		nil,
		factory.Identifier(HashMixMember),
		nil,
		nil,
		[]tsgo.ParameterDeclaration{
			parameter(
				factory,
				"hash",
				factory.KeywordTypeNode(
					tsgo.KeywordTypeSyntaxKindNumberKeyword,
				),
			),
			parameter(
				factory,
				"next",
				factory.KeywordTypeNode(
					tsgo.KeywordTypeSyntaxKindNumberKeyword,
				),
			),
		},
		factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindNumberKeyword),
		factory.Block(
			[]tsgo.Statement{
				factory.ReturnStatement(unsigned32(factory, product)),
			},
			true,
		),
	)
}

func hashMethod(
	factory tsgo.Factory,
	name string,
	valueType tsgo.TypeNode,
	statements []tsgo.Statement,
) tsgo.MethodDeclaration {
	return factory.MethodDeclaration(
		[]tsgo.ModifierLike{factory.StaticKeyword()},
		nil,
		factory.Identifier(name),
		nil,
		nil,
		[]tsgo.ParameterDeclaration{parameter(factory, "value", valueType)},
		factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindNumberKeyword),
		factory.Block(statements, true),
	)
}

func unsigned32(
	factory tsgo.Factory,
	value tsgo.Expression,
) tsgo.BinaryExpression {
	return factory.BinaryExpression(
		nil,
		value,
		nil,
		factory.BinaryOperatorToken(
			tsgo.BinaryOperatorGreaterThanGreaterThanGreaterThanToken,
		),
		factory.NumericLiteral("0", tsgo.TokenFlagsNone),
	)
}

func runtimeCall(
	factory tsgo.Factory,
	receiver tsgo.Expression,
	name string,
	arguments ...tsgo.Expression,
) tsgo.CallExpression {
	return factory.CallExpression(
		factory.PropertyAccessExpression(
			receiver,
			nil,
			factory.Identifier(name),
			tsgo.NodeFlagsNone,
		),
		nil,
		nil,
		arguments,
		tsgo.NodeFlagsNone,
	)
}

func hashVariable(
	factory tsgo.Factory,
	flags tsgo.NodeFlags,
	name string,
	value tsgo.Expression,
) tsgo.VariableStatement {
	return factory.VariableStatement(
		nil,
		factory.VariableDeclarationList(
			[]tsgo.VariableDeclaration{
				factory.VariableDeclaration(
					factory.Identifier(name),
					nil,
					nil,
					value,
				),
			},
			flags,
		),
	)
}
