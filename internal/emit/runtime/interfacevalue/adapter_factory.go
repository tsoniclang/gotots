package interfacevalue

import (
	interfacecontract "github.com/tsoniclang/gotots/internal/emit/runtime/interfacevalue/contract"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

const emptyAdapterMethodsMember = "$go$emptyAdapterMethods"

func adapterFactoryFunction(
	factory tsgo.Factory,
	name string,
	valueName string,
) tsgo.FunctionDeclaration {
	typeT := factory.TypeReferenceNode(factory.Identifier("T"), nil)
	stringType := factory.KeywordTypeNode(
		tsgo.KeywordTypeSyntaxKindStringKeyword,
	)
	precisionType := factory.UnionTypeNode([]tsgo.TypeNode{
		factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindNumberKeyword),
		factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindUndefinedKeyword),
	})
	return factory.FunctionDeclaration(
		[]tsgo.ModifierLike{factory.ExportKeyword()},
		nil,
		factory.Identifier(name),
		[]tsgo.TypeParameterDeclaration{factory.TypeParameterDeclaration(
			nil,
			factory.Identifier("T"),
			nil,
			nil,
			nil,
		)},
		[]tsgo.ParameterDeclaration{
			parameter(factory, "dynamicType", interfacecontract.DynamicType(factory)),
			parameter(factory, "equal", factory.FunctionTypeNode(
				nil,
				[]tsgo.ParameterDeclaration{
					parameter(factory, "left", typeT),
					parameter(factory, "right", typeT),
				},
				factory.KeywordTypeNode(
					tsgo.KeywordTypeSyntaxKindBooleanKeyword,
				),
			)),
			parameter(factory, "hash", factory.FunctionTypeNode(
				nil,
				[]tsgo.ParameterDeclaration{parameter(factory, "value", typeT)},
				factory.KeywordTypeNode(
					tsgo.KeywordTypeSyntaxKindNumberKeyword,
				),
			)),
			parameter(
				factory,
				"formatString",
				factory.KeywordTypeNode(
					tsgo.KeywordTypeSyntaxKindBooleanKeyword,
				),
			),
			parameter(factory, "format", factory.FunctionTypeNode(
				nil,
				[]tsgo.ParameterDeclaration{
					parameter(factory, "value", typeT),
					parameter(factory, "verb", stringType),
					parameter(factory, "flags", stringType),
					parameter(factory, "precision", precisionType),
				},
				stringType,
			)),
		},
		interfacecontract.AdapterConstructor(factory, valueName, typeT),
		factory.Block([]tsgo.Statement{factory.ReturnStatement(
			adapterClassExpression(factory, valueName, typeT),
		)}, true),
	)
}

func adapterClassExpression(
	factory tsgo.Factory,
	valueName string,
	typeT tsgo.TypeNode,
) tsgo.ClassExpression {
	return factory.ClassExpression(
		nil,
		factory.Identifier("Adapter"),
		nil,
		[]tsgo.HeritageClause{factory.HeritageClause(
			tsgo.HeritageClauseTokenKindExtendsKeyword,
			[]tsgo.ExpressionWithTypeArguments{
				factory.ExpressionWithTypeArguments(
					factory.Identifier(valueName),
					nil,
				),
			},
		)},
		[]tsgo.ClassElement{
			adapterEmptyMethodsProperty(factory),
			adapterConstructor(factory, typeT),
			adapterDynamicTypeProperty(factory),
			adapterGuardMethod(factory, valueName),
			adapterMethodsProperty(factory, valueName),
			adapterImplementsMethod(factory),
			adapterEqualMethod(factory, valueName),
			adapterHashMethod(factory),
			adapterFormatStringProperty(factory),
			adapterFormatMethod(factory),
		},
	)
}

func adapterEmptyMethodsProperty(factory tsgo.Factory) tsgo.PropertyDeclaration {
	return factory.PropertyDeclaration(
		[]tsgo.ModifierLike{
			factory.PrivateKeyword(),
			factory.StaticKeyword(),
			factory.ReadonlyKeyword(),
		},
		factory.Identifier(emptyAdapterMethodsMember),
		nil,
		readonlySetType(factory),
		factory.NewExpression(
			factory.Identifier("Set"),
			[]tsgo.TypeNode{factory.KeywordTypeNode(
				tsgo.KeywordTypeSyntaxKindObjectKeyword,
			)},
			nil,
		),
	)
}

