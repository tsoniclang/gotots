package interfacevalue

import (
	"github.com/tsoniclang/gotots/internal/emit/api"
	panicruntime "github.com/tsoniclang/gotots/internal/emit/runtime/panic"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

const (
	DynamicTypeMember = "$go$type"
	MethodsMember     = "$go$methods"
	ImplementsMember  = "$go$implements"
	EqualMember       = "$go$equal"
	HashMember        = "$go$hash"
)

func Build(
	factory tsgo.Factory,
	symbol api.RuntimeSymbol,
	valueName string,
	panicName string,
) (tsgo.Statement, error) {
	switch symbol {
	case api.RuntimeInterfaceValue:
		return valueContract(factory, valueName), nil
	case api.RuntimeInterfaceNonNil:
		return nonNil(factory, valueName, panicName), nil
	case api.RuntimeInterfaceEqual:
		return equal(factory, valueName), nil
	default:
		return nil, &api.RuntimeSymbolError{Symbol: symbol}
	}
}

func equal(
	factory tsgo.Factory,
	valueName string,
) tsgo.FunctionDeclaration {
	valueType := factory.UnionTypeNode([]tsgo.TypeNode{
		factory.TypeReferenceNode(
			factory.Identifier(valueName),
			nil,
		),
		factory.KeywordTypeNode(
			tsgo.KeywordTypeSyntaxKindUndefinedKeyword,
		),
	})
	left := factory.Identifier("left")
	right := factory.Identifier("right")
	return factory.FunctionDeclaration(
		[]tsgo.ModifierLike{factory.ExportKeyword()},
		nil,
		factory.Identifier("goInterfaceEqual"),
		nil,
		[]tsgo.ParameterDeclaration{
			parameter(factory, "left", valueType),
			parameter(factory, "right", valueType),
		},
		factory.KeywordTypeNode(
			tsgo.KeywordTypeSyntaxKindBooleanKeyword,
		),
		factory.Block(
			[]tsgo.Statement{
				factory.ReturnStatement(
					factory.ConditionalExpression(
						strictUndefined(factory, left),
						factory.QuestionToken(),
						strictUndefined(factory, right),
						factory.ColonToken(),
						factory.BinaryExpression(
							nil,
							strictDefined(factory, right),
							nil,
							factory.BinaryOperatorToken(
								tsgo.BinaryOperatorAmpersandAmpersandToken,
							),
							factory.CallExpression(
								factory.PropertyAccessExpression(
									left,
									nil,
									factory.Identifier(EqualMember),
									tsgo.NodeFlagsNone,
								),
								nil,
								nil,
								[]tsgo.Expression{right},
								tsgo.NodeFlagsNone,
							),
						),
					),
				),
			},
			true,
		),
	)
}

func strictUndefined(
	factory tsgo.Factory,
	value tsgo.Expression,
) tsgo.BinaryExpression {
	return factory.BinaryExpression(
		nil,
		value,
		nil,
		factory.BinaryOperatorToken(
			tsgo.BinaryOperatorEqualsEqualsEqualsToken,
		),
		factory.Identifier("undefined"),
	)
}

func strictDefined(
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

func valueContract(
	factory tsgo.Factory,
	name string,
) tsgo.ClassDeclaration {
	return factory.ClassDeclaration(
		[]tsgo.ModifierLike{
			factory.ExportKeyword(),
			factory.AbstractKeyword(),
		},
		factory.Identifier(name),
		nil,
		nil,
		[]tsgo.ClassElement{
			factory.PropertyDeclaration(
				[]tsgo.ModifierLike{
					factory.AbstractKeyword(),
					factory.ReadonlyKeyword(),
				},
				factory.Identifier(DynamicTypeMember),
				nil,
				factory.KeywordTypeNode(
					tsgo.KeywordTypeSyntaxKindObjectKeyword,
				),
				nil,
			),
			factory.PropertyDeclaration(
				[]tsgo.ModifierLike{
					factory.AbstractKeyword(),
					factory.ReadonlyKeyword(),
				},
				factory.Identifier(MethodsMember),
				nil,
				readonlySetType(factory),
				nil,
			),
			factory.MethodDeclaration(
				[]tsgo.ModifierLike{factory.AbstractKeyword()},
				nil,
				factory.Identifier(ImplementsMember),
				nil,
				nil,
				[]tsgo.ParameterDeclaration{
					parameter(
						factory,
						"contract",
						readonlyObjectArray(factory),
					),
				},
				factory.KeywordTypeNode(
					tsgo.KeywordTypeSyntaxKindBooleanKeyword,
				),
				nil,
			),
			factory.MethodDeclaration(
				[]tsgo.ModifierLike{factory.AbstractKeyword()},
				nil,
				factory.Identifier(EqualMember),
				nil,
				nil,
				[]tsgo.ParameterDeclaration{
					parameter(
						factory,
						"other",
						factory.TypeReferenceNode(
							factory.Identifier(name),
							nil,
						),
					),
				},
				factory.KeywordTypeNode(
					tsgo.KeywordTypeSyntaxKindBooleanKeyword,
				),
				nil,
			),
			factory.MethodDeclaration(
				[]tsgo.ModifierLike{factory.AbstractKeyword()},
				nil,
				factory.Identifier(HashMember),
				nil,
				nil,
				nil,
				factory.KeywordTypeNode(
					tsgo.KeywordTypeSyntaxKindNumberKeyword,
				),
				nil,
			),
		},
	)
}

func nonNil(
	factory tsgo.Factory,
	valueName string,
	panicName string,
) tsgo.FunctionDeclaration {
	typeName := factory.TypeReferenceNode(factory.Identifier("T"), nil)
	value := factory.Identifier("value")
	return factory.FunctionDeclaration(
		[]tsgo.ModifierLike{factory.ExportKeyword()},
		nil,
		factory.Identifier("goInterfaceNonNil"),
		[]tsgo.TypeParameterDeclaration{
			factory.TypeParameterDeclaration(
				nil,
				factory.Identifier("T"),
				factory.TypeReferenceNode(
					factory.Identifier(valueName),
					nil,
				),
				nil,
				nil,
			),
		},
		[]tsgo.ParameterDeclaration{
			parameter(
				factory,
				"value",
				factory.UnionTypeNode([]tsgo.TypeNode{
					typeName,
					factory.KeywordTypeNode(
						tsgo.KeywordTypeSyntaxKindUndefinedKeyword,
					),
				}),
			),
		},
		typeName,
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
							factory.ExpressionStatement(
								panicruntime.Call(
									factory,
									panicName,
									factory.StringLiteral(
										"runtime error: invalid memory address or nil pointer dereference",
										tsgo.TokenFlagsNone,
									),
								),
							),
						},
						true,
					),
					nil,
				),
				factory.ReturnStatement(value),
			},
			true,
		),
	)
}

func readonlySetType(factory tsgo.Factory) tsgo.TypeNode {
	return factory.TypeReferenceNode(
		factory.Identifier("ReadonlySet"),
		[]tsgo.TypeNode{
			factory.KeywordTypeNode(
				tsgo.KeywordTypeSyntaxKindObjectKeyword,
			),
		},
	)
}

func readonlyObjectArray(factory tsgo.Factory) tsgo.TypeNode {
	return factory.TypeOperatorNode(
		tsgo.TypeOperatorNodeOperatorKindReadonlyKeyword,
		factory.ArrayTypeNode(
			factory.KeywordTypeNode(
				tsgo.KeywordTypeSyntaxKindObjectKeyword,
			),
		),
	)
}

func parameter(
	factory tsgo.Factory,
	name string,
	typeNode tsgo.TypeNode,
) tsgo.ParameterDeclaration {
	return factory.ParameterDeclaration(
		nil,
		nil,
		factory.Identifier(name),
		nil,
		typeNode,
		nil,
	)
}
