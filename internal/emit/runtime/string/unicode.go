package stringruntime

import "github.com/tsoniclang/gotots/internal/target/tsgo"

func encodeRune(
	factory tsgo.Factory,
	exportedName string,
) tsgo.FunctionDeclaration {
	input := factory.Identifier("input")
	runeValue := factory.Identifier("runeValue")
	invalid := or(
		factory,
		notInteger(factory, runeValue),
		or(
			factory,
			lessThan(factory, runeValue, numeric(factory, "0")),
			or(
				factory,
				greaterThan(factory, runeValue, numeric(factory, "1114111")),
				and(
					factory,
					greaterThanOrEqual(
						factory,
						runeValue,
						numeric(factory, "55296"),
					),
					lessThanOrEqual(
						factory,
						runeValue,
						numeric(factory, "57343"),
					),
				),
			),
		),
	)
	return factory.FunctionDeclaration(
		[]tsgo.ModifierLike{factory.ExportKeyword()},
		nil,
		factory.Identifier(exportedName),
		nil,
		[]tsgo.ParameterDeclaration{
			parameter(factory, input, indexType(factory), false),
		},
		stringType(factory),
		factory.Block([]tsgo.Statement{
			variable(
				factory,
				tsgo.NodeFlagsLet,
				runeValue,
				callIdentifier(factory, "Number", input),
			),
			factory.IfStatement(
				invalid,
				factory.Block([]tsgo.Statement{
					assignStatement(
						factory,
						runeValue,
						numeric(factory, "65533"),
					),
				}, true),
				nil,
			),
			factory.IfStatement(
				lessThanOrEqual(
					factory,
					runeValue,
					numeric(factory, "127"),
				),
				factory.Block([]tsgo.Statement{
					factory.ReturnStatement(fromCharCode(factory, runeValue)),
				}, true),
				nil,
			),
			factory.IfStatement(
				lessThanOrEqual(
					factory,
					runeValue,
					numeric(factory, "2047"),
				),
				factory.Block([]tsgo.Statement{
					factory.ReturnStatement(fromCharCode(
						factory,
						binary(
							factory,
							numeric(factory, "192"),
							tsgo.BinaryOperatorBarToken,
							binary(
								factory,
								runeValue,
								tsgo.BinaryOperatorGreaterThanGreaterThanToken,
								numeric(factory, "6"),
							),
						),
						continuationByte(factory, runeValue),
					)),
				}, true),
				nil,
			),
			factory.IfStatement(
				lessThanOrEqual(
					factory,
					runeValue,
					numeric(factory, "65535"),
				),
				factory.Block([]tsgo.Statement{
					factory.ReturnStatement(fromCharCode(
						factory,
						binary(
							factory,
							numeric(factory, "224"),
							tsgo.BinaryOperatorBarToken,
							binary(
								factory,
								runeValue,
								tsgo.BinaryOperatorGreaterThanGreaterThanToken,
								numeric(factory, "12"),
							),
						),
						continuationByte(
							factory,
							binary(
								factory,
								runeValue,
								tsgo.BinaryOperatorGreaterThanGreaterThanToken,
								numeric(factory, "6"),
							),
						),
						continuationByte(factory, runeValue),
					)),
				}, true),
				nil,
			),
			factory.ReturnStatement(fromCharCode(
				factory,
				binary(
					factory,
					numeric(factory, "240"),
					tsgo.BinaryOperatorBarToken,
					binary(
						factory,
						runeValue,
						tsgo.BinaryOperatorGreaterThanGreaterThanToken,
						numeric(factory, "18"),
					),
				),
				continuationByte(
					factory,
					binary(
						factory,
						runeValue,
						tsgo.BinaryOperatorGreaterThanGreaterThanToken,
						numeric(factory, "12"),
					),
				),
				continuationByte(
					factory,
					binary(
						factory,
						runeValue,
						tsgo.BinaryOperatorGreaterThanGreaterThanToken,
						numeric(factory, "6"),
					),
				),
				continuationByte(factory, runeValue),
			)),
		}, true),
	)
}

