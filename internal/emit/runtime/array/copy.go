package array

import (
	"github.com/tsoniclang/gotots/internal/emit/runtime/memoryview"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func copyBody(factory tsgo.Factory, exportedName string, elementType, lengthType tsgo.TypeNode) []tsgo.Statement {
	values := factory.Identifier("values")
	result := func(backing tsgo.Expression) tsgo.Statement {
		return factory.ReturnStatement(factory.NewExpression(factory.Identifier(exportedName), []tsgo.TypeNode{elementType, lengthType},
			[]tsgo.Expression{memoryview.Indexed(factory, elementType, backing, factory.NumericLiteral("0", tsgo.TokenFlagsNone)), property(factory, factory.ThisExpression(), "length")}))
	}
	native := call(factory, property(factory, factory.Identifier("Array"), "from"), nil,
		property(factory, regionValue(factory), memoryview.ValuesMember))
	body := []tsgo.Statement{factory.IfStatement(wholeBacking(factory), factory.Block([]tsgo.Statement{result(native)}, true), nil)}
	body = append(body, copyWindow(factory, elementType)...)
	return append(body, result(values))
}

func wholeBacking(factory tsgo.Factory) tsgo.Expression {
	return binary(factory, binary(factory, property(factory, regionValue(factory), memoryview.KindMember),
		tsgo.BinaryOperatorEqualsEqualsEqualsToken, factory.StringLiteral(memoryview.IndexedKind, tsgo.TokenFlagsNone)),
		tsgo.BinaryOperatorAmpersandAmpersandToken, binary(factory,
			binary(factory, property(factory, regionValue(factory), memoryview.OffsetMember), tsgo.BinaryOperatorEqualsEqualsEqualsToken,
				factory.NumericLiteral("0", tsgo.TokenFlagsNone)),
			tsgo.BinaryOperatorAmpersandAmpersandToken,
			binary(factory, call(factory, factory.Identifier("BigInt"), nil, property(factory, property(factory, regionValue(factory), memoryview.ValuesMember), "length")),
				tsgo.BinaryOperatorEqualsEqualsEqualsToken, call(factory, factory.Identifier("BigInt"), nil, property(factory, factory.ThisExpression(), "length")))))
}

func copyWindow(factory tsgo.Factory, elementType tsgo.TypeNode) []tsgo.Statement {
	values := factory.Identifier("values")
	index := factory.Identifier("index")
	length := property(factory, factory.ThisExpression(), "length")
	return []tsgo.Statement{
		variable(factory, tsgo.NodeFlagsConst, "values", factory.ArrayTypeNode(elementType),
			factory.ArrayLiteralExpression(nil, false)),
		factory.ForStatement(factory.VariableDeclarationList([]tsgo.VariableDeclaration{
			factory.VariableDeclaration(index, nil, nil, factory.NumericLiteral("0", tsgo.TokenFlagsNone)),
		}, tsgo.NodeFlagsLet), binary(factory, index, tsgo.BinaryOperatorLessThanToken, length),
			factory.PostfixUnaryExpression(index, tsgo.PostfixUnaryExpressionOperatorKindPlusPlusToken),
			factory.Block([]tsgo.Statement{factory.ExpressionStatement(call(factory, property(factory, values, "push"), nil,
				call(factory, property(factory, factory.ThisExpression(), "get"), nil, index)))}, true)),
	}
}
