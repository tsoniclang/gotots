package array

import "github.com/tsoniclang/gotots/internal/target/tsgo"

const className = "GoArray"

func Build(factory tsgo.Factory) tsgo.ClassDeclaration {
	elementType := typeReference(factory, "T")
	lengthType := typeReference(factory, "N")
	typeParameters := []tsgo.TypeParameterDeclaration{
		factory.TypeParameterDeclaration(
			nil,
			factory.Identifier("T"),
			nil,
			nil,
			nil,
		),
		factory.TypeParameterDeclaration(
			nil,
			factory.Identifier("N"),
			factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindNumberKeyword),
			nil,
			nil,
		),
	}
	members := []tsgo.ClassElement{
		constructor(factory, elementType, lengthType),
		zeroMethod(factory),
		literalMethod(factory),
		copyMethod(factory, elementType, lengthType),
		equalMethod(factory, elementType, lengthType),
		getMethod(factory, elementType),
		setMethod(factory, elementType),
		checkMethod(factory),
	}
	return factory.ClassDeclaration(
		[]tsgo.ModifierLike{factory.ExportKeyword()},
		factory.Identifier(className),
		typeParameters,
		nil,
		members,
	)
}

func constructor(
	factory tsgo.Factory,
	elementType tsgo.TypeNode,
	lengthType tsgo.TypeNode,
) tsgo.ConstructorDeclaration {
	return factory.ConstructorDeclaration(
		[]tsgo.ModifierLike{factory.PrivateKeyword()},
		nil,
		[]tsgo.ParameterDeclaration{
			parameter(
				factory,
				[]tsgo.ModifierLike{
					factory.PrivateKeyword(),
					factory.ReadonlyKeyword(),
				},
				"$values",
				factory.ArrayTypeNode(elementType),
			),
			parameter(
				factory,
				[]tsgo.ModifierLike{
					factory.PublicKeyword(),
					factory.ReadonlyKeyword(),
				},
				"length",
				lengthType,
			),
		},
		nil,
		factory.Block(nil, true),
	)
}

