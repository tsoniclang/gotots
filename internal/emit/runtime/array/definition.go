package array

import (
	"github.com/tsoniclang/gotots/internal/emit/api"
	arraymember "github.com/tsoniclang/gotots/internal/emit/runtime/array/member"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type Capabilities struct {
	Allocate bool
	View     bool
	Location bool
}

func Build(
	factory tsgo.Factory,
	panicName string,
) (tsgo.ClassDeclaration, error) {
	return BuildWithCapabilities(
		factory,
		panicName,
		Capabilities{},
	)
}

func BuildWithCapabilities(
	factory tsgo.Factory,
	panicName string,
	capabilities Capabilities,
) (tsgo.ClassDeclaration, error) {
	contract, err := api.RuntimeContract(api.RuntimeArray)
	if err != nil {
		return nil, err
	}
	exportedName := contract.ExportedName()
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
	}
	if capabilities.Allocate {
		members = append(members, allocateMethod(factory, exportedName))
	}
	if capabilities.View {
		members = append(members, viewMethod(factory, exportedName))
	}
	if capabilities.Location {
		members = append(members, locationMethod(factory))
	}
	members = append(
		members,
		zeroMethod(factory, exportedName),
		literalMethod(factory, exportedName, panicName),
		copyMethod(factory, exportedName, elementType, lengthType),
		getMethod(factory, elementType),
		setMethod(factory, elementType),
		checkMethod(factory, panicName),
	)
	return factory.ClassDeclaration(
		[]tsgo.ModifierLike{factory.ExportKeyword()},
		factory.Identifier(exportedName),
		typeParameters,
		nil,
		members,
	), nil
}

const StorageAllocateMember = "$allocate"

func allocateMethod(
	factory tsgo.Factory,
	exportedName string,
) tsgo.MethodDeclaration {
	elementType := typeReference(factory, "T")
	lengthType := typeReference(factory, "N")
	length := factory.Identifier("length")
	values := factory.NewExpression(
		factory.Identifier("Array"),
		[]tsgo.TypeNode{elementType},
		[]tsgo.Expression{call(
			factory,
			api.TargetIntrinsicNumber.Expression(factory),
			nil,
			length,
		)},
	)
	return method(
		factory,
		[]tsgo.ModifierLike{
			factory.PublicKeyword(),
			factory.StaticKeyword(),
		},
		StorageAllocateMember,
		typeParameters(factory),
		[]tsgo.ParameterDeclaration{
			parameter(factory, nil, "length", lengthType),
		},
		arrayType(factory, exportedName, elementType, lengthType),
		[]tsgo.Statement{factory.ReturnStatement(factory.NewExpression(
			factory.Identifier(exportedName),
			[]tsgo.TypeNode{elementType, lengthType},
			[]tsgo.Expression{
				values,
				factory.NumericLiteral("0", tsgo.TokenFlagsNone),
				length,
			},
		))},
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
					factory.PrivateKeyword(),
					factory.ReadonlyKeyword(),
				},
				"$offset",
				factory.KeywordTypeNode(
					tsgo.KeywordTypeSyntaxKindNumberKeyword,
				),
			),
			parameter(
				factory,
				[]tsgo.ModifierLike{
					factory.PublicKeyword(),
					factory.ReadonlyKeyword(),
				},
				arraymember.Length.Name(),
				lengthType,
			),
		},
		nil,
		factory.Block(nil, true),
	)
}

func zeroMethod(
	factory tsgo.Factory,
	exportedName string,
) tsgo.MethodDeclaration {
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
			factory.Identifier(exportedName),
			[]tsgo.TypeNode{elementType, lengthType},
			[]tsgo.Expression{
				values,
				factory.NumericLiteral("0", tsgo.TokenFlagsNone),
				length,
			},
		)),
	}
	return runtimeMethod(
		factory,
		[]tsgo.ModifierLike{
			factory.PublicKeyword(),
			factory.StaticKeyword(),
		},
		arraymember.Zero,
		typeParameters(factory),
		[]tsgo.ParameterDeclaration{
			parameter(factory, nil, "length", lengthType),
			parameter(factory, nil, "zero", elementType),
		},
		arrayType(factory, exportedName, elementType, lengthType),
		body,
	)
}

func literalMethod(
	factory tsgo.Factory,
	exportedName string,
	panicName string,
) tsgo.MethodDeclaration {
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
			factory.Block([]tsgo.Statement{boundsPanic(
				factory,
				panicName,
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
				runtimeProperty(
					factory,
					factory.Identifier(exportedName),
					arraymember.Zero,
				),
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
					runtimeProperty(factory, result, arraymember.Set),
					nil,
					definedElement(factory, indexes, entry),
					definedElement(factory, values, entry),
				)),
			}, true),
		),
		factory.ReturnStatement(result),
	}
	return runtimeMethod(
		factory,
		[]tsgo.ModifierLike{
			factory.PublicKeyword(),
			factory.StaticKeyword(),
		},
		arraymember.Literal,
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
		arrayType(factory, exportedName, elementType, lengthType),
		body,
	)
}

func copyMethod(
	factory tsgo.Factory,
	exportedName string,
	elementType tsgo.TypeNode,
	lengthType tsgo.TypeNode,
) tsgo.MethodDeclaration {
	return runtimeMethod(
		factory,
		[]tsgo.ModifierLike{factory.PublicKeyword()},
		arraymember.Copy,
		nil,
		nil,
		arrayType(factory, exportedName, elementType, lengthType),
		[]tsgo.Statement{factory.ReturnStatement(factory.NewExpression(
			factory.Identifier(exportedName),
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
					property(factory, factory.ThisExpression(), "$offset"),
					binary(
						factory,
						property(factory, factory.ThisExpression(), "$offset"),
						tsgo.BinaryOperatorPlusToken,
						call(
							factory,
							api.TargetIntrinsicNumber.Expression(factory),
							nil,
							runtimeProperty(
								factory,
								factory.ThisExpression(),
								arraymember.Length,
							),
						),
					),
				),
				factory.NumericLiteral("0", tsgo.TokenFlagsNone),
				runtimeProperty(
					factory,
					factory.ThisExpression(),
					arraymember.Length,
				),
			},
		))},
	)
}

func getMethod(
	factory tsgo.Factory,
	elementType tsgo.TypeNode,
) tsgo.MethodDeclaration {
	index := factory.Identifier("index")
	return runtimeMethod(
		factory,
		[]tsgo.ModifierLike{factory.PublicKeyword()},
		arraymember.Get,
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
			factory.ReturnStatement(definedElement(
				factory,
				property(factory, factory.ThisExpression(), "$values"),
				binary(
					factory,
					property(factory, factory.ThisExpression(), "$offset"),
					tsgo.BinaryOperatorPlusToken,
					factory.Identifier("offset"),
				),
			)),
		},
	)
}

func setMethod(
	factory tsgo.Factory,
	elementType tsgo.TypeNode,
) tsgo.MethodDeclaration {
	index := factory.Identifier("index")
	return runtimeMethod(
		factory,
		[]tsgo.ModifierLike{factory.PublicKeyword()},
		arraymember.Set,
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
					binary(
						factory,
						property(factory, factory.ThisExpression(), "$offset"),
						tsgo.BinaryOperatorPlusToken,
						factory.Identifier("offset"),
					),
				),
				tsgo.BinaryOperatorEqualsToken,
				factory.Identifier("value"),
			)),
		},
	)
}

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