func adapterConstructor(
	factory tsgo.Factory,
	typeT tsgo.TypeNode,
) tsgo.ConstructorDeclaration {
	return factory.ConstructorDeclaration(
		nil,
		nil,
		[]tsgo.ParameterDeclaration{factory.ParameterDeclaration(
			[]tsgo.ModifierLike{
				factory.PublicKeyword(),
				factory.ReadonlyKeyword(),
			},
			nil,
			factory.Identifier(interfacecontract.PayloadMember),
			nil,
			typeT,
			nil,
		)},
		nil,
		factory.Block([]tsgo.Statement{factory.ExpressionStatement(
			factory.CallExpression(
				factory.SuperExpression(),
				nil,
				nil,
				nil,
				tsgo.NodeFlagsNone,
			),
		)}, true),
	)
}

func adapterDynamicTypeProperty(factory tsgo.Factory) tsgo.PropertyDeclaration {
	return factory.PropertyDeclaration(
		[]tsgo.ModifierLike{factory.ReadonlyKeyword()},
		factory.Identifier(interfacecontract.DynamicTypeMember),
		nil,
		interfacecontract.DynamicType(factory),
		factory.Identifier("dynamicType"),
	)
}

func adapterGuardMethod(
	factory tsgo.Factory,
	valueName string,
) tsgo.MethodDeclaration {
	value := factory.Identifier("value")
	return factory.MethodDeclaration(
		[]tsgo.ModifierLike{factory.StaticKeyword()},
		nil,
		factory.Identifier("$is"),
		nil,
		nil,
		[]tsgo.ParameterDeclaration{parameter(
			factory,
			"value",
			factory.UnionTypeNode([]tsgo.TypeNode{
				factory.TypeReferenceNode(factory.Identifier(valueName), nil),
				factory.KeywordTypeNode(
					tsgo.KeywordTypeSyntaxKindUndefinedKeyword,
				),
			}),
		)},
		factory.TypePredicateNode(
			nil,
			value,
			factory.TypeReferenceNode(factory.Identifier("Adapter"), nil),
		),
		factory.Block([]tsgo.Statement{factory.ReturnStatement(
			factory.BinaryExpression(
				nil,
				factory.BinaryExpression(
					nil,
					value,
					nil,
					factory.BinaryOperatorToken(
						tsgo.BinaryOperatorExclamationEqualsEqualsToken,
					),
					factory.Identifier("undefined"),
				),
				nil,
				factory.BinaryOperatorToken(
					tsgo.BinaryOperatorAmpersandAmpersandToken,
				),
				factory.BinaryExpression(
					nil,
					factory.PropertyAccessExpression(
						value,
						nil,
						factory.Identifier(interfacecontract.DynamicTypeMember),
						tsgo.NodeFlagsNone,
					),
					nil,
					factory.BinaryOperatorToken(
						tsgo.BinaryOperatorEqualsEqualsEqualsToken,
					),
					factory.Identifier("dynamicType"),
				),
			),
		)}, true),
	)
}

func adapterMethodsProperty(factory tsgo.Factory, valueName string) tsgo.PropertyDeclaration {
	return factory.PropertyDeclaration(
		[]tsgo.ModifierLike{factory.ReadonlyKeyword()},
		factory.Identifier(interfacecontract.MethodsMember),
		nil,
		readonlySetType(factory),
		factory.PropertyAccessExpression(
			factory.Identifier("Adapter"),
			nil,
			factory.Identifier(emptyAdapterMethodsMember),
			tsgo.NodeFlagsNone,
		),
	)
}

func adapterImplementsMethod(factory tsgo.Factory) tsgo.MethodDeclaration {
	contract := factory.Identifier("contract")
	token := factory.Identifier("token")
	return factory.MethodDeclaration(
		nil,
		nil,
		factory.Identifier(interfacecontract.ImplementsMember),
		nil,
		nil,
		[]tsgo.ParameterDeclaration{parameter(
			factory,
			"contract",
			readonlyObjectArray(factory),
		)},
		factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindBooleanKeyword),
		factory.Block([]tsgo.Statement{factory.ReturnStatement(
			factory.CallExpression(
				factory.PropertyAccessExpression(
					contract,
					nil,
					factory.Identifier("every"),
					tsgo.NodeFlagsNone,
				),
				nil,
				nil,
				[]tsgo.Expression{factory.ArrowFunction(
					nil,
					nil,
					[]tsgo.ParameterDeclaration{parameter(
						factory,
						"token",
						factory.KeywordTypeNode(
							tsgo.KeywordTypeSyntaxKindObjectKeyword,
						),
					)},
					factory.KeywordTypeNode(
						tsgo.KeywordTypeSyntaxKindBooleanKeyword,
					),
					factory.EqualsGreaterThanToken(),
					factory.CallExpression(
						factory.PropertyAccessExpression(
							factory.PropertyAccessExpression(
								factory.ThisExpression(),
								nil,
								factory.Identifier(interfacecontract.MethodsMember),
								tsgo.NodeFlagsNone,
							),
							nil,
							factory.Identifier("has"),
							tsgo.NodeFlagsNone,
						),
						nil,
						nil,
						[]tsgo.Expression{token},
						tsgo.NodeFlagsNone,
					),
				)},
				tsgo.NodeFlagsNone,
			),
		)}, true),
	)
}