func decodeRune(
	factory tsgo.Factory,
	exportedName string,
) tsgo.FunctionDeclaration {
	value := factory.Identifier("value")
	input := factory.Identifier("input")
	index := factory.Identifier("index")
	first := factory.Identifier("first")
	second := factory.Identifier("second")
	third := factory.Identifier("third")
	fourth := factory.Identifier("fourth")
	width := factory.Identifier("width")
	result := factory.Identifier("result")
	invalid := runeTuple(factory, "65533", "1")
	return factory.FunctionDeclaration(
		[]tsgo.ModifierLike{factory.ExportKeyword()},
		nil,
		factory.Identifier(exportedName),
		nil,
		[]tsgo.ParameterDeclaration{
			parameter(factory, value, stringType(factory), false),
			parameter(factory, input, indexType(factory), false),
		},
		factory.TupleTypeNode([]tsgo.TypeNode{
			numberType(factory),
			numberType(factory),
		}),
		factory.Block([]tsgo.Statement{
			variable(
				factory,
				tsgo.NodeFlagsConst,
				index,
				callIdentifier(factory, "Number", input),
			),
			variable(
				factory,
				tsgo.NodeFlagsConst,
				first,
				charCodeAt(factory, value, index),
			),
			factory.IfStatement(
				lessThan(factory, first, numeric(factory, "128")),
				factory.Block([]tsgo.Statement{
					factory.ReturnStatement(
						factory.ArrayLiteralExpression(
							[]tsgo.Expression{first, numeric(factory, "1")},
							false,
						),
					),
				}, true),
				nil,
			),
			factory.IfStatement(
				or(
					factory,
					lessThan(factory, first, numeric(factory, "194")),
					greaterThan(factory, first, numeric(factory, "244")),
				),
				factory.Block(
					[]tsgo.Statement{factory.ReturnStatement(invalid)},
					true,
				),
				nil,
			),
			variable(
				factory,
				tsgo.NodeFlagsConst,
				width,
				factory.ConditionalExpression(
					lessThan(factory, first, numeric(factory, "224")),
					factory.QuestionToken(),
					numeric(factory, "2"),
					factory.ColonToken(),
					factory.ConditionalExpression(
						lessThan(factory, first, numeric(factory, "240")),
						factory.QuestionToken(),
						numeric(factory, "3"),
						factory.ColonToken(),
						numeric(factory, "4"),
					),
				),
			),
			factory.IfStatement(
				greaterThan(
					factory,
					binary(
						factory,
						index,
						tsgo.BinaryOperatorPlusToken,
						width,
					),
					length(factory, value),
				),
				factory.Block(
					[]tsgo.Statement{factory.ReturnStatement(invalid)},
					true,
				),
				nil,
			),
			variable(
				factory,
				tsgo.NodeFlagsConst,
				second,
				charCodeAt(
					factory,
					value,
					binary(
						factory,
						index,
						tsgo.BinaryOperatorPlusToken,
						numeric(factory, "1"),
					),
				),
			),
			factory.IfStatement(
				invalidContinuation(factory, second),
				factory.Block(
					[]tsgo.Statement{factory.ReturnStatement(invalid)},
					true,
				),
				nil,
			),
			factory.IfStatement(
				or(
					factory,
					and(
						factory,
						binary(
							factory,
							first,
							tsgo.BinaryOperatorEqualsEqualsEqualsToken,
							numeric(factory, "224"),
						),
						lessThan(factory, second, numeric(factory, "160")),
					),
					or(
						factory,
						and(
							factory,
							binary(
								factory,
								first,
								tsgo.BinaryOperatorEqualsEqualsEqualsToken,
								numeric(factory, "237"),
							),
							greaterThan(factory, second, numeric(factory, "159")),
						),
						or(
							factory,
							and(
								factory,
								binary(
									factory,
									first,
									tsgo.BinaryOperatorEqualsEqualsEqualsToken,
									numeric(factory, "240"),
								),
								lessThan(factory, second, numeric(factory, "144")),
							),
							and(
								factory,
								binary(
									factory,
									first,
									tsgo.BinaryOperatorEqualsEqualsEqualsToken,
									numeric(factory, "244"),
								),
								greaterThan(factory, second, numeric(factory, "143")),
							),
						),
					),
				),
				factory.Block(
					[]tsgo.Statement{factory.ReturnStatement(invalid)},
					true,
				),
				nil,
			),
			variable(
				factory,
				tsgo.NodeFlagsLet,
				result,
				binary(
					factory,
					binary(
						factory,
						first,
						tsgo.BinaryOperatorAmpersandToken,
						firstMask(factory, width),
					),
					tsgo.BinaryOperatorLessThanLessThanToken,
					numeric(factory, "6"),
				),
			),
			assignStatement(
				factory,
				result,
				binary(
					factory,
					result,
					tsgo.BinaryOperatorBarToken,
					binary(
						factory,
						second,
						tsgo.BinaryOperatorAmpersandToken,
						numeric(factory, "63"),
					),
				),
			),
			factory.IfStatement(
				binary(
					factory,
					width,
					tsgo.BinaryOperatorEqualsEqualsEqualsToken,
					numeric(factory, "2"),
				),
				factory.Block([]tsgo.Statement{
					factory.ReturnStatement(
						factory.ArrayLiteralExpression(
							[]tsgo.Expression{result, width},
							false,
						),
					),
				}, true),
				nil,
			),
			variable(
				factory,
				tsgo.NodeFlagsConst,
				third,
				charCodeAt(
					factory,
					value,
					binary(
						factory,
						index,
						tsgo.BinaryOperatorPlusToken,
						numeric(factory, "2"),
					),
				),
			),
			factory.IfStatement(
				invalidContinuation(factory, third),
				factory.Block(
					[]tsgo.Statement{factory.ReturnStatement(invalid)},
					true,
				),
				nil,
			),
			assignStatement(
				factory,
				result,
				binary(
					factory,
					binary(
						factory,
						result,
						tsgo.BinaryOperatorLessThanLessThanToken,
						numeric(factory, "6"),
					),
					tsgo.BinaryOperatorBarToken,
					binary(
						factory,
						third,
						tsgo.BinaryOperatorAmpersandToken,
						numeric(factory, "63"),
					),
				),
			),
			factory.IfStatement(
				binary(
					factory,
					width,
					tsgo.BinaryOperatorEqualsEqualsEqualsToken,
					numeric(factory, "3"),
				),
				factory.Block([]tsgo.Statement{
					factory.ReturnStatement(
						factory.ArrayLiteralExpression(
							[]tsgo.Expression{result, width},
							false,
						),
					),
				}, true),
				nil,
			),
			variable(
				factory,
				tsgo.NodeFlagsConst,
				fourth,
				charCodeAt(
					factory,
					value,
					binary(
						factory,
						index,
						tsgo.BinaryOperatorPlusToken,
						numeric(factory, "3"),
					),
				),
			),
			factory.IfStatement(
				invalidContinuation(factory, fourth),
				factory.Block(
					[]tsgo.Statement{factory.ReturnStatement(invalid)},
					true,
				),
				nil,
			),
			assignStatement(
				factory,
				result,
				binary(
					factory,
					binary(
						factory,
						result,
						tsgo.BinaryOperatorLessThanLessThanToken,
						numeric(factory, "6"),
					),
					tsgo.BinaryOperatorBarToken,
					binary(
						factory,
						fourth,
						tsgo.BinaryOperatorAmpersandToken,
						numeric(factory, "63"),
					),
				),
			),
			factory.ReturnStatement(
				factory.ArrayLiteralExpression(
					[]tsgo.Expression{result, width},
					false,
				),
			),
		}, true),
	)
}