func zeroMethod(factory tsgo.Factory) tsgo.MethodDeclaration {
	elementType := typeReference(factory, "T")
	lengthType := typeReference(factory, "N")
	values := factory.Identifier("values")
	index := factory.Identifier("index")
	length := factory.Identifier("length")
	body := []tsgo.Statement{
		variable(
			factory,
			tsgo.NodeFlagsConst,
			"values",
			factory.ArrayTypeNode(elementType),
			factory.ArrayLiteralExpression(nil, false),
		),
		factory.ForStatement(
			factory.VariableDeclarationList(
				[]tsgo.VariableDeclaration{factory.VariableDeclaration(
					index,
					nil,
					nil,
					factory.NumericLiteral("0", tsgo.TokenFlagsNone),
				)},
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
			factory.Block([]tsgo.Statement{
				factory.ExpressionStatement(call(
					factory,
					property(factory, values, "push"),
					nil,
					factory.Identifier("zero"),
				)),
			}, true),
		),
		factory.ReturnStatement(factory.NewExpression(
			factory.Identifier(className),
			[]tsgo.TypeNode{elementType, lengthType},
			[]tsgo.Expression{values, length},
		)),
	}
	return method(
		factory,
		[]tsgo.ModifierLike{
			factory.PublicKeyword(),
			factory.StaticKeyword(),
		},
		"zero",
		typeParameters(factory),
		[]tsgo.ParameterDeclaration{
			parameter(factory, nil, "length", lengthType),
			parameter(factory, nil, "zero", elementType),
		},
		arrayType(factory, elementType, lengthType),
		body,
	)
}

func literalMethod(factory tsgo.Factory) tsgo.MethodDeclaration {
	elementType := typeReference(factory, "T")
	lengthType := typeReference(factory, "N")
	indexes := factory.Identifier("indexes")
	values := factory.Identifier("values")
	entry := factory.Identifier("entry")
	result := factory.Identifier("result")
	body := []tsgo.Statement{
		factory.IfStatement(
			binary(
				factory,
				property(factory, indexes, "length"),
				tsgo.BinaryOperatorExclamationEqualsEqualsToken,
				property(factory, values, "length"),
			),
			factory.Block([]tsgo.Statement{rangeError(
				factory,
				"array literal index/value length mismatch",
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
				property(factory, factory.Identifier(className), "zero"),
				[]tsgo.TypeNode{elementType, lengthType},
				factory.Identifier("length"),
				factory.Identifier("zero"),
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
				property(factory, indexes, "length"),
			),
			factory.PostfixUnaryExpression(
				entry,
				tsgo.PostfixUnaryExpressionOperatorKindPlusPlusToken,
			),
			factory.Block([]tsgo.Statement{
				factory.ExpressionStatement(call(
					factory,
					property(factory, result, "set"),
					nil,
					element(factory, indexes, entry),
					element(factory, values, entry),
				)),
			}, true),
		),
		factory.ReturnStatement(result),
	}
	return method(
		factory,
		[]tsgo.ModifierLike{
			factory.PublicKeyword(),
			factory.StaticKeyword(),
		},
		"literal",
		typeParameters(factory),
		[]tsgo.ParameterDeclaration{
			parameter(factory, nil, "length", lengthType),
			parameter(factory, nil, "zero", elementType),
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
		arrayType(factory, elementType, lengthType),
		body,
	)
}

func copyMethod(
	factory tsgo.Factory,
	elementType tsgo.TypeNode,
	lengthType tsgo.TypeNode,
) tsgo.MethodDeclaration {
	return method(
		factory,
		[]tsgo.ModifierLike{factory.PublicKeyword()},
		"copy",
		nil,
		nil,
		arrayType(factory, elementType, lengthType),
		[]tsgo.Statement{factory.ReturnStatement(factory.NewExpression(
			factory.Identifier(className),
			[]tsgo.TypeNode{elementType, lengthType},
			[]tsgo.Expression{
				call(
					factory,
					property(
						factory,
						property(factory, factory.ThisExpression(), "$values"),
						"slice",
					),
					nil,
				),
				property(factory, factory.ThisExpression(), "length"),
			},
		))},
	)
}

func equalMethod(
	factory tsgo.Factory,
	elementType tsgo.TypeNode,
	lengthType tsgo.TypeNode,
) tsgo.MethodDeclaration {
	index := factory.Identifier("index")
	left := element(
		factory,
		property(factory, factory.ThisExpression(), "$values"),
		index,
	)
	right := element(
		factory,
		property(factory, factory.Identifier("other"), "$values"),
		index,
	)
	return method(
		factory,
		[]tsgo.ModifierLike{factory.PublicKeyword()},
		"equal",
		nil,
		[]tsgo.ParameterDeclaration{parameter(
			factory,
			nil,
			"other",
			arrayType(factory, elementType, lengthType),
		)},
		factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindBooleanKeyword),
		[]tsgo.Statement{
			factory.ForStatement(
				factory.VariableDeclarationList(
					[]tsgo.VariableDeclaration{factory.VariableDeclaration(
						index,
						nil,
						nil,
						factory.NumericLiteral("0", tsgo.TokenFlagsNone),
					)},
					tsgo.NodeFlagsLet,
				),
				binary(
					factory,
					index,
					tsgo.BinaryOperatorLessThanToken,
					property(factory, factory.ThisExpression(), "length"),
				),
				factory.PostfixUnaryExpression(
					index,
					tsgo.PostfixUnaryExpressionOperatorKindPlusPlusToken,
				),
				factory.Block([]tsgo.Statement{
					factory.IfStatement(
						binary(
							factory,
							left,
							tsgo.BinaryOperatorExclamationEqualsEqualsToken,
							right,
						),
						factory.Block([]tsgo.Statement{
							factory.ReturnStatement(factory.FalseLiteral()),
						}, true),
						nil,
					),
				}, true),
			),
			factory.ReturnStatement(factory.TrueLiteral()),
		},
	)
}

func getMethod(
	factory tsgo.Factory,
	elementType tsgo.TypeNode,
) tsgo.MethodDeclaration {
	index := factory.Identifier("index")
	return method(
		factory,
		[]tsgo.ModifierLike{factory.PublicKeyword()},
		"get",
		nil,
		[]tsgo.ParameterDeclaration{parameter(
			factory,
			nil,
			"index",
			indexType(factory),
		)},
		elementType,
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
					property(factory, factory.ThisExpression(), "$check"),
					nil,
					index,
				),
			),
			factory.ReturnStatement(element(
				factory,
				property(factory, factory.ThisExpression(), "$values"),
				factory.Identifier("offset"),
			)),
		},
	)
}

func setMethod(
	factory tsgo.Factory,
	elementType tsgo.TypeNode,
) tsgo.MethodDeclaration {
	index := factory.Identifier("index")
	return method(
		factory,
		[]tsgo.ModifierLike{factory.PublicKeyword()},
		"set",
		nil,
		[]tsgo.ParameterDeclaration{
			parameter(
				factory,
				nil,
				"index",
				indexType(factory),
			),
			parameter(factory, nil, "value", elementType),
		},
		factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindVoidKeyword),
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
					property(factory, factory.ThisExpression(), "$check"),
					nil,
					index,
				),
			),
			factory.ExpressionStatement(binary(
				factory,
				element(
					factory,
					property(factory, factory.ThisExpression(), "$values"),
					factory.Identifier("offset"),
				),
				tsgo.BinaryOperatorEqualsToken,
				factory.Identifier("value"),
			)),
		},
	)
}

func checkMethod(factory tsgo.Factory) tsgo.MethodDeclaration {
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
		property(factory, factory.ThisExpression(), "length"),
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
					factory.Identifier("Number"),
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
									factory.Identifier("Number"),
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
					rangeError(factory, "array index out of bounds"),
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