func adapterEqualMethod(factory tsgo.Factory, valueName string) tsgo.MethodDeclaration {
	other := factory.Identifier("other")
	return factory.MethodDeclaration(
		nil,
		nil,
		factory.Identifier(interfacecontract.EqualMember),
		nil,
		nil,
		[]tsgo.ParameterDeclaration{parameter(
			factory,
			"other",
			factory.TypeReferenceNode(factory.Identifier(valueName), nil),
		)},
		factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindBooleanKeyword),
		factory.Block([]tsgo.Statement{factory.ReturnStatement(
			factory.BinaryExpression(
				nil,
				factory.CallExpression(
					factory.PropertyAccessExpression(
						factory.Identifier("Adapter"),
						nil,
						factory.Identifier("$is"),
						tsgo.NodeFlagsNone,
					),
					nil,
					nil,
					[]tsgo.Expression{other},
					tsgo.NodeFlagsNone,
				),
				nil,
				factory.BinaryOperatorToken(
					tsgo.BinaryOperatorAmpersandAmpersandToken,
				),
				factory.CallExpression(
					factory.Identifier("equal"),
					nil,
					nil,
					[]tsgo.Expression{
						factory.PropertyAccessExpression(
							factory.ThisExpression(),
							nil,
							factory.Identifier(interfacecontract.PayloadMember),
							tsgo.NodeFlagsNone,
						),
						factory.PropertyAccessExpression(
							other,
							nil,
							factory.Identifier(interfacecontract.PayloadMember),
							tsgo.NodeFlagsNone,
						),
					},
					tsgo.NodeFlagsNone,
				),
			),
		)}, true),
	)
}

func adapterHashMethod(factory tsgo.Factory) tsgo.MethodDeclaration {
	return zeroArgumentAdapterOperation(
		factory,
		interfacecontract.HashMember,
		"hash",
		factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindNumberKeyword),
	)
}

func adapterFormatStringProperty(factory tsgo.Factory) tsgo.PropertyDeclaration {
	return factory.PropertyDeclaration(
		[]tsgo.ModifierLike{factory.ReadonlyKeyword()},
		factory.Identifier(interfacecontract.FormatStringMember),
		nil,
		factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindBooleanKeyword),
		factory.Identifier("formatString"),
	)
}

func adapterFormatMethod(factory tsgo.Factory) tsgo.MethodDeclaration {
	stringType := factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindStringKeyword)
	precisionType := factory.UnionTypeNode([]tsgo.TypeNode{
		factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindNumberKeyword),
		factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindUndefinedKeyword),
	})
	parameters := []tsgo.ParameterDeclaration{
		parameter(factory, "verb", stringType),
		parameter(factory, "flags", stringType),
		parameter(factory, "precision", precisionType),
	}
	return factory.MethodDeclaration(
		nil,
		nil,
		factory.Identifier(interfacecontract.FormatMember),
		nil,
		nil,
		parameters,
		stringType,
		factory.Block([]tsgo.Statement{factory.ReturnStatement(
			factory.CallExpression(
				factory.Identifier("format"),
				nil,
				nil,
				[]tsgo.Expression{
					factory.PropertyAccessExpression(
						factory.ThisExpression(),
						nil,
						factory.Identifier(interfacecontract.PayloadMember),
						tsgo.NodeFlagsNone,
					),
					factory.Identifier("verb"),
					factory.Identifier("flags"),
					factory.Identifier("precision"),
				},
				tsgo.NodeFlagsNone,
			),
		)}, true),
	)
}

func zeroArgumentAdapterOperation(
	factory tsgo.Factory,
	member string,
	operation string,
	result tsgo.TypeNode,
) tsgo.MethodDeclaration {
	return factory.MethodDeclaration(
		nil,
		nil,
		factory.Identifier(member),
		nil,
		nil,
		nil,
		result,
		factory.Block([]tsgo.Statement{factory.ReturnStatement(
			factory.CallExpression(
				factory.Identifier(operation),
				nil,
				nil,
				[]tsgo.Expression{factory.PropertyAccessExpression(
					factory.ThisExpression(),
					nil,
					factory.Identifier(interfacecontract.PayloadMember),
					tsgo.NodeFlagsNone,
				)},
				tsgo.NodeFlagsNone,
			),
		)}, true),
	)
}
