package unsafeoperation

import (
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type builder struct {
	factory   tsgo.Factory
	sliceName string
	panicName string
}

func Build(factory tsgo.Factory, symbol api.RuntimeSymbol) (tsgo.Statement, error) {
	if symbol != api.RuntimeUnsafeString {
		return nil, &api.RuntimeSymbolError{Symbol: symbol}
	}
	slice, err := api.RuntimeContract(api.RuntimeSlice)
	if err != nil {
		return nil, err
	}
	contract, err := api.RuntimeContract(symbol)
	if err != nil {
		return nil, err
	}
	panicContract, err := api.RuntimeContract(api.RuntimePanic)
	if err != nil {
		return nil, err
	}
	b := builder{
		factory:   factory,
		sliceName: slice.ExportedName(),
		panicName: panicContract.ExportedName(),
	}
	return b.stringFunction(contract.ExportedName()), nil
}

func (b builder) id(name string) tsgo.Identifier {
	return b.factory.Identifier(name)
}

func (b builder) typeReference(name string, arguments ...tsgo.TypeNode) tsgo.TypeReferenceNode {
	return b.factory.TypeReferenceNode(b.id(name), arguments)
}

func (b builder) typeI() tsgo.TypeNode { return b.typeReference("I") }

func (b builder) numberType() tsgo.TypeNode {
	return b.factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindNumberKeyword)
}

func (b builder) bigintType() tsgo.TypeNode {
	return b.factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindBigIntKeyword)
}

func (b builder) integerType() tsgo.TypeNode {
	return b.factory.UnionTypeNode([]tsgo.TypeNode{b.numberType(), b.bigintType()})
}

func (b builder) stringType() tsgo.TypeNode {
	return b.factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindStringKeyword)
}

func (b builder) sliceType(storage tsgo.TypeNode) tsgo.TypeNode {
	return b.typeReference(b.sliceName, storage)
}

func (b builder) typeParameter(name string, constraint tsgo.TypeNode) tsgo.TypeParameterDeclaration {
	return b.factory.TypeParameterDeclaration(nil, b.id(name), constraint, nil, nil)
}

func (b builder) parameter(name string, target tsgo.TypeNode) tsgo.ParameterDeclaration {
	return b.factory.ParameterDeclaration(nil, nil, b.id(name), nil, target, nil)
}

func (b builder) property(receiver tsgo.Expression, name string) tsgo.PropertyAccessExpression {
	return b.factory.PropertyAccessExpression(receiver, nil, b.id(name), tsgo.NodeFlagsNone)
}

func (b builder) call(
	receiver tsgo.Expression,
	name string,
	typeArguments []tsgo.TypeNode,
	arguments ...tsgo.Expression,
) tsgo.CallExpression {
	return b.factory.CallExpression(
		b.property(receiver, name),
		nil,
		typeArguments,
		arguments,
		tsgo.NodeFlagsNone,
	)
}

func (b builder) globalCall(
	name string,
	arguments ...tsgo.Expression,
) tsgo.CallExpression {
	return b.factory.CallExpression(
		b.property(b.id("globalThis"), name),
		nil,
		nil,
		arguments,
		tsgo.NodeFlagsNone,
	)
}

func (b builder) binary(left tsgo.Expression, operator tsgo.BinaryOperator, right tsgo.Expression) tsgo.BinaryExpression {
	return b.factory.BinaryExpression(nil, left, nil, b.factory.BinaryOperatorToken(operator), right)
}

func (b builder) variable(flags tsgo.NodeFlags, name string, target tsgo.TypeNode, value tsgo.Expression) tsgo.VariableStatement {
	return b.factory.VariableStatement(nil, b.factory.VariableDeclarationList(
		[]tsgo.VariableDeclaration{b.factory.VariableDeclaration(b.id(name), nil, target, value)},
		flags,
	))
}

func (b builder) number(value string) tsgo.NumericLiteral {
	return b.factory.NumericLiteral(value, tsgo.TokenFlagsNone)
}

func (b builder) loop(limit tsgo.Expression, statements ...tsgo.Statement) tsgo.ForStatement {
	index := b.id("index")
	return b.factory.ForStatement(
		b.factory.VariableDeclarationList(
			[]tsgo.VariableDeclaration{b.factory.VariableDeclaration(index, nil, nil, b.number("0"))},
			tsgo.NodeFlagsLet,
		),
		b.binary(index, tsgo.BinaryOperatorLessThanToken, limit),
		b.factory.PostfixUnaryExpression(index, tsgo.PostfixUnaryExpressionOperatorKindPlusPlusToken),
		b.factory.Block(statements, true),
	)
}
