package array

import "github.com/tsoniclang/gotots/internal/target/tsgo"

func checkMethod(factory tsgo.Factory, panicName string) tsgo.MethodDeclaration {
	index := factory.Identifier("index")
	invalidNumber := binary(factory,
		binary(factory, factory.TypeOfExpression(index), tsgo.BinaryOperatorEqualsEqualsEqualsToken,
			factory.StringLiteral("number", tsgo.TokenFlagsNone)), tsgo.BinaryOperatorAmpersandAmpersandToken,
		factory.PrefixUnaryExpression(tsgo.PrefixUnaryExpressionOperatorKindExclamationToken,
			call(factory, property(factory, factory.Identifier("Number"), "isInteger"), nil, index)))
	invalid := binary(factory, invalidNumber, tsgo.BinaryOperatorBarBarToken,
		binary(factory, binary(factory, index, tsgo.BinaryOperatorLessThanToken, factory.NumericLiteral("0", tsgo.TokenFlagsNone)),
			tsgo.BinaryOperatorBarBarToken, binary(factory, index, tsgo.BinaryOperatorGreaterThanEqualsToken,
				property(factory, factory.ThisExpression(), "length"))))
	return method(factory, []tsgo.ModifierLike{factory.PrivateKeyword()}, "$check", nil,
		[]tsgo.ParameterDeclaration{parameter(factory, nil, "index", indexType(factory))}, indexType(factory),
		[]tsgo.Statement{factory.IfStatement(invalid, factory.Block([]tsgo.Statement{
			boundsPanic(factory, panicName, "array index out of bounds")}, true), nil), factory.ReturnStatement(index)})
}

func indexType(factory tsgo.Factory) tsgo.UnionTypeNode {
	return factory.UnionTypeNode([]tsgo.TypeNode{
		factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindNumberKeyword),
		factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindBigIntKeyword),
	})
}
