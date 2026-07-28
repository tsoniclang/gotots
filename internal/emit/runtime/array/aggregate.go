package array

import (
	"github.com/tsoniclang/gotots/internal/emit/api"
	arraymember "github.com/tsoniclang/gotots/internal/emit/runtime/array/member"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func BuildAggregateOperation(
	factory tsgo.Factory,
	symbol api.RuntimeSymbol,
) (tsgo.Statement, error) {
	contract, err := api.RuntimeContract(symbol)
	if err != nil {
		return nil, err
	}
	arrayContract, err := api.RuntimeContract(api.RuntimeArray)
	if err != nil {
		return nil, err
	}
	zeroContract, err := api.RuntimeContract(api.RuntimeArrayZeroWith)
	if err != nil {
		return nil, err
	}
	switch symbol {
	case api.RuntimeArrayZeroWith:
		return aggregateZero(
			factory,
			contract.ExportedName(),
			arrayContract.ExportedName(),
		), nil
	case api.RuntimeArrayLiteralWith:
		return aggregateLiteral(
			factory,
			contract.ExportedName(),
			zeroContract.ExportedName(),
			arrayContract.ExportedName(),
		), nil
	case api.RuntimeArrayCopyWith:
		return aggregateCopy(
			factory,
			contract.ExportedName(),
			zeroContract.ExportedName(),
			arrayContract.ExportedName(),
		), nil
	default:
		return nil, &api.RuntimeSymbolError{Symbol: symbol}
	}
}

func aggregateZero(
	factory tsgo.Factory,
	name string,
	arrayName string,
) tsgo.FunctionDeclaration {
	elementType := typeReference(factory, "T")
	lengthType := typeReference(factory, "N")
	length := factory.Identifier("length")
	zero := factory.Identifier("zero")
	result := factory.Identifier("result")
	index := factory.Identifier("index")
	return aggregateFunction(
		factory,
		name,
		[]tsgo.ParameterDeclaration{
			parameter(factory, nil, "length", lengthType),
			parameter(
				factory,
				nil,
				"zero",
				factory.FunctionTypeNode(nil, nil, elementType),
			),
		},
		arrayType(factory, arrayName, elementType, lengthType),
		[]tsgo.Statement{
			variable(
				factory,
				tsgo.NodeFlagsConst,
				"result",
				nil,
				call(
					factory,
					runtimeProperty(
						factory,
						factory.Identifier(arrayName),
						arraymember.Zero,
					),
					[]tsgo.TypeNode{elementType, lengthType},
					length,
					call(factory, zero, nil),
				),
			),
			aggregateLoop(
				factory,
				index,
				length,
				call(
					factory,
					runtimeProperty(factory, result, arraymember.Set),
					nil,
					index,
					call(factory, zero, nil),
				),
			),
			factory.ReturnStatement(result),
		},
	)
}

func aggregateLiteral(
	factory tsgo.Factory,
	name string,
	zeroName string,
	arrayName string,
) tsgo.FunctionDeclaration {
	elementType := typeReference(factory, "T")
	lengthType := typeReference(factory, "N")
	length := factory.Identifier("length")
	zero := factory.Identifier("zero")
	indexes := factory.Identifier("indexes")
	values := factory.Identifier("values")
	result := factory.Identifier("result")
	entry := factory.Identifier("entry")
	return aggregateFunction(
		factory,
		name,
		[]tsgo.ParameterDeclaration{
			parameter(factory, nil, "length", lengthType),
			parameter(
				factory,
				nil,
				"zero",
				factory.FunctionTypeNode(nil, nil, elementType),
			),
			parameter(
				factory,
				nil,
				"indexes",
				factory.ArrayTypeNode(
					factory.KeywordTypeNode(
						tsgo.KeywordTypeSyntaxKindNumberKeyword,
					),
				),
			),
			parameter(
				factory,
				nil,
				"values",
				factory.ArrayTypeNode(elementType),
			),
		},
		arrayType(factory, arrayName, elementType, lengthType),
		[]tsgo.Statement{
			variable(
				factory,
				tsgo.NodeFlagsConst,
				"result",
				nil,
				call(
					factory,
					factory.Identifier(zeroName),
					[]tsgo.TypeNode{elementType, lengthType},
					length,
					zero,
				),
			),
			aggregateLoop(
				factory,
				entry,
				property(factory, indexes, "length"),
				call(
					factory,
					runtimeProperty(factory, result, arraymember.Set),
					nil,
					element(factory, indexes, entry),
					element(factory, values, entry),
				),
			),
			factory.ReturnStatement(result),
		},
	)
}

func aggregateCopy(
	factory tsgo.Factory,
	name string,
	zeroName string,
	arrayName string,
) tsgo.FunctionDeclaration {
	elementType := typeReference(factory, "T")
	lengthType := typeReference(factory, "N")
	source := factory.Identifier("source")
	zero := factory.Identifier("zero")
	copyValue := factory.Identifier("copyValue")
	result := factory.Identifier("result")
	index := factory.Identifier("index")
	return aggregateFunction(
		factory,
		name,
		[]tsgo.ParameterDeclaration{
			parameter(
				factory,
				nil,
				"source",
				arrayType(factory, arrayName, elementType, lengthType),
			),
			parameter(
				factory,
				nil,
				"zero",
				factory.FunctionTypeNode(nil, nil, elementType),
			),
			parameter(
				factory,
				nil,
				"copyValue",
				factory.FunctionTypeNode(
					nil,
					[]tsgo.ParameterDeclaration{
						parameter(factory, nil, "value", elementType),
					},
					elementType,
				),
			),
		},
		arrayType(factory, arrayName, elementType, lengthType),
		[]tsgo.Statement{
			variable(
				factory,
				tsgo.NodeFlagsConst,
				"result",
				nil,
				call(
					factory,
					factory.Identifier(zeroName),
					[]tsgo.TypeNode{elementType, lengthType},
					runtimeProperty(factory, source, arraymember.Length),
					zero,
				),
			),
			aggregateLoop(
				factory,
				index,
				runtimeProperty(factory, source, arraymember.Length),
				call(
					factory,
					runtimeProperty(factory, result, arraymember.Set),
					nil,
					index,
					call(
						factory,
						copyValue,
						nil,
						call(
							factory,
							runtimeProperty(
								factory,
								source,
								arraymember.Get,
							),
							nil,
							index,
						),
					),
				),
			),
			factory.ReturnStatement(result),
		},
	)
}

func aggregateFunction(
	factory tsgo.Factory,
	name string,
	parameters []tsgo.ParameterDeclaration,
	result tsgo.TypeNode,
	body []tsgo.Statement,
) tsgo.FunctionDeclaration {
	return factory.FunctionDeclaration(
		[]tsgo.ModifierLike{factory.ExportKeyword()},
		nil,
		factory.Identifier(name),
		typeParameters(factory),
		parameters,
		result,
		factory.Block(body, true),
	)
}

func aggregateLoop(
	factory tsgo.Factory,
	index tsgo.Identifier,
	length tsgo.Expression,
	body tsgo.Expression,
) tsgo.ForStatement {
	return factory.ForStatement(
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
		binary(
			factory,
			index,
			tsgo.BinaryOperatorLessThanToken,
			length,
		),
		factory.PostfixUnaryExpression(
			index,
			tsgo.PostfixUnaryExpressionOperatorKindPlusPlusToken,
		),
		factory.Block(
			[]tsgo.Statement{factory.ExpressionStatement(body)},
			true,
		),
	)
}
