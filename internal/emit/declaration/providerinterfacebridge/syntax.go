package providerinterfacebridge

import (
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/runtime/interfacevalue/contract"
	panicruntime "github.com/tsoniclang/gotots/internal/emit/runtime/panic"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func constructor(
	factory tsgo.Factory,
	providerType tsgo.TypeNode,
	contractName string,
	capabilities []capabilitySelection,
	conflicts []capabilityConflict,
	panicName string,
) tsgo.ConstructorDeclaration {
	value := factory.Identifier("value")
	body := make([]tsgo.Statement, 0, len(capabilities)*2+len(conflicts)+1)
	for _, capability := range capabilities {
		body = append(body, capabilityViewDeclaration(
			factory,
			capability,
			value,
		))
	}
	for _, conflict := range conflicts {
		body = append(body, capabilityConflictStatement(
			factory,
			conflict,
			panicName,
		))
	}
	body = append(body, factory.ExpressionStatement(
		factory.CallExpression(
			factory.SuperExpression(),
			nil,
			nil,
			[]tsgo.Expression{
				value,
				capabilityContracts(factory, contractName, capabilities),
			},
			tsgo.NodeFlagsNone,
		),
	))
	for _, capability := range capabilities {
		body = append(body, capabilityFieldAssignment(factory, capability))
	}
	return factory.ConstructorDeclaration(
		[]tsgo.ModifierLike{factory.PrivateKeyword()},
		nil,
		[]tsgo.ParameterDeclaration{
			parameter(factory, value, providerType),
		},
		nil,
		factory.Block(body, true),
	)
}

func capabilityFieldDeclaration(
	factory tsgo.Factory,
	capability capabilitySelection,
) tsgo.PropertyDeclaration {
	return factory.PropertyDeclaration(
		[]tsgo.ModifierLike{
			factory.PrivateKeyword(),
			factory.ReadonlyKeyword(),
		},
		factory.Identifier(capability.fieldName),
		nil,
		nullableReferenceType(factory, capability.reference.Target()),
		nil,
	)
}

func capabilityViewDeclaration(
	factory tsgo.Factory,
	capability capabilitySelection,
	value tsgo.Expression,
) tsgo.VariableStatement {
	return factory.VariableStatement(
		nil,
		factory.VariableDeclarationList(
			[]tsgo.VariableDeclaration{
				factory.VariableDeclaration(
					factory.Identifier(capability.fieldName),
					nil,
					nil,
					factory.CallExpression(
						capability.reference.View().Expression(factory),
						nil,
						nil,
						[]tsgo.Expression{value},
						tsgo.NodeFlagsNone,
					),
				),
			},
			tsgo.NodeFlagsConst,
		),
	)
}

func capabilityConflictStatement(
	factory tsgo.Factory,
	conflict capabilityConflict,
	panicName string,
) tsgo.IfStatement {
	return factory.IfStatement(
		factory.BinaryExpression(
			nil,
			isDefined(factory, factory.Identifier(conflict.left)),
			nil,
			factory.BinaryOperatorToken(
				tsgo.BinaryOperatorAmpersandAmpersandToken,
			),
			isDefined(factory, factory.Identifier(conflict.right)),
		),
		factory.Block(
			[]tsgo.Statement{
				factory.ExpressionStatement(panicruntime.Call(
					factory,
					panicName,
					factory.StringLiteral(
						"provider exposed incompatible Go interface capabilities",
						tsgo.TokenFlagsNone,
					),
				)),
			},
			true,
		),
		nil,
	)
}

func capabilityContracts(
	factory tsgo.Factory,
	base string,
	capabilities []capabilitySelection,
) tsgo.Expression {
	if len(capabilities) == 0 {
		return factory.Identifier(base)
	}
	elements := []tsgo.Expression{
		factory.SpreadElement(factory.Identifier(base)),
	}
	for _, capability := range capabilities {
		elements = append(elements, factory.SpreadElement(
			factory.ConditionalExpression(
				isDefined(
					factory,
					factory.Identifier(capability.fieldName),
				),
				factory.QuestionToken(),
				factory.Identifier(capability.canonical.ContractName()),
				factory.ColonToken(),
				factory.ArrayLiteralExpression(nil, false),
			),
		))
	}
	return factory.ArrayLiteralExpression(elements, false)
}

func capabilityFieldAssignment(
	factory tsgo.Factory,
	capability capabilitySelection,
) tsgo.ExpressionStatement {
	return factory.ExpressionStatement(factory.BinaryExpression(
		nil,
		capabilityField(factory, capability.fieldName),
		nil,
		factory.BinaryOperatorToken(tsgo.BinaryOperatorEqualsToken),
		factory.Identifier(capability.fieldName),
	))
}

func capabilityField(
	factory tsgo.Factory,
	fieldName string,
) tsgo.PropertyAccessExpression {
	return factory.PropertyAccessExpression(
		factory.ThisExpression(),
		nil,
		factory.Identifier(fieldName),
		tsgo.NodeFlagsNone,
	)
}

func isDefined(
	factory tsgo.Factory,
	value tsgo.Expression,
) tsgo.BinaryExpression {
	return factory.BinaryExpression(
		nil,
		value,
		nil,
		factory.BinaryOperatorToken(
			tsgo.BinaryOperatorExclamationEqualsEqualsToken,
		),
		factory.Identifier("undefined"),
	)
}

func fromMethod(
	factory tsgo.Factory,
	name string,
	providerType api.NameReference,
	canonicalType string,
) tsgo.MethodDeclaration {
	value := factory.Identifier("value")
	return factory.MethodDeclaration(
		[]tsgo.ModifierLike{factory.StaticKeyword()},
		nil,
		factory.Identifier(api.ProviderBridgeFromMember),
		nil,
		nil,
		[]tsgo.ParameterDeclaration{
			parameter(
				factory,
				value,
				nullableReferenceType(factory, providerType),
			),
		},
		nullableType(factory, canonicalType),
		factory.Block(
			[]tsgo.Statement{
				factory.ReturnStatement(
					factory.ConditionalExpression(
						factory.BinaryExpression(
							nil,
							value,
							nil,
							factory.BinaryOperatorToken(
								tsgo.BinaryOperatorEqualsEqualsEqualsToken,
							),
							factory.Identifier("undefined"),
						),
						factory.QuestionToken(),
						factory.Identifier("undefined"),
						factory.ColonToken(),
						factory.ConditionalExpression(
							factory.BinaryExpression(
								nil,
								value,
								nil,
								factory.BinaryOperatorToken(
									tsgo.BinaryOperatorInstanceOfKeyword,
								),
								factory.Identifier(name),
							),
							factory.QuestionToken(),
							value,
							factory.ColonToken(),
							factory.NewExpression(
								factory.Identifier(name),
								nil,
								[]tsgo.Expression{value},
							),
						),
					),
				),
			},
			true,
		),
	)
}

func toMethod(
	factory tsgo.Factory,
	name string,
	providerType api.NameReference,
	canonicalType string,
	panicName string,
	directProviderUse bool,
) tsgo.MethodDeclaration {
	value := factory.Identifier("value")
	var finalReturn tsgo.Statement
	if directProviderUse {
		finalReturn = factory.ReturnStatement(value)
	} else {
		finalReturn = factory.ReturnStatement(
			panicruntime.Call(
				factory,
				panicName,
				factory.StringLiteral(
					"provider interface received a foreign implementation",
					tsgo.TokenFlagsNone,
				),
			),
		)
	}
	return factory.MethodDeclaration(
		[]tsgo.ModifierLike{factory.StaticKeyword()},
		nil,
		factory.Identifier(api.ProviderBridgeToMember),
		nil,
		nil,
		[]tsgo.ParameterDeclaration{
			parameter(factory, value, nullableType(factory, canonicalType)),
		},
		nullableReferenceType(factory, providerType),
		factory.Block(
			[]tsgo.Statement{
				factory.IfStatement(
					factory.BinaryExpression(
						nil,
						value,
						nil,
						factory.BinaryOperatorToken(
							tsgo.BinaryOperatorEqualsEqualsEqualsToken,
						),
						factory.Identifier("undefined"),
					),
					factory.Block(
						[]tsgo.Statement{
							factory.ReturnStatement(factory.Identifier("undefined")),
						},
						true,
					),
					nil,
				),
				factory.IfStatement(
					factory.BinaryExpression(
						nil,
						value,
						nil,
						factory.BinaryOperatorToken(
							tsgo.BinaryOperatorInstanceOfKeyword,
						),
						factory.Identifier(name),
					),
					factory.Block(
						[]tsgo.Statement{
							factory.ReturnStatement(
								factory.PropertyAccessExpression(
									value,
									nil,
									factory.Identifier(contract.PayloadMember),
									tsgo.NodeFlagsNone,
								),
							),
						},
						true,
					),
					nil,
				),
				finalReturn,
			},
			true,
		),
	)
}

func payload(factory tsgo.Factory) tsgo.PropertyAccessExpression {
	return factory.PropertyAccessExpression(
		factory.ThisExpression(),
		nil,
		factory.Identifier(contract.PayloadMember),
		tsgo.NodeFlagsNone,
	)
}

func nullableType(factory tsgo.Factory, name string) tsgo.TypeNode {
	return factory.UnionTypeNode([]tsgo.TypeNode{
		factory.TypeReferenceNode(factory.Identifier(name), nil),
		factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindUndefinedKeyword),
	})
}

func nullableReferenceType(
	factory tsgo.Factory,
	reference api.NameReference,
) tsgo.TypeNode {
	return factory.UnionTypeNode([]tsgo.TypeNode{
		factory.TypeReferenceNode(reference.EntityName(factory), nil),
		factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindUndefinedKeyword),
	})
}

func parameter(
	factory tsgo.Factory,
	name tsgo.BindingName,
	typeNode tsgo.TypeNode,
) tsgo.ParameterDeclaration {
	return factory.ParameterDeclaration(nil, nil, name, nil, typeNode, nil)
}
