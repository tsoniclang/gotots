package panicnil

import (
	"github.com/tsoniclang/gotots/internal/emit/api"
	interfacecontract "github.com/tsoniclang/gotots/internal/emit/runtime/interfacevalue/contract"
	pointerruntime "github.com/tsoniclang/gotots/internal/emit/runtime/pointer"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

const (
	CreateName = "create"
	GuardName  = "$is"
)

func Build(
	factory tsgo.Factory,
	symbol api.RuntimeSymbol,
	errorName string,
	valueName string,
	runtimeValueName string,
	interfaceValueName string,
	pointerName string,
) (tsgo.Statement, error) {
	switch symbol {
	case api.RuntimePanicNilError:
		return panicNilError(factory, errorName), nil
	case api.RuntimePanicNilValue:
		return panicNilValue(
			factory,
			valueName,
			errorName,
			runtimeValueName,
			interfaceValueName,
			pointerName,
		), nil
	default:
		return nil, &api.RuntimeSymbolError{Symbol: symbol}
	}
}

func Create(
	factory tsgo.Factory,
	className string,
) tsgo.CallExpression {
	return factory.CallExpression(
		factory.PropertyAccessExpression(
			factory.Identifier(className),
			nil,
			factory.Identifier(CreateName),
			tsgo.NodeFlagsNone,
		),
		nil,
		nil,
		nil,
		tsgo.NodeFlagsNone,
	)
}

func panicNilError(
	factory tsgo.Factory,
	className string,
) tsgo.ClassDeclaration {
	return factory.ClassDeclaration(
		[]tsgo.ModifierLike{factory.ExportKeyword()},
		factory.Identifier(className),
		nil,
		nil,
		[]tsgo.ClassElement{
			factory.PropertyDeclaration(
				[]tsgo.ModifierLike{
					factory.DeclareKeyword(),
					factory.PrivateKeyword(),
					factory.ReadonlyKeyword(),
				},
				factory.Identifier("$go$panicNil"),
				nil,
				factory.KeywordTypeNode(
					tsgo.KeywordTypeSyntaxKindUndefinedKeyword,
				),
				nil,
			),
		},
	)
}

func panicNilValue(
	factory tsgo.Factory,
	className string,
	errorName string,
	runtimeValueName string,
	interfaceValueName string,
	pointerName string,
) tsgo.ClassDeclaration {
	errorType := factory.TypeReferenceNode(factory.Identifier(errorName), nil)
	pointerType := factory.TypeReferenceNode(
		factory.Identifier(pointerName),
		[]tsgo.TypeNode{errorType, errorType},
	)
	return factory.ClassDeclaration(
		[]tsgo.ModifierLike{factory.ExportKeyword()},
		factory.Identifier(className),
		nil,
		[]tsgo.HeritageClause{
			factory.HeritageClause(
				tsgo.HeritageClauseTokenKindExtendsKeyword,
				[]tsgo.ExpressionWithTypeArguments{
					factory.ExpressionWithTypeArguments(
						factory.Identifier(runtimeValueName),
						nil,
					),
				},
			),
		},
		[]tsgo.ClassElement{
			panicNilConstructor(factory, pointerType),
			factory.PropertyDeclaration(
				[]tsgo.ModifierLike{
					factory.OverrideKeyword(),
					factory.ReadonlyKeyword(),
				},
				factory.Identifier(interfacecontract.DynamicTypeMember),
				nil,
				factory.KeywordTypeNode(
					tsgo.KeywordTypeSyntaxKindObjectKeyword,
				),
				factory.Identifier(errorName),
			),
			panicNilCreate(
				factory,
				className,
				errorName,
				pointerName,
				errorType,
			),
			panicNilGuard(
				factory,
				className,
				errorName,
				interfaceValueName,
			),
		},
	)
}

func panicNilConstructor(
	factory tsgo.Factory,
	pointerType tsgo.TypeNode,
) tsgo.ConstructorDeclaration {
	return factory.ConstructorDeclaration(
		[]tsgo.ModifierLike{factory.PrivateKeyword()},
		nil,
		[]tsgo.ParameterDeclaration{
			factory.ParameterDeclaration(
				[]tsgo.ModifierLike{
					factory.PublicKeyword(),
					factory.ReadonlyKeyword(),
				},
				nil,
				factory.Identifier(interfacecontract.PayloadMember),
				nil,
				pointerType,
				nil,
			),
		},
		nil,
		factory.Block(
			[]tsgo.Statement{
				factory.ExpressionStatement(
					factory.CallExpression(
						factory.SuperExpression(),
						nil,
						nil,
						[]tsgo.Expression{
							factory.StringLiteral(
								"panic called with nil argument",
								tsgo.TokenFlagsNone,
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

func panicNilCreate(
	factory tsgo.Factory,
	className string,
	errorName string,
	pointerName string,
	errorType tsgo.TypeNode,
) tsgo.MethodDeclaration {
	payload := pointerruntime.Cell(
		factory,
		pointerName,
		errorType,
		errorType,
		factory.NewExpression(
			factory.Identifier(errorName),
			nil,
			nil,
		),
	)
	return factory.MethodDeclaration(
		[]tsgo.ModifierLike{factory.StaticKeyword()},
		nil,
		factory.Identifier(CreateName),
		nil,
		nil,
		nil,
		factory.TypeReferenceNode(factory.Identifier(className), nil),
		factory.Block(
			[]tsgo.Statement{
				factory.ReturnStatement(
					factory.NewExpression(
						factory.Identifier(className),
						nil,
						[]tsgo.Expression{payload},
					),
				),
			},
			true,
		),
	)
}

func panicNilGuard(
	factory tsgo.Factory,
	className string,
	errorName string,
	interfaceValueName string,
) tsgo.MethodDeclaration {
	value := factory.Identifier("value")
	return factory.MethodDeclaration(
		[]tsgo.ModifierLike{factory.StaticKeyword()},
		nil,
		factory.Identifier(GuardName),
		nil,
		nil,
		[]tsgo.ParameterDeclaration{
			factory.ParameterDeclaration(
				nil,
				nil,
				value,
				nil,
				factory.UnionTypeNode([]tsgo.TypeNode{
					factory.TypeReferenceNode(
						factory.Identifier(interfaceValueName),
						nil,
					),
					factory.KeywordTypeNode(
						tsgo.KeywordTypeSyntaxKindUndefinedKeyword,
					),
				}),
				nil,
			),
		},
		factory.TypePredicateNode(
			nil,
			value,
			factory.TypeReferenceNode(factory.Identifier(className), nil),
		),
		factory.Block(
			[]tsgo.Statement{
				factory.ReturnStatement(
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
								factory.Identifier(
									interfacecontract.DynamicTypeMember,
								),
								tsgo.NodeFlagsNone,
							),
							nil,
							factory.BinaryOperatorToken(
								tsgo.BinaryOperatorEqualsEqualsEqualsToken,
							),
							factory.Identifier(errorName),
						),
					),
				),
			},
			true,
		),
	)
}
