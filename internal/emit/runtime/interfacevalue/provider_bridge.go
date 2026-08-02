package interfacevalue

import (
	interfacecontract "github.com/tsoniclang/gotots/internal/emit/runtime/interfacevalue/contract"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func providerBridgeContract(
	factory tsgo.Factory,
	valueName string,
) tsgo.ClassDeclaration {
	valueType := factory.TypeReferenceNode(factory.Identifier("T"), nil)
	baseType := factory.TypeReferenceNode(factory.Identifier(valueName), nil)
	return factory.ClassDeclaration(
		[]tsgo.ModifierLike{factory.ExportKeyword(), factory.AbstractKeyword()},
		factory.Identifier("GoProviderInterfaceBridge"),
		[]tsgo.TypeParameterDeclaration{
			factory.TypeParameterDeclaration(
				nil,
				factory.Identifier("T"),
				baseType,
				nil,
				nil,
			),
		},
		[]tsgo.HeritageClause{
			factory.HeritageClause(
				tsgo.HeritageClauseTokenKindExtendsKeyword,
				[]tsgo.ExpressionWithTypeArguments{
					factory.ExpressionWithTypeArguments(
						factory.Identifier(valueName),
						nil,
					),
				},
			),
		},
		[]tsgo.ClassElement{
			providerBridgeProperty(factory, interfacecontract.PayloadMember, valueType),
			providerBridgeProperty(factory, interfacecontract.DynamicTypeMember, interfacecontract.DynamicType(factory)),
			providerBridgeMethodsProperty(factory),
			providerBridgeProperty(factory, interfacecontract.FormatStringMember, booleanType(factory)),
			providerBridgeConstructor(factory, valueType),
			providerBridgeImplements(factory),
			providerBridgeEqual(factory, valueName),
			providerBridgeDelegate(factory, interfacecontract.HashMember, nil, numberType(factory)),
			providerBridgeFormat(factory),
		},
	)
}

func providerBridgeConstructor(
	factory tsgo.Factory,
	valueType tsgo.TypeNode,
) tsgo.ConstructorDeclaration {
	methodsType := readonlyObjectArray(factory)
	return factory.ConstructorDeclaration(
		nil,
		nil,
		[]tsgo.ParameterDeclaration{
			parameter(factory, interfacecontract.PayloadMember, valueType),
			parameter(factory, "methods", methodsType),
		},
		nil,
		factory.Block(
			[]tsgo.Statement{
				factory.ExpressionStatement(
					factory.CallExpression(
						factory.SuperExpression(),
						nil,
						nil,
						nil,
						tsgo.NodeFlagsNone,
					),
				),
				factory.ExpressionStatement(
					factory.BinaryExpression(
						nil,
						factory.PropertyAccessExpression(
							factory.ThisExpression(),
							nil,
							factory.Identifier(interfacecontract.PayloadMember),
							tsgo.NodeFlagsNone,
						),
						nil,
						factory.BinaryOperatorToken(tsgo.BinaryOperatorEqualsToken),
						factory.Identifier(interfacecontract.PayloadMember),
					),
				),
				providerBridgeAssignment(
					factory,
					interfacecontract.DynamicTypeMember,
					providerValueMember(factory, interfacecontract.DynamicTypeMember),
				),
				providerBridgeAssignment(
					factory,
					interfacecontract.FormatStringMember,
					providerValueMember(factory, interfacecontract.FormatStringMember),
				),
				factory.ExpressionStatement(
					factory.BinaryExpression(
						nil,
						factory.PropertyAccessExpression(
							factory.ThisExpression(),
							nil,
							factory.Identifier(interfacecontract.MethodsMember),
							tsgo.NodeFlagsNone,
						),
						nil,
						factory.BinaryOperatorToken(tsgo.BinaryOperatorEqualsToken),
						factory.NewExpression(
							factory.Identifier("Set"),
							[]tsgo.TypeNode{objectType(factory)},
							[]tsgo.Expression{
								factory.ArrayLiteralExpression(
									[]tsgo.Expression{
										factory.SpreadElement(
											providerValueMember(factory, interfacecontract.MethodsMember),
										),
										factory.SpreadElement(factory.Identifier("methods")),
									},
									false,
								),
							},
						),
					),
				),
			},
			true,
		),
	)
}

func providerBridgeProperty(
	factory tsgo.Factory,
	name string,
	typeNode tsgo.TypeNode,
) tsgo.PropertyDeclaration {
	modifiers := []tsgo.ModifierLike{factory.ReadonlyKeyword()}
	if name == interfacecontract.PayloadMember {
		modifiers = append([]tsgo.ModifierLike{factory.ProtectedKeyword()}, modifiers...)
	}
	return factory.PropertyDeclaration(
		modifiers,
		factory.Identifier(name),
		nil,
		typeNode,
		nil,
	)
}

func providerBridgeAssignment(
	factory tsgo.Factory,
	member string,
	value tsgo.Expression,
) tsgo.Statement {
	return factory.ExpressionStatement(
		factory.BinaryExpression(
			nil,
			factory.PropertyAccessExpression(
				factory.ThisExpression(),
				nil,
				factory.Identifier(member),
				tsgo.NodeFlagsNone,
			),
			nil,
			factory.BinaryOperatorToken(tsgo.BinaryOperatorEqualsToken),
			value,
		),
	)
}

func providerBridgeMethodsProperty(factory tsgo.Factory) tsgo.PropertyDeclaration {
	return factory.PropertyDeclaration(
		[]tsgo.ModifierLike{factory.ReadonlyKeyword()},
		factory.Identifier(interfacecontract.MethodsMember),
		nil,
		readonlySetType(factory),
		nil,
	)
}

func providerBridgeImplements(factory tsgo.Factory) tsgo.MethodDeclaration {
	contract := factory.Identifier("contract")
	token := factory.Identifier("token")
	return factory.MethodDeclaration(
		nil,
		nil,
		factory.Identifier(interfacecontract.ImplementsMember),
		nil,
		nil,
		[]tsgo.ParameterDeclaration{
			parameter(factory, "contract", readonlyObjectArray(factory)),
		},
		booleanType(factory),
		factory.Block(
			[]tsgo.Statement{
				factory.ReturnStatement(
					factory.CallExpression(
						factory.PropertyAccessExpression(contract, nil, factory.Identifier("every"), tsgo.NodeFlagsNone),
						nil,
						nil,
						[]tsgo.Expression{
							factory.ArrowFunction(
								nil,
								nil,
								[]tsgo.ParameterDeclaration{parameter(factory, "token", objectType(factory))},
								booleanType(factory),
								factory.EqualsGreaterThanToken(),
								factory.CallExpression(
									factory.PropertyAccessExpression(
										factory.PropertyAccessExpression(factory.ThisExpression(), nil, factory.Identifier(interfacecontract.MethodsMember), tsgo.NodeFlagsNone),
										nil,
										factory.Identifier("has"),
										tsgo.NodeFlagsNone,
									),
									nil,
									nil,
									[]tsgo.Expression{token},
									tsgo.NodeFlagsNone,
								),
							),
						},
						tsgo.NodeFlagsNone,
					),
				),
			},
			true,
		),
	)
}

func providerBridgeEqual(
	factory tsgo.Factory,
	valueName string,
) tsgo.MethodDeclaration {
	other := factory.Identifier("other")
	bridgeName := factory.Identifier("GoProviderInterfaceBridge")
	otherValue := factory.PropertyAccessExpression(other, nil, factory.Identifier(interfacecontract.PayloadMember), tsgo.NodeFlagsNone)
	return factory.MethodDeclaration(
		nil,
		nil,
		factory.Identifier(interfacecontract.EqualMember),
		nil,
		nil,
		[]tsgo.ParameterDeclaration{
			parameter(factory, "other", factory.TypeReferenceNode(factory.Identifier(valueName), nil)),
		},
		booleanType(factory),
		factory.Block(
			[]tsgo.Statement{
				factory.ReturnStatement(
					factory.BinaryExpression(
						nil,
						factory.BinaryExpression(
							nil,
							other,
							nil,
							factory.BinaryOperatorToken(tsgo.BinaryOperatorInstanceOfKeyword),
							bridgeName,
						),
						nil,
						factory.BinaryOperatorToken(tsgo.BinaryOperatorAmpersandAmpersandToken),
						factory.CallExpression(
							providerValueMember(factory, interfacecontract.EqualMember),
							nil,
							nil,
							[]tsgo.Expression{otherValue},
							tsgo.NodeFlagsNone,
						),
					),
				),
			},
			true,
		),
	)
}

func providerBridgeDelegate(
	factory tsgo.Factory,
	name string,
	parameters []tsgo.ParameterDeclaration,
	result tsgo.TypeNode,
) tsgo.MethodDeclaration {
	return factory.MethodDeclaration(
		nil,
		nil,
		factory.Identifier(name),
		nil,
		nil,
		parameters,
		result,
		factory.Block(
			[]tsgo.Statement{
				factory.ReturnStatement(
					factory.CallExpression(
						providerValueMember(factory, name),
						nil,
						nil,
						nil,
						tsgo.NodeFlagsNone,
					),
				),
			},
			true,
		),
	)
}

func providerBridgeFormat(factory tsgo.Factory) tsgo.MethodDeclaration {
	parameters := []tsgo.ParameterDeclaration{
		parameter(factory, "verb", stringType(factory)),
		parameter(factory, "flags", stringType(factory)),
		parameter(
			factory,
			"precision",
			factory.UnionTypeNode([]tsgo.TypeNode{
				numberType(factory),
				factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindUndefinedKeyword),
			}),
		),
	}
	references := []tsgo.Expression{
		factory.Identifier("verb"),
		factory.Identifier("flags"),
		factory.Identifier("precision"),
	}
	return factory.MethodDeclaration(
		nil,
		nil,
		factory.Identifier(interfacecontract.FormatMember),
		nil,
		nil,
		parameters,
		stringType(factory),
		factory.Block(
			[]tsgo.Statement{
				factory.ReturnStatement(
					factory.CallExpression(
						providerValueMember(factory, interfacecontract.FormatMember),
						nil,
						nil,
						references,
						tsgo.NodeFlagsNone,
					),
				),
			},
			true,
		),
	)
}

func providerValueMember(
	factory tsgo.Factory,
	member string,
) tsgo.PropertyAccessExpression {
	return factory.PropertyAccessExpression(
		factory.PropertyAccessExpression(
			factory.ThisExpression(),
			nil,
			factory.Identifier(interfacecontract.PayloadMember),
			tsgo.NodeFlagsNone,
		),
		nil,
		factory.Identifier(member),
		tsgo.NodeFlagsNone,
	)
}

func objectType(factory tsgo.Factory) tsgo.TypeNode {
	return factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindObjectKeyword)
}

func booleanType(factory tsgo.Factory) tsgo.TypeNode {
	return factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindBooleanKeyword)
}

func numberType(factory tsgo.Factory) tsgo.TypeNode {
	return factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindNumberKeyword)
}

func stringType(factory tsgo.Factory) tsgo.TypeNode {
	return factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindStringKeyword)
}
