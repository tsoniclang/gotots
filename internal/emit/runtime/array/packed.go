package array

import (
	"github.com/tsoniclang/gotots/internal/emit/api"
	arraymember "github.com/tsoniclang/gotots/internal/emit/runtime/array/member"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func buildPackedOperation(factory tsgo.Factory) (tsgo.Statement, error) {
	contract, err := api.RuntimeContract(api.RuntimeArrayPacked)
	if err != nil {
		return nil, err
	}
	arrayContract, err := api.RuntimeContract(api.RuntimeArray)
	if err != nil {
		return nil, err
	}
	panicContract, err := api.RuntimeContract(api.RuntimePanic)
	if err != nil {
		return nil, err
	}
	arrayName := arrayContract.ExportedName()
	panicName := panicContract.ExportedName()
	elementType := typeReference(factory, "T")
	lengthType := typeReference(factory, "N")
	length := factory.Identifier("length")
	zero := factory.Identifier("zero")
	entryCount := factory.Identifier("entryCount")
	encoded := factory.Identifier("encoded")
	fields := factory.Identifier("fields")
	entry := factory.Identifier("entry")
	index := factory.Identifier("index")
	indexText := factory.Identifier("indexText")
	value := factory.Identifier("value")
	valueText := factory.Identifier("valueText")
	result := factory.Identifier("result")
	number := api.TargetIntrinsicNumber.Expression(factory)
	stringType := factory.KeywordTypeNode(
		tsgo.KeywordTypeSyntaxKindStringKeyword,
	)
	indexFields := packedFieldVariables(
		factory,
		panicName,
		"index",
		fields,
		entry,
		stringType,
		number,
	)
	valueFields := packedFieldVariables(
		factory,
		panicName,
		"value",
		fields,
		binary(
			factory,
			entry,
			tsgo.BinaryOperatorPlusToken,
			factory.NumericLiteral("1", tsgo.TokenFlagsNone),
		),
		stringType,
		number,
	)
	loopBody := append(indexFields, valueFields...)
	loopBody = append(loopBody,
		factory.IfStatement(
			packedFieldInvalid(
				factory,
				number,
				indexText,
				index,
				valueText,
				value,
			),
			factory.Block([]tsgo.Statement{boundsPanic(
				factory,
				panicName,
				"array packed payload is malformed",
			)}, true),
			nil,
		),
		factory.ExpressionStatement(call(
			factory,
			runtimeProperty(factory, result, arraymember.Set),
			nil,
			index,
			factory.AsExpression(value, elementType),
		)),
	)
	body := []tsgo.Statement{
		variable(
			factory,
			tsgo.NodeFlagsConst,
			"fields",
			nil,
			call(
				factory,
				property(factory, encoded, "split"),
				nil,
				factory.StringLiteral(",", tsgo.TokenFlagsNone),
			),
		),
		factory.IfStatement(
			binary(
				factory,
				property(factory, fields, "length"),
				tsgo.BinaryOperatorExclamationEqualsEqualsToken,
				binary(
					factory,
					entryCount,
					tsgo.BinaryOperatorAsteriskToken,
					factory.NumericLiteral("2", tsgo.TokenFlagsNone),
				),
			),
			factory.Block([]tsgo.Statement{boundsPanic(
				factory,
				panicName,
				"array packed payload length mismatch",
			)}, true),
			nil,
		),
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
				zero,
			),
		),
		factory.ForStatement(
			factory.VariableDeclarationList(
				[]tsgo.VariableDeclaration{factory.VariableDeclaration(
					entry,
					nil,
					nil,
					factory.NumericLiteral("0", tsgo.TokenFlagsNone),
				)},
				tsgo.NodeFlagsLet,
			),
			binary(
				factory,
				entry,
				tsgo.BinaryOperatorLessThanToken,
				property(factory, fields, "length"),
			),
			binary(
				factory,
				entry,
				tsgo.BinaryOperatorPlusEqualsToken,
				factory.NumericLiteral("2", tsgo.TokenFlagsNone),
			),
			factory.Block(loopBody, true),
		),
		factory.ReturnStatement(result),
	}
	return factory.FunctionDeclaration(
		[]tsgo.ModifierLike{factory.ExportKeyword()},
		nil,
		factory.Identifier(contract.ExportedName()),
		packedTypeParameters(factory),
		[]tsgo.ParameterDeclaration{
			parameter(factory, nil, "length", lengthType),
			parameter(factory, nil, "zero", elementType),
			parameter(
				factory,
				nil,
				"entryCount",
				factory.KeywordTypeNode(
					tsgo.KeywordTypeSyntaxKindNumberKeyword,
				),
			),
			parameter(factory, nil, "encoded", stringType),
		},
		arrayType(factory, arrayName, elementType, lengthType),
		factory.Block(body, true),
	), nil
}

func packedTypeParameters(
	factory tsgo.Factory,
) []tsgo.TypeParameterDeclaration {
	return []tsgo.TypeParameterDeclaration{
		factory.TypeParameterDeclaration(
			nil,
			factory.Identifier("T"),
			factory.KeywordTypeNode(
				tsgo.KeywordTypeSyntaxKindNumberKeyword,
			),
			nil,
			nil,
		),
		factory.TypeParameterDeclaration(
			nil,
			factory.Identifier("N"),
			factory.KeywordTypeNode(
				tsgo.KeywordTypeSyntaxKindNumberKeyword,
			),
			nil,
			nil,
		),
	}
}

func packedFieldVariables(
	factory tsgo.Factory,
	panicName string,
	name string,
	fields tsgo.Expression,
	entry tsgo.Expression,
	stringType tsgo.TypeNode,
	number tsgo.Expression,
) []tsgo.Statement {
	textName := name + "Text"
	return []tsgo.Statement{
		variable(
			factory,
			tsgo.NodeFlagsConst,
			textName,
			stringType,
			definedElement(
				factory,
				panicName,
				fields,
				entry,
				stringType,
			),
		),
		variable(
			factory,
			tsgo.NodeFlagsConst,
			name,
			factory.KeywordTypeNode(
				tsgo.KeywordTypeSyntaxKindNumberKeyword,
			),
			call(
				factory,
				property(factory, number, "parseInt"),
				nil,
				factory.Identifier(textName),
				factory.NumericLiteral("36", tsgo.TokenFlagsNone),
			),
		),
	}
}

func packedFieldInvalid(
	factory tsgo.Factory,
	number tsgo.Expression,
	indexText tsgo.Expression,
	index tsgo.Expression,
	valueText tsgo.Expression,
	value tsgo.Expression,
) tsgo.Expression {
	invalid := func(text tsgo.Expression, parsed tsgo.Expression) tsgo.Expression {
		return binary(
			factory,
			factory.PrefixUnaryExpression(
				tsgo.PrefixUnaryExpressionOperatorKindExclamationToken,
				call(
					factory,
					property(factory, number, "isSafeInteger"),
					nil,
					parsed,
				),
			),
			tsgo.BinaryOperatorBarBarToken,
			binary(
				factory,
				call(
					factory,
					property(factory, parsed, "toString"),
					nil,
					factory.NumericLiteral("36", tsgo.TokenFlagsNone),
				),
				tsgo.BinaryOperatorExclamationEqualsEqualsToken,
				text,
			),
		)
	}
	return binary(
		factory,
		invalid(indexText, index),
		tsgo.BinaryOperatorBarBarToken,
		invalid(valueText, value),
	)
}
